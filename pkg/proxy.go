package proxy

import (
	"time"

	log "github.com/sirupsen/logrus"
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
	rProxy := &rudderProxy{
		client:      api.NewReleaseModuleServiceClient(conn),
		metaManager: NewMetaManager(),
	}
	if err := rProxy.metaManager.LoadDataOnce(istioContainerPath, istioInitPath); err != nil {
		if err := conn.Close(); err != nil {
			log.Errorf("error closing grpc connection: %v\n", err)
		}
		return nil, err
	}
	server := grpc.NewServer()
	api.RegisterReleaseModuleServiceServer(server, rProxy)
	return server, nil
}

type rudderProxy struct {
	// it is safe to use single connection from multiple threads and in case of failures grpc
	// will reestablish connection itself
	client      api.ReleaseModuleServiceClient
	metaManager *MetaManager
}

func (r *rudderProxy) Version(ctx context.Context, in *api.VersionReleaseRequest) (*api.VersionReleaseResponse, error) {
	return r.client.Version(ctx, in)
}

// DeleteRelease requests deletion of a named release.
func (r *rudderProxy) DeleteRelease(ctx context.Context, in *api.DeleteReleaseRequest) (*api.DeleteReleaseResponse, error) {
	return r.client.DeleteRelease(ctx, in)
}

func (r *rudderProxy) InstallRelease(ctx context.Context, in *api.InstallReleaseRequest) (*api.InstallReleaseResponse, error) {
	r.metaManager.MangleRelease(in.GetRelease())
	return r.client.InstallRelease(ctx, in)
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
