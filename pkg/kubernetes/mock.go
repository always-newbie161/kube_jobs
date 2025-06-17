package kubernetes

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

func NewFakeKubernetesClient() *fake.Clientset {
	f := fake.NewSimpleClientset()

	f.Fake.PrependReactor("create", "jobs", func(action testing.Action) (bool, runtime.Object, error) {
		fmt.Println("simulating 10 sec job creation delay")
		time.Sleep(10 * time.Second)

		// Let the default create proceed
		createAction := action.(testing.CreateAction)
		obj := createAction.GetObject()
		return false, obj, nil
	})
	return f
}
