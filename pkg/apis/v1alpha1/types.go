// Copyright netdev-cni authors. Apache 2.0 License.
package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// SriovNetworkNodePolicy configures per-node SR-IOV topology.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type SriovNetworkNodePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovNetworkNodePolicySpec   `json:"spec,omitempty"`
	Status SriovNetworkNodePolicyStatus `json:"status,omitempty"`
}

type SriovNetworkNodePolicySpec struct {
	// NodeSelector selects nodes this policy applies to.
	NodeSelector map[string]string `json:"nodeSelector"`
	// NumVfs is the number of VFs to create (or simulate).
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=64
	NumVfs int `json:"numVfs"`
	// NicSelector identifies the physical NIC.
	NicSelector NicSelector `json:"nicSelector"`
	// DeviceType controls how VFs are exposed: "netdevice", "vfio-pci", or "rdma".
	// +kubebuilder:validation:Enum=netdevice;vfio-pci;rdma
	DeviceType string `json:"deviceType"`
}

type NicSelector struct {
	// PfNames is a list of physical function interface names (e.g. "eth1").
	PfNames []string `json:"pfNames"`
}

type SriovNetworkNodePolicyStatus struct {
	// Conditions reflect per-node readiness.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
type SriovNetworkNodePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SriovNetworkNodePolicy `json:"items"`
}

// SriovNetwork creates a NetworkAttachmentDefinition pods can reference.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type SriovNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovNetworkSpec   `json:"spec,omitempty"`
	Status SriovNetworkStatus `json:"status,omitempty"`
}

type SriovNetworkSpec struct {
	// NetworkNamespace is the namespace where the NetworkAttachmentDefinition is created.
	NetworkNamespace string `json:"networkNamespace"`
	// ResourceName is the device plugin resource name (e.g. "sriov.io/vf").
	ResourceName string `json:"resourceName"`
	// IPAM is the IPAM config JSON string passed to the CNI IPAM plugin.
	IPAM string `json:"ipam"`
	// Capabilities enables optional network capabilities.
	Capabilities NetworkCapabilities `json:"capabilities,omitempty"`
}

type NetworkCapabilities struct {
	DPDK bool `json:"dpdk,omitempty"`
	RoCE bool `json:"roce,omitempty"`
}

type SriovNetworkStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
type SriovNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SriovNetwork `json:"items"`
}

// SriovNetworkNodeState is the per-node VF allocation state, written by the node agent.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type SriovNetworkNodeState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovNetworkNodeStateSpec   `json:"spec,omitempty"`
	Status SriovNetworkNodeStateStatus `json:"status,omitempty"`
}

type SriovNetworkNodeStateSpec struct {
	// Interfaces is written by the operator to instruct the node agent.
	Interfaces []InterfaceSpec `json:"interfaces,omitempty"`
}

type InterfaceSpec struct {
	Name       string `json:"name"`
	NumVfs     int    `json:"numVfs"`
	DeviceType string `json:"deviceType"`
}

type SriovNetworkNodeStateStatus struct {
	// Interfaces reflects the actual VF state on this node.
	Interfaces []InterfaceStatus `json:"interfaces,omitempty"`
	// SyncStatus is "Succeeded", "Failed", or "InProgress".
	SyncStatus string `json:"syncStatus,omitempty"`
	// SimulationMode is true when the node agent is running in simulation mode.
	SimulationMode bool `json:"simulationMode,omitempty"`
}

type InterfaceStatus struct {
	Name         string   `json:"name"`
	TotalVfs     int      `json:"totalVfs"`
	AllocatedVfs int      `json:"allocatedVfs"`
	DeviceType   string   `json:"deviceType"`
	VFs          []VFInfo `json:"vfs,omitempty"`
}

type VFInfo struct {
	// ID is the VF index (0-based).
	ID int `json:"id"`
	// Name is the interface name (e.g. "eth1v0" or "sim_vf0").
	Name string `json:"name"`
	// PCIAddress is set for vfio-pci device type.
	PCIAddress string `json:"pciAddress,omitempty"`
	// Allocated is true when this VF is assigned to a pod.
	Allocated bool `json:"allocated"`
	// PodRef is the namespace/name of the pod that holds this VF.
	PodRef string `json:"podRef,omitempty"`
}

// +kubebuilder:object:root=true
type SriovNetworkNodeStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SriovNetworkNodeState `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&SriovNetworkNodePolicy{}, &SriovNetworkNodePolicyList{},
		&SriovNetwork{}, &SriovNetworkList{},
		&SriovNetworkNodeState{}, &SriovNetworkNodeStateList{},
	)
}
