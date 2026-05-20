//go:build linux

// Copyright netdev-cni authors. Apache 2.0 License.
package cni_test

import (
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	agentpkg "github.com/netdev-cni/netdev-cni/pkg/agent"
	"github.com/netdev-cni/netdev-cni/pkg/apis/v1alpha1"
	"github.com/netdev-cni/netdev-cni/pkg/cni"
)

func startTestAgent(t *testing.T, sockPath string) {
	t.Helper()
	os.Remove(sockPath)
	vfs := []v1alpha1.VFInfo{
		{ID: 0, Name: "lo", Allocated: false, PCIAddress: "0000:00:00.0"},
	}
	pool := agentpkg.NewPool(vfs)
	srv := agentpkg.NewServer(sockPath, pool, "netdevice")
	go srv.ListenAndServe()
	t.Cleanup(srv.Stop)
	time.Sleep(30 * time.Millisecond)
}

func TestPluginAllocatesVF(t *testing.T) {
	sockPath := "/tmp/netdev-cni-plugin-test.sock"
	startTestAgent(t, sockPath)

	cfg := cni.PluginConf{
		AgentSocket: sockPath,
		DeviceType:  "netdevice",
		IPAM:        json.RawMessage(`{"type":"static"}`),
	}
	_ = cfg

	client, err := cni.NewAgentClient(sockPath, 2)
	if err != nil {
		t.Fatalf("NewAgentClient: %v", err)
	}

	resp, err := client.Allocate("default/test-pod")
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if resp.Name != "lo" {
		t.Errorf("expected lo (test VF), got %s", resp.Name)
	}
	conn, _ := net.Dial("unix", sockPath)
	conn.Close()
}
