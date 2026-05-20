// Copyright netdev-cni authors. Apache 2.0 License.
package agent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/netdev-cni/netdev-cni/pkg/agent"
)

func TestDetectSimulationMode(t *testing.T) {
	// Simulate a real sysfs structure: tmpDir/<iface>/device/sriov_numvfs
	tmpDir := t.TempDir()
	ifaceDir := filepath.Join(tmpDir, "eth1", "device")
	if err := os.MkdirAll(ifaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	sriovPath := filepath.Join(ifaceDir, "sriov_numvfs")
	if err := os.WriteFile(sriovPath, []byte("4"), 0644); err != nil {
		t.Fatal(err)
	}
	if agent.IsSimulationMode(tmpDir) {
		t.Error("expected real SR-IOV mode when sriov_numvfs present")
	}

	// Empty dir = no sriov_numvfs = simulation mode.
	emptyDir := t.TempDir()
	if !agent.IsSimulationMode(emptyDir) {
		t.Error("expected simulation mode when sriov_numvfs absent")
	}
}
