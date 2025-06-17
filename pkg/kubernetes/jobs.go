package kubernetes

import (
	"context"
	"fmt"
	"time"

	"kubejobs/pkg/models"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	jobCreationTimeout = 30 * time.Second // seconds
)

type JobManager struct {
	clientset kubernetes.Interface
}

func NewJobManager(clientset kubernetes.Interface) *JobManager {
	return &JobManager{clientset: clientset}
}

// CreateKubernetesJob creates a Kubernetes job from a JobSpec
func (jm *JobManager) CreateKubernetesJob(ctx context.Context, jobSpec models.JobSpec) error {
	namespace := jobSpec.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Define the Kubernetes job [TODO: get the whole job spec from request only]
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobSpec.Name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: jobSpec.Name,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    jobSpec.Name,
							Image:   jobSpec.Image,
							Command: []string{jobSpec.Command},
							Args:    jobSpec.Args,
						},
					},
				},
			},
		},
	}

	createCtx, cancel := context.WithTimeout(ctx, jobCreationTimeout)
	defer cancel()

	return jm.CreateJob(createCtx, namespace, job)
}

// CreateJob creates a Kubernetes job with the given specification
func (jm *JobManager) CreateJob(ctx context.Context, namespace string, job *batchv1.Job) error {
	_, err := jm.clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return fmt.Errorf("job already exists: %v", err)
		}
		return fmt.Errorf("failed to create job: %v", err)
	}
	return nil
}
