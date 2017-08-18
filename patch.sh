#!/bin/bash

kubectl patch deployment/tiller-deploy -n kube-system -p='{"spec":{"template":{"spec":{"$setElementOrder/containers":[{"name":"tiller"},{"name":"rudder"},{"name":"istio-rudder-proxy"}],"containers":[{"args":["--experimental-release"],"name":"tiller"},{"args":["-l","0.0.0.0:10002"],"image":"yashulyak/rudder","name":"rudder","resources":{}},{"args":["-l","0.0.0.0:10001","-s","0.0.0.0:10002"],"image":"yashulyak/istio-rudder-proxy","name":"istio-rudder-proxy","resources":{}}]}}}}'
