package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/ghodss/yaml"
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
	newManifest := bytes.NewBuffer(make([]byte, 0, len([]byte(in.Release.Manifest))))
	originalManifest := bytes.NewReader([]byte(in.Release.Manifest))
	proxyManifest := bytes.NewBuffer(make([]byte, 0, len([]byte(in.Release.Manifest))))
	inject.IntoResourceFile(r.params, bytes.NewReader(proxyManifest.Bytes()), newManifest)
	skipWithAnnotation(r.annotation, originalManifest, newManifest, proxyManifest)
	in.Release.Manifest = newManifest.String()
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
		if addMeta(annotation, raw) {
			if _, err := proxy.Write(raw); err != nil {
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

func addMeta(annotation string, object []byte) bool {
	data := map[string]interface{}{}
	yaml.Unmarshal(object, &data)
	path := []string{"metadata", "annotations"}
	for _, p := range path {
		if level, exist := data[p]; !exist {
			return true
		} else {
			data = level.(map[string]interface{})
		}
	}
	_, exist := data[annotation]
	return !exist
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
