// Copyright netdev-cni authors. Apache 2.0 License.
package agent

// Request is sent by the CNI binary to the node agent over the Unix socket.
type Request struct {
	// Command is "allocate" or "release".
	Command string `json:"command"`
	// PodRef is "namespace/name".
	PodRef string `json:"podRef"`
}

// Response is sent back to the CNI binary.
type Response struct {
	// VF is the allocated VF info. Nil on release or error.
	VF    *VFResponse `json:"vf,omitempty"`
	Error string      `json:"error,omitempty"`
}

// VFResponse is the VF info the CNI binary needs to wire the interface.
type VFResponse struct {
	Name       string `json:"name"`
	PCIAddress string `json:"pciAddress,omitempty"`
	DeviceType string `json:"deviceType"`
}
