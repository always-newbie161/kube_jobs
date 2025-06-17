package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type KubernetesClient struct {
	clientset kubernetes.Interface
}

// NewKubernetesClient creates a new Kubernetes client
// If kubeconfig is "~/.kube/config", it expands the home directory
func NewKubernetesClient(kubeconfig string) (*KubernetesClient, error) {
	var config *rest.Config
	var err error

	// Handle special case for home directory
	if kubeconfig == "~/.kube/config" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			return nil, fmt.Errorf("could not locate home directory for kubeconfig")
		}
	}

	// Check if kubeconfig file exists
	if kubeconfig != "" {
		if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
			return nil, fmt.Errorf("kubeconfig file not found at %s", kubeconfig)
		}
	} else {
		return nil, fmt.Errorf("kubeconfig must be provided")
	}

	// Use provided kubeconfig
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig %s: %v", kubeconfig, err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	// Test connection
	_, err = clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{Limit: 1})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Kubernetes API: %v", err)
	}

	return &KubernetesClient{clientset: clientset}, nil
}

// GetJobManager returns a new JobManager using this client's clientset
func (kc *KubernetesClient) GetJobManager() *JobManager {
	return NewJobManager(kc.clientset)
}
