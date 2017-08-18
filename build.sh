#!/bin/bash

glide install --strip-vendor
pushd vendor/istio.io/api/
protoc --go_out=plugins=grpc:. ./proxy/v1/config/*.proto
popd
go build -o istio-proxy ./cmd/
docker build -t yashulyak/istio-rudder-proxy .
docker push yashulyak/istio-rudder-proxy
