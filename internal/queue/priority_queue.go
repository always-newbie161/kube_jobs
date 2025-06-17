package queue

import (
	"container/heap"
	"sync"
)

// Job represents a job in the priority queue
type Job struct {
	Priority int
	Spec     interface{}
	ID       string // Unique identifier for the job
	Posn     int    // Position in the queue, used for tracking
}

// PriorityQueue implements a max-priority queue using container/heap
type PriorityQueue []*Job

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	// update the position of the jobs after swapping
	pq[i].Posn = i
	pq[j].Posn = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*Job))
	x.(*Job).Posn = len(*pq) - 1 // Set the position of the job in the queue
}

// Pop removes and returns the highest priority item
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	job := old[n-1]
	job.Posn = -1 // reset position since it's being popped out
	*pq = old[0 : n-1]
	return job
}

// Peek returns the highest priority item without removing it
func (pq PriorityQueue) Peek() *Job {
	if len(pq) == 0 {
		return nil
	}
	return pq[0]
}

// ConcJobQueue is a thread-safe priority queue for jobs
type ConcJobQueue struct {
	pq PriorityQueue
	mu sync.Mutex
}

// NewConcJobQueue creates a new job queue
func NewConcJobQueue() *ConcJobQueue {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)
	return &ConcJobQueue{
		pq: pq,
	}
}

// Enqueue adds a job to the queue with the specified priority, returning its position in the queue
func (jq *ConcJobQueue) Enqueue(id string, priority int, spec interface{}) int {
	job := &Job{Priority: priority, ID: id, Spec: spec}

	jq.mu.Lock()
	heap.Push(&jq.pq, job)
	jq.mu.Unlock()

	return job.Posn
}

// Dequeue removes and returns the highest priority job
func (jq *ConcJobQueue) Dequeue() *Job {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	if jq.pq.Len() == 0 {
		return nil
	}

	return heap.Pop(&jq.pq).(*Job)
}

// Peek returns the highest priority job without removing it
func (jq *ConcJobQueue) Peek() *Job {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	return jq.pq.Peek()
}

// Size returns the number of jobs in the queue
func (jq *ConcJobQueue) Size() int {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	return jq.pq.Len()
}

// IsEmpty returns true if the queue is empty
func (jq *ConcJobQueue) IsEmpty() bool {
	return jq.Size() == 0
}
