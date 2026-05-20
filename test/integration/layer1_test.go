//go:build integration

// Copyright netdev-cni authors. Apache 2.0 License.
package integration_test

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

func kubectl(args ...string) (string, error) {
	out, err := exec.Command("kubectl", args...).CombinedOutput()
	return string(out), err
}

func TestLayer1NetdeviceInterface(t *testing.T) {
	if _, err := kubectl("apply", "-f", "../../deploy/kind/test-pod-layer1.yaml"); err != nil {
		t.Fatalf("apply pod: %v", err)
	}
	defer kubectl("delete", "-f", "../../deploy/kind/test-pod-layer1.yaml", "--ignore-not-found") //nolint:errcheck

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		out, _ := kubectl("get", "pod", "test-layer1", "-o", "jsonpath={.status.phase}")
		if strings.TrimSpace(out) == "Running" {
			break
		}
		time.Sleep(2 * time.Second)
	}

	out, err := kubectl("exec", "test-layer1", "--", "ip", "link", "show", "net1")
	if err != nil {
		t.Fatalf("net1 not found in pod: %v\n%s", err, out)
	}
	if !strings.Contains(out, "net1") {
		t.Errorf("unexpected ip link output: %s", out)
	}
	t.Logf("Layer 1 PASS: net1 interface present in pod\n%s", out)
}
