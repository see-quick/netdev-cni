// Copyright netdev-cni authors. Apache 2.0 License.
package simulation

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/netdev-cni/netdev-cni/pkg/apis/v1alpha1"
)

// VethBackend creates veth pairs to simulate SR-IOV VFs.
type VethBackend struct {
	prefix string
}

func NewVethBackend(prefix string) *VethBackend {
	return &VethBackend{prefix: prefix}
}

// CreateVFs creates n veth pairs. The "vf" end of each pair is returned as a VFInfo.
// The peer end stays in the host netns for routing purposes.
func (b *VethBackend) CreateVFs(n int) ([]v1alpha1.VFInfo, error) {
	vfs := make([]v1alpha1.VFInfo, 0, n)
	for i := 0; i < n; i++ {
		vfName := fmt.Sprintf("%s_vf%d", b.prefix, i)
		peerName := fmt.Sprintf("%s_vf%dp", b.prefix, i)
		if err := b.createVeth(vfName, peerName); err != nil {
			return nil, fmt.Errorf("create veth %s: %w", vfName, err)
		}
		vfs = append(vfs, v1alpha1.VFInfo{
			ID:         i,
			Name:       vfName,
			PCIAddress: fmt.Sprintf("0000:00:%02x.0", i),
			Allocated:  false,
		})
	}
	return vfs, nil
}

// DeleteVFs removes the veth pairs.
func (b *VethBackend) DeleteVFs(vfs []v1alpha1.VFInfo) error {
	for _, vf := range vfs {
		if _, err := net.InterfaceByName(vf.Name); err != nil {
			continue // already gone
		}
		if err := exec.Command("ip", "link", "delete", vf.Name).Run(); err != nil {
			return fmt.Errorf("delete link %s: %w", vf.Name, err)
		}
	}
	return nil
}

func (b *VethBackend) createVeth(name, peer string) error {
	// delete stale pair if present
	_ = exec.Command("ip", "link", "delete", name).Run()
	cmd := exec.Command("ip", "link", "add", name, "type", "veth", "peer", "name", peer)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	for _, iface := range []string{name, peer} {
		if out, err := exec.Command("ip", "link", "set", iface, "up").CombinedOutput(); err != nil {
			return fmt.Errorf("set %s up: %w: %s", iface, err, out)
		}
	}
	return nil
}
