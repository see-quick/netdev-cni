CONTROLLER_GEN ?= $(shell which controller-gen 2>/dev/null || echo "$(GOPATH)/bin/controller-gen")
IMG_OPERATOR   ?= netdev-cni/operator:latest
IMG_AGENT      ?= netdev-cni/node-agent:latest
IMG_CNI        ?= netdev-cni/cni-plugin:latest

.PHONY: all build test generate fmt vet kind-setup kind-deploy kind-test kind-teardown

all: generate build test

build:
	GOOS=linux GOARCH=amd64 go build -o bin/cni-plugin ./cmd/cni-plugin/
	GOOS=linux GOARCH=amd64 go build -o bin/operator   ./cmd/operator/
	GOOS=linux GOARCH=amd64 go build -o bin/node-agent ./cmd/node-agent/

test:
	go test ./... -v -count=1

fmt:
	go fmt ./...

vet:
	go vet ./...

generate:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/apis/..."
	$(CONTROLLER_GEN) crd paths="./pkg/apis/..." output:crd:artifacts:config=deploy/crds

install-tools:
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0

kind-setup:
	kind create cluster --config deploy/kind/cluster.yaml --name netdev-cni
	kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset.yml

kind-deploy: build
	kind load docker-image $(IMG_OPERATOR) --name netdev-cni
	kind load docker-image $(IMG_AGENT)    --name netdev-cni
	kubectl apply -f deploy/crds/
	kubectl apply -f deploy/operator.yaml
	kubectl apply -f deploy/daemonset.yaml

kind-test:
	go test ./test/integration/... -v -tags=integration

kind-teardown:
	kind delete cluster --name netdev-cni
