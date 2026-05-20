// Copyright netdev-cni authors. Apache 2.0 License.
package agent_test

import (
	"testing"

	"github.com/netdev-cni/netdev-cni/pkg/agent"
	"github.com/netdev-cni/netdev-cni/pkg/apis/v1alpha1"
)

func TestAllocateAndRelease(t *testing.T) {
	vfs := []v1alpha1.VFInfo{
		{ID: 0, Name: "sim_vf0", Allocated: false},
		{ID: 1, Name: "sim_vf1", Allocated: false},
	}
	pool := agent.NewPool(vfs)

	vf, err := pool.Allocate("default/pod-a")
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if vf.Name != "sim_vf0" {
		t.Errorf("expected sim_vf0, got %s", vf.Name)
	}
	if !vf.Allocated {
		t.Error("VF should be marked allocated")
	}

	if err := pool.Release("default/pod-a"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	snapshot := pool.Snapshot()
	for _, v := range snapshot {
		if v.Allocated {
			t.Errorf("VF %s still allocated after release", v.Name)
		}
	}
}

func TestAllocateExhausted(t *testing.T) {
	vfs := []v1alpha1.VFInfo{
		{ID: 0, Name: "sim_vf0", Allocated: false},
	}
	pool := agent.NewPool(vfs)
	if _, err := pool.Allocate("default/pod-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Allocate("default/pod-b"); err == nil {
		t.Error("expected error on exhausted pool")
	}
}

func TestRehydrateFromSnapshot(t *testing.T) {
	vfs := []v1alpha1.VFInfo{
		{ID: 0, Name: "sim_vf0", Allocated: true, PodRef: "default/pod-a"},
		{ID: 1, Name: "sim_vf1", Allocated: false},
	}
	pool := agent.NewPool(vfs)
	_, err := pool.Allocate("default/pod-b")
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	snap := pool.Snapshot()
	if snap[0].PodRef != "default/pod-a" {
		t.Errorf("pod-a allocation lost after rehydrate")
	}
}
