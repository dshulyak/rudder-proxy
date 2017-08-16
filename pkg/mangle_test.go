package proxy

import (
	"bytes"
	"fmt"
	"testing"

	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var podYaml = `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - command: ["/bin/echo", "test"]
    image: gcr.io/google_containers/busybox
    name: test-container
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 2
  template:
    metadata:
      labels:
        app: nginx
    spec:
      nodeName: ctl01
      containers:
      - name: sleeper
        image: kubernetes/pause
---
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  name: test-petset
spec:
  serviceName: "petset"
  replicas: 2
  template:
    metadata:
      labels:
        app: nginx-petset
      annotations:
        pod.alpha.kubernetes.io/initialized: "true"
    spec:
      terminationGracePeriodSeconds: 0
      containers:
      - command: ["/bin/sh"]
        args:
        - -c
        - sleep 3; echo ok > /tmp/health; sleep 600
        image: gcr.io/google_containers/busybox
        readinessProbe:
          exec:
            command:
            - /bin/cat
            - /tmp/health
        name: test-container
      - command: ["/bin/sh"]
        args:
        - -c
        - sleep 3; echo ok > /tmp/health; sleep 600
        image: gcr.io/google_containers/busybox
        readinessProbe:
          exec:
            command:
            - /bin/cat
            - /tmp/health
        name: test-container2
`

func TestMangleList(t *testing.T) {
	m := map[string]interface{}{}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(podYaml)), 10)
	decoder.Decode(&m)
	converter := unstructured.NewConverter(false)
	p := v1.Pod{}
	if err := converter.FromUnstructured(m, &p); err != nil {
		t.Error(err)
	}
	if p.Name != "test-pod" {
		t.Errorf("unexpected parsing results: %v", p)
	}
	for key := range m {
		m[key] = nil
	}
	decoder.Decode(&m)
	dep := v1beta1.Deployment{}
	if err := converter.FromUnstructured(m, &dep); err != nil {
		t.Error(err)
	}
	if dep.Name != "nginx-deployment" {
		t.Errorf("unexpected parser results: %v", dep)
	}
	decoder.Decode(&m)
	statefulSet := v1beta1.StatefulSet{}
	if err := converter.FromUnstructured(m, &statefulSet); err != nil {
		t.Error(err)
	}
	if statefulSet.Name != "test-petset" {
		t.Errorf("unexpected parser results: %v", dep)
	}
	p1 := v1.Pod{}
	fmt.Println(decoder.Decode(&p1))
}
