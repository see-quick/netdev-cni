// Copyright netdev-cni authors. Apache 2.0 License.
package agent

import (
	"fmt"
	"sync"

	"github.com/netdev-cni/netdev-cni/pkg/apis/v1alpha1"
)

// Pool manages VF allocation. Thread-safe.
type Pool struct {
	mu  sync.Mutex
	vfs []v1alpha1.VFInfo
}

func NewPool(vfs []v1alpha1.VFInfo) *Pool {
	copied := make([]v1alpha1.VFInfo, len(vfs))
	copy(copied, vfs)
	return &Pool{vfs: copied}
}

// Allocate assigns a free VF to podRef. Returns error if pool is exhausted.
func (p *Pool) Allocate(podRef string) (v1alpha1.VFInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, vf := range p.vfs {
		if !vf.Allocated {
			p.vfs[i].Allocated = true
			p.vfs[i].PodRef = podRef
			return p.vfs[i], nil
		}
	}
	return v1alpha1.VFInfo{}, fmt.Errorf("VF pool exhausted")
}

// Release marks the VF held by podRef as free.
func (p *Pool) Release(podRef string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, vf := range p.vfs {
		if vf.PodRef == podRef {
			p.vfs[i].Allocated = false
			p.vfs[i].PodRef = ""
			return nil
		}
	}
	return fmt.Errorf("no VF found for pod %s", podRef)
}

// Snapshot returns a copy of the current VF state (for status reporting).
func (p *Pool) Snapshot() []v1alpha1.VFInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]v1alpha1.VFInfo, len(p.vfs))
	copy(out, p.vfs)
	return out
}
