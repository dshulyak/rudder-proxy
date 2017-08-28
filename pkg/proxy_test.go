package proxy

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	proxyconfig "istio.io/api/proxy/v1/config"
	"istio.io/pilot/platform/kube/inject"
)

func TestAddMeta(t *testing.T) {
	for i, tc := range []struct {
		object   string
		expected bool
	}{
		{
			object: `
metadata:
  name: test-job
  annotations:
    istio.skip: 1
`,
			expected: false,
		},
		{
			object: `
metadata:
  name: test-job
`,
			expected: true,
		},
		{
			object:   "",
			expected: true,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if rst, _ := annotationDoesNotExist("istio.skip", []byte(tc.object)); rst != tc.expected {
				t.Errorf("result '%t' is different from expected '%t' for object %s", rst, tc.expected, tc.object)
			}
		})
	}
}

// TODO finish this test
func TestInjection(t *testing.T) {
	man := `
kind: Secret
metadata:
  name: test-secret
---

kind: Job
metadata:
  name: test-pod
  annotations:
    istio.skip: 1
---

apiVersion: batch/v1
kind: Job
metadata:
  name: test-job
spec:
  template:
    metadata:
      name: test-job
    spec:
      containers:
      - name: test-container
        image: gcr.io/google_containers/busybox
        command: [ "/bin/sh", "-c", "sleep 10; env"]
      restartPolicy: Never
---

`
	newManifest := bytes.NewBuffer(make([]byte, 0, len([]byte(man))))
	originalManifest := bytes.NewReader([]byte(man))
	proxyManifest := bytes.NewBuffer(make([]byte, 0, len([]byte(man))))
	if err := skipWithAnnotation("istio.skip", originalManifest, newManifest, proxyManifest); err != nil {
		t.Fatalf("error skiping object with annotation 'istio.skip': %v\n", err)
	}
	if err := inject.IntoResourceFile(&inject.Params{
		Mesh: &proxyconfig.ProxyMeshConfig{},
	}, bytes.NewReader(proxyManifest.Bytes()), newManifest); err != nil {
		t.Fatalf("error injecting istio proxy %v\n", err)
	}
	fmt.Println(proxyManifest.String())
	fmt.Println("##########################")
	fmt.Println(newManifest.String())
}
