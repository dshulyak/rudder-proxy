FROM alpine:3.4
LABEL Name="istio-rudder-proxy" Version="0.1"
ADD istio-proxy /usr/sbin/istio-proxy
ENTRYPOINT ["/usr/sbin/istio-proxy"]