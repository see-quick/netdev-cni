// Copyright netdev-cni authors. Apache 2.0 License.
package netns

import (
	"fmt"
	"net"
	"os/exec"

	current "github.com/containernetworking/cni/pkg/types/100"
)

// MoveInterfaceToNetns moves ifName into the network namespace at netnsPath,
// renaming it to newName inside the netns.
func MoveInterfaceToNetns(ifName, newName, netnsPath string) error {
	out, err := exec.Command("ip", "link", "set", ifName, "netns", netnsPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("move %s to netns %s: %w: %s", ifName, netnsPath, err, out)
	}
	out, err = exec.Command(
		"nsenter", "--net="+netnsPath,
		"ip", "link", "set", ifName, "name", newName,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rename %s -> %s in netns: %w: %s", ifName, newName, err, out)
	}
	out, err = exec.Command(
		"nsenter", "--net="+netnsPath,
		"ip", "link", "set", newName, "up",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("set %s up in netns: %w: %s", newName, err, out)
	}
	return nil
}

// MoveInterfaceToHostNetns moves ifName from netnsPath back to the host netns (pid 1's netns).
func MoveInterfaceToHostNetns(ifName, netnsPath string) error {
	out, err := exec.Command(
		"nsenter", "--net="+netnsPath,
		"ip", "link", "set", ifName, "netns", "1",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("move %s back to host netns: %w: %s", ifName, err, out)
	}
	return nil
}

// ConfigureIP assigns addr (CIDR) and default gateway gw to ifName inside netnsPath.
func ConfigureIP(ifName, addr, gw, netnsPath string) error {
	out, err := exec.Command(
		"nsenter", "--net="+netnsPath,
		"ip", "addr", "add", addr, "dev", ifName,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("add addr %s to %s: %w: %s", addr, ifName, err, out)
	}
	out, err = exec.Command(
		"nsenter", "--net="+netnsPath,
		"ip", "route", "add", "default", "via", gw,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("add default route via %s: %w: %s", gw, err, out)
	}
	return nil
}

// BuildCNIResult constructs the CNI result structure the runtime expects.
func BuildCNIResult(ifName, cidr, gateway string) *current.Result {
	ip, ipNet, _ := net.ParseCIDR(cidr)
	ipNet.IP = ip
	gwIP := net.ParseIP(gateway)
	idx := 0
	return &current.Result{
		CNIVersion: "1.0.0",
		Interfaces: []*current.Interface{
			{Name: ifName, Sandbox: "container"},
		},
		IPs: []*current.IPConfig{
			{
				Interface: &idx,
				Address:   *ipNet,
				Gateway:   gwIP,
			},
		},
	}
}
