package proxy

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/helm/pkg/proto/hapi/release"
)

var testManifest = `
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
---
apiVersion: v1
kind: Secret
metadata:
  name: example-secret
type: Opaque
data:
  password: MWYyZDFlMmU2N2Rm
  username: YWRtaW4=
---
`

func TestMangleRelease(t *testing.T) {
	testRelease := &release.Release{Manifest: testManifest}
	manager := NewMetaManager()
	manager.istioAnnotations = map[string]string{
		"test.annotation":        "111",
		"test.second.annotation": "222",
	}
	manager.istioContainer = v1.Container{Name: "test.container"}
	manager.istioInitContainers = []v1.Container{{Name: "test.init.1"}, {Name: "test.init.2"}}

	require.NoError(t, manager.MangleRelease(testRelease), "mangle release shouldn't return error")
	// we can rely on order to be preserved
	d := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(testRelease.Manifest)), 10)
	pod := &v1.Pod{}
	require.NoError(t, d.Decode(pod))
	verifyPodSpecAndAnnotations(t, manager, pod.Spec, pod.ObjectMeta.Annotations)
	deployment := &v1beta1.Deployment{}
	require.NoError(t, d.Decode(deployment))
	verifyPodSpecAndAnnotations(t, manager, deployment.Spec.Template.Spec,
		deployment.Spec.Template.ObjectMeta.Annotations)
	statefulSet := &v1beta1.StatefulSet{}
	require.NoError(t, d.Decode(statefulSet))
	verifyPodSpecAndAnnotations(t, manager, statefulSet.Spec.Template.Spec,
		statefulSet.Spec.Template.ObjectMeta.Annotations)
	secret := v1.Secret{}
	require.NoError(t, d.Decode(&secret))
	assert.Equal(t, "example-secret", secret.Name)
}

func verifyPodSpecAndAnnotations(t *testing.T, m *MetaManager, podSpec v1.PodSpec, annotations map[string]string) {
	assert.Equal(t, m.istioContainer.Name, podSpec.Containers[len(podSpec.Containers)-1].Name)
	for key, val := range m.istioAnnotations {
		assert.Equal(t, val, annotations[key])
	}
	initContainers, isSet := annotations[v1.PodInitContainersBetaAnnotationKey]
	require.True(t, isSet, "init containers should be present")
	d := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(initContainers)), 10)
	containers := make([]v1.Container, 0, 2)
	assert.NoError(t, d.Decode(&containers))
	assert.Equal(t, len(m.istioInitContainers), len(containers))
	for i, c := range m.istioInitContainers {
		assert.Equal(t, c.Name, containers[i].Name)
	}

}
