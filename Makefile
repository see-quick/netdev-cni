CONTROLLER_GEN ?= $(shell which controller-gen 2>/dev/null || echo "$(GOPATH)/bin/controller-gen")
CONTAINER_CMD  ?= $(shell command -v podman 2>/dev/null || echo docker)
IMG_OPERATOR   ?= localhost/netdev-cni/operator:latest
IMG_AGENT      ?= localhost/netdev-cni/node-agent:latest
IMG_CNI        ?= localhost/netdev-cni/cni-plugin:latest

.PHONY: all build test generate fmt vet docker-build kind-setup kind-deploy kind-test kind-teardown

all: generate build test

build:
	GOOS=linux GOARCH=amd64 go build -o bin/cni-plugin ./cmd/cni-plugin/
	GOOS=linux GOARCH=amd64 go build -o bin/operator   ./cmd/operator/
	GOOS=linux GOARCH=amd64 go build -o bin/node-agent ./cmd/node-agent/

docker-build:
	$(CONTAINER_CMD) build -t $(IMG_CNI)      -f cmd/cni-plugin/Dockerfile .
	$(CONTAINER_CMD) build -t $(IMG_OPERATOR)  -f cmd/operator/Dockerfile .
	$(CONTAINER_CMD) build -t $(IMG_AGENT)     -f cmd/node-agent/Dockerfile .

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

kind-load:
	$(CONTAINER_CMD) save $(IMG_CNI) $(IMG_OPERATOR) $(IMG_AGENT) -o /tmp/netdev-cni-images.tar
	kind load image-archive /tmp/netdev-cni-images.tar --name netdev-cni
	rm -f /tmp/netdev-cni-images.tar

kind-deploy: docker-build kind-load
	kubectl apply -f deploy/crds/
	kubectl apply -f deploy/operator.yaml
	kubectl apply -f deploy/daemonset.yaml

kind-test:
	go test ./test/integration/... -v -tags=integration

kind-teardown:
	kind delete cluster --name netdev-cni
