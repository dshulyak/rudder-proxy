SHELL := /bin/bash

test:
	go test ./pkg/...

patch:
	./patch.sh

deps:
	glide install --strip-vendor
	pushd vendor/istio.io/api/ && \
	protoc --go_out=plugins=grpc:. ./proxy/v1/config/*.proto && \
	popd

build: vendor/
	CGO_ENABLED=0 go build -o istio-proxy ./cmd/

docker: build
	docker build -t yashulyak/istio-rudder-proxy .
	docker push yashulyak/istio-rudder-proxy

clean:
	-rm -f istio-proxy
	-docker rmi yashulyak/istio-rudder-proxy

release: clean docker
