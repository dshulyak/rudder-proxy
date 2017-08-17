package main

import (
	"flag"
	"net"
	"os"

	proxy "github.com/Mirantis/istio-rudder-proxy/pkg"
	"github.com/istio/pilot/platform/kube/inject"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type options struct {
	listen    string
	rudderURL string
	logLevel  string

	hub             string
	tag             string
	sidecarProxyUID int64
	verbosity       int
	versionStr      string // override build version
	enableCoreDump  bool
	meshConfig      string
	imagePullPolicy string
	includeIPRanges string

	kubeconfig string
}

func init() {
	log.SetOutput(os.Stderr)
}

func main() {
	opts := options{}
	flag.StringVar(&opts.rudderURL, "rudder", "http://localhost:8788",
		"URL that will be used for qcommunication with rudder.")
	flag.StringVar(&opts.logLevel, "log-level", "debug", "Logger level controls amount of details in logs.")
	flag.StringVar(&opts.listen, "--listen", "0.0.0.0:8989", "Listening socket.")
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
		ImagePullPolicy:   opts.imagePullPolicy,
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
