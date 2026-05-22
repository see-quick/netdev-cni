# netdev-cni

> [!NOTE] 
> This project is for learning purposes only. It is not production-ready and should not be used in real environments.

A learning/reference CNI plugin for Kubernetes demonstrating three progressive network device technologies:

| Layer | Technology           | Mode                              |
|-------|----------------------|-----------------------------------|
| 1     | SR-IOV (`netdevice`) | VF moved into pod netns           |
| 2     | DPDK (`vfio-pci`)    | PCI address written to pod volume |
| 3     | RoCE / RDMA (`rdma`) | VF + soft-RoCE (`rdma_rxe`)       |

On clusters without real SR-IOV hardware (e.g. kind), the agent detects simulation mode and creates **veth pairs** instead of VFs.

## Architecture

```
┌──────────────────────────────────────┐
│  Cluster Operator (controller-runtime)│
│  Reconciles SriovNetworkNodePolicy   │
│  → SriovNetworkNodeState             │
│  Reconciles SriovNetwork             │
│  → Multus NetworkAttachmentDefinition│
└──────────────────────────────────────┘
              ↓ per node
┌──────────────────────────────────────┐
│  Node Agent (DaemonSet)              │
│  Detects VFs or creates veth pairs   │
│  Manages VF pool                     │
│  Unix socket: /var/run/netdev-cni/   │
└──────────────────────────────────────┘
              ↓ CNI ADD/DEL
┌──────────────────────────────────────┐
│  CNI Binary (/opt/cni/bin/netdev-cni)│
│  Calls agent to allocate/release VF  │
│  Moves interface into pod netns      │
└──────────────────────────────────────┘
```

## Quick Start (kind simulation)

```bash
# 1. Build images
make build

# 2. Create kind cluster
kind create cluster --config deploy/kind/cluster.yaml

# 3. Load images
kind load docker-image netdev-cni/cni-plugin:latest netdev-cni/node-agent:latest netdev-cni/operator:latest

# 4. Install CRDs and Multus
kubectl apply -f deploy/crds/

# 5. Deploy operator and agent
kubectl apply -f deploy/operator.yaml
kubectl apply -f deploy/daemonset.yaml

# 6. Create SR-IOV network policy and network
kubectl apply -f deploy/kind/sriov-net.yaml

# 7. Test layer 1 (netdevice)
kubectl apply -f deploy/kind/test-pod-layer1.yaml
kubectl exec test-layer1 -- ip link show net1
```

## Packages

| Package             | Purpose                                                                      |
|---------------------|------------------------------------------------------------------------------|
| `pkg/apis/v1alpha1` | CRD types: `SriovNetworkNodePolicy`, `SriovNetwork`, `SriovNetworkNodeState` |
| `pkg/simulation`    | Veth-pair simulation backend for kind                                        |
| `pkg/agent`         | VF pool, Unix socket server, IPC protocol                                    |
| `pkg/netns`         | Move interfaces into pod network namespaces                                  |
| `pkg/cni`           | CNI ADD/DEL/CHECK handlers (Linux only)                                      |
| `pkg/controllers`   | Operator reconcilers                                                         |

## Testing

```bash
# Unit tests
KUBEBUILDER_ASSETS=~/envtest-binaries/k8s/1.29.5-darwin-arm64 go test ./... -short

# Integration tests (requires a running kind cluster with the stack deployed)
go test -tags integration ./test/integration/ -v
```

## CRD Generation

```bash
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
controller-gen crd paths="./pkg/apis/..." output:crd:artifacts:config=deploy/crds
controller-gen object paths="./pkg/apis/..."
```

## License

Apache 2.0
