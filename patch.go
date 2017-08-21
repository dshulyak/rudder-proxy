package main

import (
	"encoding/json"
	"fmt"

	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	cfg := "/home/ds/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", cfg)
	must(err)
	client, err := kubernetes.NewForConfig(config)
	must(err)
	dep, err := client.AppsV1beta1().Deployments("kube-system").Get("tiller-deploy", v1.GetOptions{})
	must(err)
	orig, err := json.Marshal(dep)
	must(err)
	mod, err := json.Marshal(makeNew(*dep))
	must(err)
	patch, err := strategicpatch.CreateTwoWayMergePatch(orig, mod, v1beta1.Deployment{})
	must(err)
	fmt.Println(string(patch))
}

func makeNew(new v1beta1.Deployment) *v1beta1.Deployment {
	newContainers := []corev1.Container{
		{Name: "rudder", Image: "yashulyak/rudder", Command: []string{"/rudder", "-l", "0.0.0.0:10002"}},
		{Name: "istio-rudder-proxy", Image: "yashulyak/istio-rudder-proxy",
			Args: []string{"-l", "0.0.0.0:10001", "-s", "0.0.0.0:10002", "--tag", "0.2.0"}},
	}
	new.Spec.Template.Spec.Containers[0].Command = []string{"/tiller", "--experimental-release"}
	new.Spec.Template.Spec.Containers = append(new.Spec.Template.Spec.Containers, newContainers...)
	return &new
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
