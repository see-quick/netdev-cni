//go:build linux

// Copyright netdev-cni authors. Apache 2.0 License.
package cni

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/containernetworking/cni/pkg/skel"
	cnitypes "github.com/containernetworking/cni/pkg/types"
	"github.com/netdev-cni/netdev-cni/pkg/agent"
	"github.com/netdev-cni/netdev-cni/pkg/netns"
)

// PluginConf is the CNI config JSON passed by the runtime.
type PluginConf struct {
	cnitypes.NetConf
	AgentSocket string          `json:"agentSocket"`
	DeviceType  string          `json:"deviceType"`
	IPAM        json.RawMessage `json:"ipam"`
}

// AgentClient calls the node agent Unix socket.
type AgentClient struct {
	sockPath string
	timeout  time.Duration
}

func NewAgentClient(sockPath string, timeoutSecs int) (*AgentClient, error) {
	return &AgentClient{
		sockPath: sockPath,
		timeout:  time.Duration(timeoutSecs) * time.Second,
	}, nil
}

func (c *AgentClient) Allocate(podRef string) (*agent.VFResponse, error) {
	return c.call(agent.Request{Command: "allocate", PodRef: podRef})
}

func (c *AgentClient) Release(podRef string) error {
	_, err := c.call(agent.Request{Command: "release", PodRef: podRef})
	return err
}

func (c *AgentClient) call(req agent.Request) (*agent.VFResponse, error) {
	conn, err := net.DialTimeout("unix", c.sockPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("connect to agent: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(c.timeout)) //nolint:errcheck

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	var resp agent.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("agent error: %s", resp.Error)
	}
	return resp.VF, nil
}

// CmdAdd handles the CNI ADD command.
func CmdAdd(args *skel.CmdArgs) error {
	conf, err := parseConf(args.StdinData)
	if err != nil {
		return err
	}
	podRef := fmt.Sprintf("%s/%s", args.Args, safeID(args.ContainerID))

	client, err := NewAgentClient(conf.AgentSocket, 5)
	if err != nil {
		return err
	}
	vf, err := client.Allocate(podRef)
	if err != nil {
		return fmt.Errorf("allocate VF: %w", err)
	}

	switch conf.DeviceType {
	case "vfio-pci":
		return writePCIAddress(safeID(args.ContainerID), vf.PCIAddress)
	case "rdma":
		if err := netns.MoveInterfaceToNetns(vf.Name, "net1", args.Netns); err != nil {
			_ = client.Release(podRef)
			return err
		}
		// rdma_rxe device move is a no-op in simulation; real implementation TBD.
	default: // netdevice
		if err := netns.MoveInterfaceToNetns(vf.Name, "net1", args.Netns); err != nil {
			_ = client.Release(podRef)
			return err
		}
	}

	result := netns.BuildCNIResult("net1", "0.0.0.0/0", "0.0.0.0")
	return cnitypes.PrintResult(result, conf.CNIVersion)
}

// CmdDel handles the CNI DEL command.
func CmdDel(args *skel.CmdArgs) error {
	conf, err := parseConf(args.StdinData)
	if err != nil {
		return err
	}
	podRef := fmt.Sprintf("%s/%s", args.Args, safeID(args.ContainerID))

	if conf.DeviceType != "vfio-pci" {
		_ = netns.MoveInterfaceToHostNetns("net1", args.Netns)
	}

	client, _ := NewAgentClient(conf.AgentSocket, 5)
	return client.Release(podRef)
}

// CmdCheck handles the CNI CHECK command.
func CmdCheck(args *skel.CmdArgs) error {
	conf, err := parseConf(args.StdinData)
	if err != nil {
		return err
	}
	if conf.DeviceType == "vfio-pci" {
		return nil
	}
	out, err := exec.Command("nsenter", "--net="+args.Netns, "ip", "link", "show", "net1").CombinedOutput()
	if err != nil {
		return fmt.Errorf("CHECK: net1 not found in netns %s: %w: %s", args.Netns, err, out)
	}
	return nil
}

func parseConf(data []byte) (*PluginConf, error) {
	conf := &PluginConf{}
	if err := json.Unmarshal(data, conf); err != nil {
		return nil, fmt.Errorf("parse CNI config: %w", err)
	}
	if conf.AgentSocket == "" {
		conf.AgentSocket = "/var/run/netdev-cni/agent.sock"
	}
	if conf.DeviceType == "" {
		conf.DeviceType = "netdevice"
	}
	return conf, nil
}

func writePCIAddress(containerID, pciAddr string) error {
	path := fmt.Sprintf("/var/run/netdev-cni/%s.pci", containerID)
	return os.WriteFile(path, []byte(pciAddr), 0644)
}

func safeID(containerID string) string {
	if len(containerID) >= 12 {
		return containerID[:12]
	}
	return containerID
}
