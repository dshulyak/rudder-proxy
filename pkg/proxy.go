package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"istio.io/pilot/platform/kube/inject"
	yamlDecoder "k8s.io/apimachinery/pkg/util/yaml"
	api "k8s.io/helm/pkg/proto/hapi/rudder"
)

func NewProxy(rudderURL, annotation string, params *inject.Params) (*grpc.Server, error) {
	conn, err := grpc.Dial(
		rudderURL,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(3*time.Second))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	rProxy := &rudderProxy{
		client:     api.NewReleaseModuleServiceClient(conn),
		params:     params,
		annotation: annotation,
	}
	server := grpc.NewServer()
	api.RegisterReleaseModuleServiceServer(server, rProxy)
	return server, nil
}

type rudderProxy struct {
	// it is safe to use single connection from multiple threads and in case of failures grpc
	// will reestablish connection itself
	client     api.ReleaseModuleServiceClient
	params     *inject.Params
	annotation string
}

func (r *rudderProxy) Version(ctx context.Context, in *api.VersionReleaseRequest) (*api.VersionReleaseResponse, error) {
	return r.client.Version(ctx, in)
}

// DeleteRelease requests deletion of a named release.
func (r *rudderProxy) DeleteRelease(ctx context.Context, in *api.DeleteReleaseRequest) (*api.DeleteReleaseResponse, error) {
	return r.client.DeleteRelease(ctx, in)
}

func (r *rudderProxy) InstallRelease(ctx context.Context, in *api.InstallReleaseRequest) (*api.InstallReleaseResponse, error) {
	log.Debug("Received manifest", in.Release.Manifest)
	newManifest := bytes.NewBuffer(make([]byte, 0, len([]byte(in.Release.Manifest))))
	originalManifest := bytes.NewReader([]byte(in.Release.Manifest))
	proxyManifest := bytes.NewBuffer(make([]byte, 0, len([]byte(in.Release.Manifest))))
	if err := skipWithAnnotation(r.annotation, originalManifest, newManifest, proxyManifest); err != nil {
		log.Errorf("error filtering objects with annotation %s: %v", r.annotation, err)
		return nil, err
	}
	if err := inject.IntoResourceFile(r.params, bytes.NewReader(proxyManifest.Bytes()), newManifest); err != nil {
		log.Errorf("error injecting istio proxy %v", err)
		return nil, err
	}
	in.Release.Manifest = newManifest.String()
	log.Debug("Result manifest", in.Release.Manifest)
	return r.client.InstallRelease(ctx, in)
}

func skipWithAnnotation(annotation string, original io.Reader, new, proxy io.Writer) error {
	reader := yamlDecoder.NewYAMLReader(bufio.NewReaderSize(original, 4096))
	for {
		raw, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		added, err := annotationDoesNotExist(annotation, raw)
		if err != nil {
			return err
		}

		if added {
			if _, err := proxy.Write(raw); err != nil {
				return err
			}
			if _, err = fmt.Fprint(proxy, "---\n"); err != nil {
				return err
			}
		} else {
			if _, err = new.Write(raw); err != nil {
				return err
			}
			if _, err = fmt.Fprint(new, "---\n"); err != nil {
				return err
			}
		}
	}
}

func annotationDoesNotExist(annotation string, object []byte) (bool, error) {
	data := map[string]interface{}{}
	if err := yaml.Unmarshal(object, &data); err != nil {
		return false, err
	}
	path := []string{"metadata", "annotations"}
	for _, p := range path {
		if level, exist := data[p]; !exist {
			return true, nil
		} else {
			data = level.(map[string]interface{})
		}
	}
	_, exist := data[annotation]
	return !exist, nil
}

// RollbackRelease rolls back a release to a previous version.
func (r *rudderProxy) RollbackRelease(ctx context.Context, in *api.RollbackReleaseRequest) (*api.RollbackReleaseResponse, error) {
	return r.client.RollbackRelease(ctx, in)
}

// UpgradeRelease updates release content.
func (r *rudderProxy) UpgradeRelease(ctx context.Context, in *api.UpgradeReleaseRequest) (*api.UpgradeReleaseResponse, error) {
	return r.client.UpgradeRelease(ctx, in)
}

// ReleaseStatus retrieves release status.
func (r *rudderProxy) ReleaseStatus(ctx context.Context, in *api.ReleaseStatusRequest) (*api.ReleaseStatusResponse, error) {
	return r.client.ReleaseStatus(ctx, in)
}
