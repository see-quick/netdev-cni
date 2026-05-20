# netdev-cni Design Spec

**Date:** 2026-05-20  
**Status:** Approved  
**Repo:** `netdev-cni`

---

## Overview

`netdev-cni` is a learning/reference CNI plugin for Kubernetes that implements SR-IOV Virtual Function (VF) attachment, DPDK userspace networking, and RoCE/RDMA connectivity — all testable on a local kind cluster via a simulation mode backed by veth pairs and software RoCE (`rdma_rxe`).

The project follows the full operator pattern: CNI binary + DaemonSet node agent + CRD-driven cluster operator.

---

## Goals

- Implement the CNI spec (`ADD`/`DEL`/`CHECK`) end-to-end in Go
- Build a CRD-driven operator that manages SR-IOV topology across nodes
- Support three progressive technology layers: SR-IOV (`netdevice`), DPDK (`vfio-pci`), RoCE (`rdma`)
- Run fully on kind with simulation mode — no special hardware required
- Serve as a learning reference for CNI plugin development, Kubernetes operator patterns, and high-performance networking

---

## Repository Structure

```
netdev-cni/
cmd/
  cni-plugin/        # CNI binary (ADD/DEL/CHECK)
  operator/          # Operator main
  node-agent/        # DaemonSet node agent main
pkg/
  apis/              # CRD types (SriovNetwork, SriovNetworkNodePolicy, SriovNetworkNodeState)
  controllers/       # Operator reconcilers
  agent/             # Node agent logic (VF pool, driver binding, simulation)
  netns/             # Netns wiring helpers
  simulation/        # veth/macvlan simulation backend
deploy/
  crds/              # Generated CRD YAMLs (controller-gen)
  operator.yaml
  daemonset.yaml
  kind/              # kind cluster config + test manifests
docs/
  superpowers/specs/ # Design specs
Makefile
go.mod
```

**Language:** Go  
**Key dependencies:** `controller-runtime`, `containernetworking/plugins`, `controller-gen`, `rdma-core` (for RoCE layer tests)  
**Build:** `Makefile` with `ko` or `docker buildx` for images

---

## CRDs

### `SriovNetworkNodePolicy` (cluster-scoped)

Configures per-node SR-IOV topology. Written by the user, reconciled by the operator.

```yaml
apiVersion: netdev.io/v1alpha1
kind: SriovNetworkNodePolicy
metadata:
  name: dpdk-policy
spec:
  nodeSelector:
    feature.node.kubernetes.io/network-sriov: "true"
  numVfs: 4
  nicSelector:
    pfNames: ["eth1"]
  deviceType: vfio-pci   # "netdevice" | "vfio-pci" | "rdma"
  # mode is auto-detected by node agent (simulation when no sriov sysfs present)
```

### `SriovNetwork` (namespace-scoped)

Creates a `NetworkAttachmentDefinition` (Multus-compatible) that pods reference via annotation.

```yaml
apiVersion: netdev.io/v1alpha1
kind: SriovNetwork
metadata:
  name: roce-network
  namespace: default
spec:
  networkNamespace: default
  resourceName: sriov.io/vf
  ipam: |
    {"type": "host-local", "subnet": "192.168.1.0/24"}
  capabilities:
    dpdk: false
    roce: true
```

### `SriovNetworkNodeState` (per-node status)

Written by the node agent, read by the operator. Source of truth for VF allocation state.

```yaml
apiVersion: netdev.io/v1alpha1
kind: SriovNetworkNodeState
metadata:
  name: kind-worker
status:
  interfaces:
    - name: eth1
      totalVfs: 4
      allocatedVfs: 1
      deviceType: vfio-pci
      mode: simulation
```

---

## Components

### 1. Cluster Operator (single Deployment)

- Watches `SriovNetworkNodePolicy` → writes `SriovNetworkNodeState` per matching node
- Watches `SriovNetwork` → creates/updates `NetworkAttachmentDefinition` CRs
- Aggregates per-node status back onto `SriovNetworkNodePolicy.status`
- Detects node agent not reporting → marks `SriovNetworkNodeState` as `NotReady`

### 2. Node Agent (DaemonSet, one pod per node)

- Reads its own `SriovNetworkNodeState` from the API server
- **Simulation mode** (auto-detected when `/sys/class/net/*/device/sriov_numvfs` absent):
  - Creates veth pairs named `sim_vf0`, `sim_vf1`, ... to represent VFs
  - Loads `rdma_rxe` module and creates rxe device over veth for RoCE layer
- **Real SR-IOV mode:** configures PF numVfs via sysfs, binds VFs to correct driver
- Maintains VF pool; writes allocation state to `SriovNetworkNodeState.status`
- Exposes a Unix socket — CNI binary calls it to request/release a VF

### 3. CNI Binary (`/opt/cni/bin/netdev-cni`)

- Invoked by containerd/kubelet via CNI spec (stdin JSON, stdout JSON)
- `ADD`: calls node agent socket → gets VF name → wires into pod netns → calls IPAM plugin
- `DEL`: calls node agent socket to release VF → moves interface back to host netns
- `CHECK`: verifies interface exists in pod netns
- Socket call timeout: 5s (kubelet has a hard CNI timeout)

---

## Data Flow — Pod ADD

```
kubelet
  → containerd invokes CNI binary (ADD, pod netns path, config JSON)
    → CNI binary: calls node agent Unix socket "allocate VF for pod X"
      → node agent: picks free VF from pool (sim_vf0 or eth1v0)
      → returns VF name + device info
    → CNI binary: moves VF into pod netns, renames to "net1"
    → CNI binary: calls IPAM plugin (host-local) for IP/route assignment
    → CNI binary: returns result JSON to containerd
  → pod starts with "net1" interface configured
```

---

## Progressive Technology Layers

### Layer 1 — `netdevice` (SR-IOV baseline)

- VF bound to kernel driver
- Node agent moves VF into pod netns via `ip link set ... netns`
- CNI binary configures MAC, IP, routes inside netns
- Simulation: veth pair, transparent to pod
- **Learn:** CNI spec, netns manipulation, VF lifecycle

### Layer 2 — `vfio-pci` (DPDK)

- Node agent unbinds VF from kernel driver, binds to `vfio-pci` via sysfs
- CNI binary does NOT move interface into netns — writes PCI address to a file in pod's volume mount
- Pod reads PCI address, passes to `rte_eal_init()`
- Simulation: skip driver binding, write fake PCI address `0000:00:00.0`
- **Learn:** vfio-pci lifecycle, DPDK interface contract, userspace networking

### Layer 3 — `rdma` (RoCE)

- VF stays kernel-bound; RDMA subsystem enabled
- Node agent loads `rdma_rxe` (software RoCE) on kind — **this actually works**, kind nodes share the host kernel
- CNI binary moves VF into pod netns and moves RDMA device into pod's RDMA namespace
- Simulation: `rdma_rxe` over veth = real RDMA verbs (`ibv_post_send`, `ibv_poll_cq`) in CI
- **Learn:** RDMA namespace management, soft-RoCE, real RDMA verbs without hardware

---

## Error Handling

### CNI Binary
- Node agent socket unavailable → fail ADD immediately (no retry, kubelet retries scheduling)
- VF pool exhausted → return CNI error `INSUFFICIENT_RESOURCE`
- Netns wiring fails midway → best-effort cleanup via self-DEL, return error
- All socket calls: 5s timeout

### Node Agent
- Simulation mode: permanent per node boot, logged clearly at startup
- VF state survives restart via `SriovNetworkNodeState.status` (not local memory)
- `rdma_rxe` load failure → log warning, mark RoCE unavailable in node state, continue
- Driver bind/unbind failure → update `SriovNetworkNodeState` with error condition

### Operator
- Node agent not reporting → mark `SriovNetworkNodeState` as `NotReady`
- `NetworkAttachmentDefinition` exists with different owner → log conflict, skip
- Partial node fleet → per-node status tracked independently

### kind-specific
- No sriov sysfs → simulation mode (automatic)
- `vfio-pci` not available → log warning, fake PCI address written for DPDK simulation
- `rdma_rxe` requires `rdma-core` in kind node image → kind config includes init script

---

## Testing Strategy

### Unit Tests
- CNI binary: mock node agent socket, test ADD/DEL/CHECK JSON, netns wiring logic
- Node agent: test VF pool alloc/release, simulation backend, mock sysfs for driver binding
- Operator reconcilers: `envtest` for CRD reconciliation, `NetworkAttachmentDefinition` generation, status updates

### Integration Tests (kind)
- `kind/` directory: cluster config with Multus pre-installed, `rdma_rxe` init script, custom kind node image if needed
- Layer 1: pod with `k8s.v1.cni.cncf.io/networks: sriov-net` annotation → verify `net1` interface + IP
- Layer 2: DPDK test pod → verify PCI address file written, `testpmd` stub inits
- Layer 3: two pods with RoCE → `ibv_devices` shows rxe device, `rping` succeeds between pods

### Make Targets
```makefile
make test           # unit tests
make generate       # run controller-gen for CRDs and deepcopy
make kind-setup     # create kind cluster + load images
make kind-deploy    # deploy CRDs + operator + daemonset
make kind-test      # integration tests against kind
make kind-teardown  # destroy cluster
```

### Observability
- Structured logs (zap) in operator and node agent with VF allocation events
- `SriovNetworkNodeState.status` as primary debugging surface (`kubectl get sriovnetworknodestate -o yaml`)

---

## Out of Scope

- Production-grade VF NUMA-aware scheduling
- Multi-NIC / multi-policy per node (single NIC per policy for now)
- IPv6 IPAM
- Windows nodes
