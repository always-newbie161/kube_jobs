package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"kubejobs/pkg/api"

	"k8s.io/client-go/util/homedir"
)

func main() {
	// Default kubeconfig path with home directory expansion
	var defaultKubeconfig string
	if home := homedir.HomeDir(); home != "" {
		defaultKubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		defaultKubeconfig = ""
	}

	// Command line flags
	kubeconfig := flag.String("kubeconfig", defaultKubeconfig, "Path to the kubeconfig file")
	port := flag.String("port", "8080", "Port to run the HTTP server on")
	maxConcurrency := flag.Int("maxconcurrency", 10, "Maximum number of concurrent jobs")
	dryRun := flag.Bool("dryrun", false, "Enable dry run mode (uses fake kube client)")

	flag.Parse()

	// Create server
	server, err := api.NewServer(*kubeconfig, *port, *maxConcurrency, *dryRun)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	log.Printf("Starting server on port %s...\n", *port)
	log.Printf("Using kubeconfig: %s\n", *kubeconfig)
	log.Printf("Max job concurrency: %d\n", *maxConcurrency)

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
		os.Exit(1)
	}
}
