package proxy

import (
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	api "k8s.io/helm/pkg/proto/hapi/rudder"
)

func NewProxy(rudderURL, istioContainerPath, istioInitPath string) (*grpc.Server, error) {
	conn, err := grpc.Dial(rudderURL,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(3*time.Second))
	if err != nil {
		return nil, err
	}
	rProxy := &rudderProxy{client: api.NewReleaseModuleServiceClient(conn)}
	if err := LoadDataOnce(rProxy, istioContainerPath, istioInitPath); err != nil {
		conn.Close()
		return nil, err
	}
	server := grpc.NewServer()
	api.RegisterReleaseModuleServiceServer(server, rProxy)
	return server, nil
}

type rudderProxy struct {
	client             api.ReleaseModuleServiceClient
	dataSync           sync.RWMutex
	istioContainerData string
	istioInitData      string
}

func (r *rudderProxy) Version(ctx context.Context, in *api.VersionReleaseRequest) (*api.VersionReleaseResponse, error) {
	return r.client.Version(ctx, in)
}

func (r *rudderProxy) InstallRelease(ctx context.Context, in *api.InstallReleaseRequest) (*api.InstallReleaseResponse, error) {
	return r.client.InstallRelease(ctx, in)
}

// DeleteRelease requests deletion of a named release.
func (r *rudderProxy) DeleteRelease(ctx context.Context, in *api.DeleteReleaseRequest) (*api.DeleteReleaseResponse, error) {
	return r.client.DeleteRelease(ctx, in)
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
