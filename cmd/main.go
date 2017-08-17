package main

import (
	"flag"
	"net"
	"os"

	proxy "github.com/Mirantis/istio-rudder-proxy/pkg"
	log "github.com/sirupsen/logrus"
)

type options struct {
	Listen             string
	RudderURL          string
	LogLevel           string
	IstioContainerPath string
	IstioInitPath      string
}

func init() {
	log.SetOutput(os.Stderr)
}

func main() {
	opts := options{}
	flag.StringVar(&opts.RudderURL, "rudder", "http://localhost:8788",
		"URL that will be used for qcommunication with rudder.")
	flag.StringVar(&opts.LogLevel, "log-level", "debug", "Logger level controls amount of details in logs.")
	flag.StringVar(&opts.IstioContainerPath, "istio-container-path", "",
		"Location of container istio side car container definition.")
	flag.StringVar(&opts.IstioInitPath, "istio-init-path", "", "Location of init container definition")
	flag.StringVar(&opts.Listen, "--listen", "0.0.0.0:8989", "Listening socket.")
	level, err := log.ParseLevel(opts.LogLevel)
	if err != nil {
		log.Panic(err)
	}
	log.SetLevel(level)
	log.Infof("Istio rudder proxy listening on %s with rudder URL %s\n", opts.Listen, opts.RudderURL)
	proxyServer, err := proxy.NewProxy(opts.RudderURL, opts.IstioContainerPath, opts.IstioInitPath)
	if err != nil {
		log.Panic(err)
	}
	listenSocket, err := net.Listen("tcp", opts.Listen)
	if err != nil {
		log.Panic(err)
	}
	defer proxyServer.GracefulStop()
	log.Info(proxyServer.Serve(listenSocket))
}
