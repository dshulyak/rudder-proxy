package proxy

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// LoadDataOnce reads istio container and annotation configs
func (m *MetaManager) LoadDataOnce(istioContainerPath, istioInitPath string) error {
	m.dataSync.Lock()
	defer m.dataSync.Unlock()
	containerData, err := ioutil.ReadFile(istioContainerPath)
	if err != nil {
		return err
	}
	initData, err := ioutil.ReadFile(istioInitPath)
	if err != nil {
		return err
	}
	if len(containerData) == 0 {
		return fmt.Errorf("Container definition can't be empty, source file: %s", istioContainerPath)
	}
	d := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(containerData), 10)
	container := v1.Container{}
	if err := d.Decode(&container); err != nil {
		return err
	}
	annotations := map[string]string{}
	d = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(initData), 10)
	if err := d.Decode(&annotations); err != nil {
		return err
	}
	if initContainers, isSet := annotations[v1.PodInitContainersBetaAnnotationKey]; isSet {
		delete(annotations, v1.PodInitContainersBetaAnnotationKey)
		containers := make([]v1.Container, 0, 2)
		d = yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(initContainers)), 10)
		if err := d.Decode(&containers); err != nil {
			return err
		}
		m.istioInitContainers = containers
	}
	m.istioContainer = container
	m.istioAnnotations = annotations
	return nil
}

// LoadDataPeriodically
func LoadDataPeriodically(r *rudderProxy, istioContainerPath, istioInitPath string) error {
	// TODO use inotify with golang fsnotify library to monitor changes for provided files
	// https://github.com/fsnotify/fsnotify
	return nil
}
