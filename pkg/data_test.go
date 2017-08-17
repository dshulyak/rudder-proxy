package proxy

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
)

func TestLoadDataOnce(t *testing.T) {
	container := v1.Container{
		Name: "load-data-test",
	}
	initContainers := []v1.Container{
		{Name: "first.init.container"},
		{Name: "second.init.container"},
	}
	bytes, err := json.Marshal(initContainers)
	require.NoError(t, err)
	annotations := map[string]string{
		"first.annotations.test":              "11",
		"second.annotations.test":             "22",
		v1.PodInitContainersBetaAnnotationKey: string(bytes),
	}
	containerBytes, err := yaml.Marshal(container)
	require.NoError(t, err)
	annotationsBytes, err := yaml.Marshal(annotations)
	require.NoError(t, err)
	tmpdir, err := ioutil.TempDir("/tmp", "rudder-proxy-unit-tests-XXX")
	require.NoError(t, err, "temporary directory should be created")
	defer func() { os.Remove(tmpdir) }()
	containerFilePath := filepath.Join(tmpdir, "container")
	require.NoError(t, ioutil.WriteFile(containerFilePath, containerBytes, 0644))
	annotationsFilePath := filepath.Join(tmpdir, "annotations")
	require.NoError(t, ioutil.WriteFile(annotationsFilePath, annotationsBytes, 0644))
	manager := NewMetaManager()
	manager.LoadDataOnce(containerFilePath, annotationsFilePath)
	assert.Equal(t, container.Name, manager.istioContainer.Name)
	assert.Equal(t, len(initContainers), len(manager.istioInitContainers))
	for i, c := range initContainers {
		assert.Equal(t, c.Name, manager.istioInitContainers[i].Name)
	}
	delete(annotations, v1.PodInitContainersBetaAnnotationKey)
	assert.Equal(t, annotations, manager.istioAnnotations)
}
