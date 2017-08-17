package proxy

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/ghodss/yaml"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion/unstructured"
	yamldecoder "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/helm/pkg/proto/hapi/release"
)

// NewMetaManager returns pointer to new MetaManager instance
func NewMetaManager() *MetaManager {
	return &MetaManager{
		converter: unstructured.NewConverter(false),
		reader:    ioutil.ReadFile,
	}
}

type fileReader func(filename string) ([]byte, error)

// MetaManager manages state of istio config files and mangles releases
type MetaManager struct {
	converter unstructured.Converter
	reader    fileReader

	dataSync            sync.RWMutex
	istioContainer      v1.Container
	istioInitContainers []v1.Container
	istioAnnotations    map[string]string
}

// MangleRelease will add istio side car for each pod and additional annotations
func (m *MetaManager) MangleRelease(r *release.Release) error {
	m.dataSync.RLock()
	defer m.dataSync.RUnlock()
	manifest, err := m.newManifest([]byte(r.Manifest))
	if err != nil {
		return err
	}
	r.Manifest = manifest
	return nil
}

func (m *MetaManager) newManifest(manifest []byte) (string, error) {
	writeTo := bytes.NewBuffer(make([]byte, 0, len(manifest)))
	d := yamldecoder.NewYAMLOrJSONDecoder(bytes.NewReader(manifest), 10)
	for {
		u, err := decodeSingle(d)
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		object, err := m.mangleSingle(u)
		if err != nil {
			return "", err
		}
		writeTo.WriteString(object)
		writeTo.WriteString("---\n")
	}
	return writeTo.String(), nil
}

func decodeSingle(d *yamldecoder.YAMLOrJSONDecoder) (map[string]interface{}, error) {
	into := map[string]interface{}{}
	if err := d.Decode(&into); err != nil {
		return nil, err
	}
	return into, nil
}

func (m *MetaManager) mangleSingle(u map[string]interface{}) (string, error) {
	var runtimeObj interface{}
	var podSpec *v1.PodSpec
	var objectMeta *metav1.ObjectMeta
	// is there better way to do it?
	switch strings.ToLower(u["kind"].(string)) {
	case "pod":
		obj := &v1.Pod{}
		if err := m.converter.FromUnstructured(u, obj); err != nil {
			return "", err
		}
		podSpec = &obj.Spec
		objectMeta = &obj.ObjectMeta
		runtimeObj = obj
	case "deployment":
		obj := &v1beta1.Deployment{}
		if err := m.converter.FromUnstructured(u, obj); err != nil {
			return "", err
		}
		podSpec = &obj.Spec.Template.Spec
		objectMeta = &obj.Spec.Template.ObjectMeta
		runtimeObj = obj
	case "statefulset":
		obj := &v1beta1.StatefulSet{}
		if err := m.converter.FromUnstructured(u, obj); err != nil {
			return "", err
		}

		podSpec = &obj.Spec.Template.Spec
		objectMeta = &obj.Spec.Template.ObjectMeta
		runtimeObj = obj
	case "daemonset":
		obj := &v1beta2.DaemonSet{}
		if err := m.converter.FromUnstructured(u, obj); err != nil {
			return "", err
		}
		podSpec = &obj.Spec.Template.Spec
		objectMeta = &obj.Spec.Template.ObjectMeta
		runtimeObj = obj
	case "replicaset":
		obj := &v1beta2.ReplicaSet{}
		if err := m.converter.FromUnstructured(u, obj); err != nil {
			return "", err
		}
		podSpec = &obj.Spec.Template.Spec
		objectMeta = &obj.Spec.Template.ObjectMeta
		runtimeObj = obj
	}
	if podSpec != nil {
		if err := m.manglePodSpecAndMeta(podSpec, objectMeta); err != nil {
			return "", err
		}
	} else {
		runtimeObj = u
	}
	serialized, err := yaml.Marshal(runtimeObj)
	if err != nil {
		return "", err
	}
	return string(serialized), nil
}

func (m *MetaManager) manglePodSpecAndMeta(ps *v1.PodSpec, objMeta *metav1.ObjectMeta) error {
	ps.Containers = append(ps.Containers, m.istioContainer)
	if objMeta.Annotations == nil {
		objMeta.Annotations = make(map[string]string)
	}
	var containers []v1.Container
	if initContainers, isSet := objMeta.Annotations[v1.PodInitContainersBetaAnnotationKey]; isSet {
		// 2 is a guess for a number of original init containers
		containers = make([]v1.Container, 0, 2+len(m.istioInitContainers))
		d := yamldecoder.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(initContainers)), 10)
		if err := d.Decode(&containers); err != nil {
			return err
		}
		containers = append(containers, m.istioInitContainers...)
	} else {
		containers = m.istioInitContainers
	}
	marshaled, err := yaml.Marshal(&containers)
	if err != nil {
		return err
	}
	objMeta.Annotations[v1.PodInitContainersBetaAnnotationKey] = string(marshaled)
	for key, val := range m.istioAnnotations {
		objMeta.Annotations[key] = val
	}
	return nil
}
