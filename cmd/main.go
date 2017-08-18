package main

import (
	"fmt"
	"net"
	"os"

	proxy "github.com/Mirantis/istio-rudder-proxy/pkg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"istio.io/pilot/platform/kube/inject"
	"istio.io/pilot/tools/version"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type options struct {
	listen    string
	rudderURL string

	kubeconfig      string
	hub             string
	tag             string
	sidecarProxyUID int64
	verbosity       string
	versionStr      string // override build version
	enableCoreDump  bool
	meshConfig      string
	includeIPRanges string
	istioSystem     string
}

func (opts *options) registerFlags() {
	pflag.StringVarP(&opts.rudderURL, "rudder-socket", "s", "0.0.0.0:10002", "Rudderl socket")
	pflag.StringVarP(&opts.verbosity, "verbosity", "v", "debug", "Logger verbosity")
	pflag.StringVarP(&opts.listen, "listen", "l", "0.0.0.0:10001", "Listen on this socket")
	pflag.StringVar(&opts.hub, "hub", "docker.io/istio", "Docker hub")
	pflag.StringVar(&opts.tag, "tag", version.Info.Version, "Docker tag")
	pflag.Int64Var(&opts.sidecarProxyUID, "sidecarProxyUID", inject.DefaultSidecarProxyUID, "Envoy sidecar UID")
	pflag.StringVar(&opts.versionStr, "setVersionString", "", "Override version info injected into resource")
	pflag.StringVar(&opts.meshConfig, "meshConfig", "istio",
		fmt.Sprintf("ConfigMap name for Istio mesh configuration, key should be %q", inject.ConfigMapKey))
	pflag.StringVarP(&opts.kubeconfig, "kubeconfig", "c", "", "Kubernetes configuration file")
	pflag.StringVarP(&opts.istioSystem, "namespace", "n", v1.NamespaceDefault, "Kubernetes Istio system namespace")
}

func (opts *options) parseFlags() {
	pflag.Parse()
	if opts.versionStr == "" {
		opts.versionStr = version.Line()
	}

}

func init() {
	log.SetOutput(os.Stderr)
}

func main() {
	opts := new(options)
	opts.registerFlags()
	opts.parseFlags()
	level, err := log.ParseLevel(opts.verbosity)
	if err != nil {
		log.Error(err)
		log.Exit(1)
	}
	log.SetLevel(level)
	log.Infof("Istio rudder proxy listening on %s with rudder URL %s", opts.listen, opts.rudderURL)
	config, err := clientcmd.BuildConfigFromFlags("", opts.kubeconfig)
	if err != nil {
		log.Error(err)
		log.Exit(1)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err)
		log.Exit(1)
	}
	mesh, err := inject.GetMeshConfig(client, opts.istioSystem, opts.meshConfig)
	if err != nil {
		log.Errorf("Istio configuration not found. Verify istio configmap is "+
			"installed in namespace %q with `kubectl get -n %s configmap istio`",
			opts.istioSystem, opts.istioSystem)
		log.Exit(1)
	}
	params := &inject.Params{
		InitImage:         inject.InitImageName(opts.hub, opts.tag),
		ProxyImage:        inject.ProxyImageName(opts.hub, opts.tag),
		Verbosity:         int(level),
		SidecarProxyUID:   opts.sidecarProxyUID,
		Version:           opts.versionStr,
		EnableCoreDump:    opts.enableCoreDump,
		Mesh:              mesh,
		MeshConfigMapName: opts.meshConfig,
		IncludeIPRanges:   opts.includeIPRanges,
	}
	proxyServer, err := proxy.NewProxy(opts.rudderURL, params)
	if err != nil {
		log.Error(err)
		log.Exit(1)
	}
	listenSocket, err := net.Listen("tcp", opts.listen)
	if err != nil {
		log.Error(err)
		log.Exit(1)
	}
	defer proxyServer.GracefulStop()
	log.Info(proxyServer.Serve(listenSocket))
}
