package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"kubejobs/internal/queue"
	"kubejobs/pkg/kubernetes"
	"kubejobs/pkg/models"

	"github.com/rs/zerolog/log"
	"github.com/twinj/uuid"
)

const (
	pollFrequency = 100 * time.Millisecond // frequency to poll the job queue
)

type JobHandler struct {
	jobQueue        *queue.ConcJobQueue
	jobManager      *kubernetes.JobManager
	concurrencyLock chan struct{} // manages the concurrency slots for job processing
	runningJobs     map[string]models.JobSpec
	pendingJobs     map[string]models.JobSpec
	shutdownCh      chan struct{}
	doneCh          chan struct{}
	wg              sync.WaitGroup // wait group to ensure all running jobs are completed before shutdown
	sync.RWMutex
}

func JobProcessor(maxConcurrency int, jobManager *kubernetes.JobManager) *JobHandler {
	return &JobHandler{
		jobQueue:        queue.NewConcJobQueue(),
		jobManager:      jobManager,
		concurrencyLock: make(chan struct{}, maxConcurrency),
		runningJobs:     make(map[string]models.JobSpec),
		pendingJobs:     make(map[string]models.JobSpec),
		shutdownCh:      make(chan struct{}),
		doneCh:          make(chan struct{}),
	}
}

// Start begins the job processing
func (h *JobHandler) Start(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Msgf("panic occurred in job processing: %v", r)
			}
			log.Info().Msg("Job processing stopped")
			close(h.doneCh) // signal that processing is done
		}()
		h.processJobs(ctx)
	}()
}

// processJobs continuously processes jobs from the queue
func (h *JobHandler) processJobs(ctx context.Context) {
	defer func() {
		// wait until all running jobs are completed
		h.wg.Wait()
		close(h.doneCh)
	}()
	for {
		select {
		case <-h.shutdownCh:
			// shutdown requested
			log.Info().Msg("shutdown requested, stopping job processing")
			return
		case <-ctx.Done():
			// context cancelled, exit processing
			log.Info().Msg("context cancelled, stopping job processing")
		default:
		}

		// Check if job queue is empty
		if h.jobQueue.IsEmpty() {
			// avoiding tight loop, when no jobs are available, this essentially polls every 100ms
			time.Sleep(pollFrequency)
			continue
		}

		// block until a concurrency slot is available
		select {
		case h.concurrencyLock <- struct{}{}:
			fmt.Println("Acquired concurrency slot, dequeueing job", cap(h.concurrencyLock))
		case <-h.shutdownCh:
			// Shutdown requested while waiting
			log.Info().Msg("shutdown requested while waiting for concurrency slot, stopping job processing")
			return
		case <-ctx.Done():
			// Context cancelled while waiting
			log.Info().Msg("context cancelled while waiting for concurrency slot, stopping job processing")
			return
		}

		// Get the next job from the queue
		job := h.jobQueue.Dequeue()
		if job == nil {
			// If no job is available, release the concurrency slot and try again
			<-h.concurrencyLock
			fmt.Println("freeing lock, no job available")
			log.Warn().Msg("empty job in the queue, releasing concurrency slot")
			continue
		}

		JobSpec, ok := job.Spec.(models.JobSpec)
		if !ok {
			// Invalid job spec, release the concurrency slot and continue
			<-h.concurrencyLock
			fmt.Println("freeing lock, invalid job spec type")
			log.Error().Msgf("invalid job spec type: %T, expected models.JobSpec", job.Spec)
			continue
		}

		// Remove from pending jobs
		h.Lock()
		delete(h.pendingJobs, JobSpec.Name)
		h.runningJobs[JobSpec.Name] = JobSpec
		h.Unlock()

		// Add to the wait sync group
		h.wg.Add(1)
		// Start a goroutine to process the job
		go func(JobSpec models.JobSpec) {
			// panic recoverer
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("panic occurred while processing job %s: %v", JobSpec.Name, r)
				}
			}()
			defer func() {
				// Release the concurrency slot when done
				<-h.concurrencyLock
				fmt.Println("Released concurrency slot after processing job", JobSpec.Name)

				// Remove from running jobs
				h.Lock()
				delete(h.runningJobs, JobSpec.Name)
				h.Unlock()

				// Mark the job as done in the wait group
				h.wg.Done()
			}()

			log.Info().Msgf("Processing job: %s with priority %d", JobSpec.Name, JobSpec.Priority)

			// Create and submit the Kubernetes job
			err := h.jobManager.CreateKubernetesJob(ctx, JobSpec)
			if err != nil {
				log.Error().Err(err).Msgf("failed to create Kubernetes job for %s", JobSpec.Name)
				return
			}
			log.Info().Msgf("Successfully submitted job: %s with ID: %s", JobSpec.Name, JobSpec.ID)
		}(JobSpec)
	}
}

// JobSubmissionHandler handles job submissions via the /jobs POST endpoint.
func (h *JobHandler) JobSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	var JobSpec models.JobSpec

	// Unmarshal the request body into JobSpec
	if err := json.NewDecoder(r.Body).Decode(&JobSpec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate job spec
	if JobSpec.Name == "" {
		http.Error(w, "Job name is required", http.StatusBadRequest)
		return
	}

	// Generate a unique ID for the job
	jobID := uuid.NewV1().String()
	JobSpec.ID = jobID

	// Add job to pending jobs
	h.Lock()
	h.pendingJobs[JobSpec.Name] = JobSpec
	h.Unlock()

	// Enqueue the job
	jobPosn := h.jobQueue.Enqueue(jobID, JobSpec.Priority, JobSpec)

	// Respond with a success message
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":         "Job submitted successfully",
		"name":           JobSpec.Name,
		"job_id":         jobID,
		"queue_position": strconv.Itoa(jobPosn),
	})
}

// PendingJobsHandler returns the list of pending jobs
func (h *JobHandler) PendingJobsHandler(w http.ResponseWriter, r *http.Request) {
	h.RLock()
	defer h.RUnlock()

	pendingJobsList := make([]models.JobResponse, 0, len(h.pendingJobs))
	for _, job := range h.pendingJobs {
		pendingJobsList = append(pendingJobsList, models.JobResponse{
			Name:     job.Name,
			JobID:    job.ID,
			Priority: job.Priority,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pendingJobsList)
}

// RunningJobsHandler returns the list of running jobs
func (h *JobHandler) RunningJobsHandler(w http.ResponseWriter, r *http.Request) {
	h.RLock()
	defer h.RUnlock()

	runningJobsList := make([]models.JobResponse, 0, len(h.runningJobs))
	for _, job := range h.runningJobs {
		runningJobsList = append(runningJobsList, models.JobResponse{
			Name:     job.Name,
			JobID:    job.ID,
			Priority: job.Priority,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runningJobsList)
}

// getQueuePosition returns the position of a job in the queue
func (h *JobHandler) getQueuePosition(name string) string {
	return "queued"
}

// Shutdown initiates a graceful shutdown of the job handler
func (h *JobHandler) Shutdown() {
	close(h.shutdownCh)
	<-h.doneCh // Wait for job processing to complete
}
