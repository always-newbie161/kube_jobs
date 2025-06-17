package api

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"kubejobs/pkg/api/handlers"
	"kubejobs/pkg/kubernetes"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type Server struct {
	router     *mux.Router
	httpServer *http.Server
	jobHandler *handlers.JobHandler
}

const (
	gracefulShutdownTimeout = 15 * time.Second // Timeout for graceful shutdown
)

// NewServer creates a new API server for the job scheduler
func NewServer(kubeconfig string, port string, maxConcurrency int, dryRun bool) (*Server, error) {

	var jobManager *kubernetes.JobManager

	if dryRun {

		fakeClient := kubernetes.NewFakeKubernetesClient()
		log.Info().Msg("Running in dry run mode, using fake Kubernetes client")
		jobManager = kubernetes.NewJobManager(fakeClient)

	} else {
		// Initialize Kubernetes client
		kubeClient, err := kubernetes.NewKubernetesClient(kubeconfig)
		if err != nil {
			return nil, err
		}
		log.Info().Msg("Running in production mode, using kube clientset created using kubeconfig")
		jobManager = kubeClient.GetJobManager()
	}

	// Initialize job handler to
	jobProcessor := handlers.JobProcessor(maxConcurrency, jobManager)

	// Create router
	router := mux.NewRouter()

	// Create server
	s := &Server{
		router:     router,
		jobHandler: jobProcessor,
		httpServer: &http.Server{
			Addr:         ":" + port,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	// Set up routes
	s.setupRoutes()

	// Start job handler
	jobProcessor.Start(context.Background())

	return s, nil
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", handlers.HealthCheckHandler).Methods("GET")

	// Worker endpoints
	s.router.HandleFunc("/jobs", s.jobHandler.JobSubmissionHandler).Methods("POST")
	s.router.HandleFunc("/jobs/pending", s.jobHandler.PendingJobsHandler).Methods("GET")
	s.router.HandleFunc("/jobs/running", s.jobHandler.RunningJobsHandler).Methods("GET")
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Set up graceful shutdown
	go s.gracefulShutdown()

	log.Printf("Server starting on %s", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// gracefulShutdown handles graceful server shutdown
func (s *Server) gracefulShutdown() {
	// Create channel for OS signals
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt)

	<-stopCh
	log.Info().Msg("Received interrupt signal, shutting down server...")

	// Create a deadline context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	// gracefully shutdown the http server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Info().Msgf("HTTP server shutdown failed: %v", err)
	} else {
		log.Info().Msg("HTTP server gracefully stopped")
	}

	// shutdown the job processor
	// This will wait for all in-flight jobs to complete
	s.jobHandler.Shutdown()
	log.Info().Msg("Job processor gracefully stopped")

}
