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
	logLevel  string

	kubeconfig      string
	hub             string
	tag             string
	sidecarProxyUID int64
	verbosity       int
	versionStr      string // override build version
	enableCoreDump  bool
	meshConfig      string
	includeIPRanges string
	istioSystem     string
}

func (opts *options) registerFlags() {
	defaultKubeconfig := os.Getenv("HOME") + "/.kube/config"
	if v := os.Getenv("KUBECONFIG"); v != "" {
		defaultKubeconfig = v
	}
	pflag.StringVar(&opts.rudderURL, "rudder", "http://localhost:8788",
		"URL that will be used for qcommunication with rudder.")
	pflag.StringVar(&opts.logLevel, "log-level", "debug", "Logger level controls amount of details in logs.")
	pflag.StringVar(&opts.listen, "--listen", "0.0.0.0:8989", "Listening socket.")
	pflag.StringVar(&opts.hub, "hub", "docker.io/istio", "Docker hub")
	pflag.StringVar(&opts.tag, "tag", version.Info.Version, "Docker tag")
	pflag.IntVar(&opts.verbosity, "verbosity", inject.DefaultVerbosity, "Runtime verbosity")
	pflag.Int64Var(&opts.sidecarProxyUID, "sidecarProxyUID",
		inject.DefaultSidecarProxyUID, "Envoy sidecar UID")
	pflag.StringVar(&opts.versionStr, "setVersionString",
		"", "Override version info injected into resource")
	pflag.StringVar(&opts.meshConfig, "meshConfig", "istio",
		fmt.Sprintf("ConfigMap name for Istio mesh configuration, key should be %q", inject.ConfigMapKey))
	pflag.StringVarP(&opts.kubeconfig, "kubeconfig", "c", defaultKubeconfig,
		"Kubernetes configuration file")
	pflag.StringVarP(&opts.istioSystem, "namespace", "n", v1.NamespaceDefault,
		"Kubernetes Istio system namespace")
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
	level, err := log.ParseLevel(opts.logLevel)
	if err != nil {
		log.Panic(err)
	}
	log.SetLevel(level)
	log.Infof("Istio rudder proxy listening on %s with rudder URL %s\n", opts.listen, opts.rudderURL)
	config, err := clientcmd.BuildConfigFromFlags("", opts.kubeconfig)
	if err != nil {
		log.Panic(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err)
	}
	mesh, err := inject.GetMeshConfig(client, opts.istioSystem, opts.meshConfig)
	if err != nil {
		log.Panicf("Istio configuration not found. Verify istio configmap is "+
			"installed in namespace %q with `kubectl get -n %s configmap istio`\n",
			opts.istioSystem, opts.istioSystem)
	}
	params := &inject.Params{
		InitImage:         inject.InitImageName(opts.hub, opts.tag),
		ProxyImage:        inject.ProxyImageName(opts.hub, opts.tag),
		Verbosity:         opts.verbosity,
		SidecarProxyUID:   opts.sidecarProxyUID,
		Version:           opts.versionStr,
		EnableCoreDump:    opts.enableCoreDump,
		Mesh:              mesh,
		MeshConfigMapName: opts.meshConfig,
		IncludeIPRanges:   opts.includeIPRanges,
	}
	proxyServer, err := proxy.NewProxy(opts.rudderURL, params)
	if err != nil {
		log.Panic(err)
	}
	listenSocket, err := net.Listen("tcp", opts.listen)
	if err != nil {
		log.Panic(err)
	}
	defer proxyServer.GracefulStop()
	log.Info(proxyServer.Serve(listenSocket))
}
