// Copyright netdev-cni authors. Apache 2.0 License.
package agent

import "os"

// IsSimulationMode returns true when no SR-IOV sysfs entry is found under sriovSysfsDir.
// In production pass "/sys/class/net"; in tests pass a temp dir.
func IsSimulationMode(sriovSysfsDir string) bool {
	entries, err := os.ReadDir(sriovSysfsDir)
	if err != nil {
		return true
	}
	for _, e := range entries {
		candidate := sriovSysfsDir + "/" + e.Name() + "/device/sriov_numvfs"
		if _, err := os.Stat(candidate); err == nil {
			return false
		}
	}
	return true
}
