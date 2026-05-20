// Copyright netdev-cni authors. Apache 2.0 License.
package simulation_test

import (
	"testing"

	"github.com/netdev-cni/netdev-cni/pkg/simulation"
)

func TestCreateSimulatedVFs(t *testing.T) {
	backend := simulation.NewVethBackend("testsim")
	vfs, err := backend.CreateVFs(2)
	if err != nil {
		t.Fatalf("CreateVFs: %v", err)
	}
	if len(vfs) != 2 {
		t.Fatalf("expected 2 VFs, got %d", len(vfs))
	}
	for i, vf := range vfs {
		if vf.Name == "" {
			t.Errorf("VF[%d] has empty name", i)
		}
	}
	// cleanup
	if err := backend.DeleteVFs(vfs); err != nil {
		t.Errorf("DeleteVFs: %v", err)
	}
}
