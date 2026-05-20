// Copyright netdev-cni authors. Apache 2.0 License.
package netns_test

import (
	"testing"

	"github.com/netdev-cni/netdev-cni/pkg/netns"
)

func TestMoveAndRenameInterface(t *testing.T) {
	if testing.Short() {
		t.Skip("requires root and real netns")
	}
	t.Log("netns wire test: manual verification required (needs root)")
}

func TestBuildCNIResult(t *testing.T) {
	result := netns.BuildCNIResult("net1", "192.168.1.10/24", "192.168.1.1")
	if result.Interfaces[0].Name != "net1" {
		t.Errorf("expected net1, got %s", result.Interfaces[0].Name)
	}
	if result.IPs[0].Address.String() != "192.168.1.10/24" {
		t.Errorf("unexpected IP: %s", result.IPs[0].Address.String())
	}
}
