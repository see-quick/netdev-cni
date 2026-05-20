// Copyright netdev-cni authors. Apache 2.0 License.
package agent_test

import (
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/netdev-cni/netdev-cni/pkg/agent"
	"github.com/netdev-cni/netdev-cni/pkg/apis/v1alpha1"
)

func TestServerAllocateRelease(t *testing.T) {
	sockPath := "/tmp/netdev-cni-test.sock"
	os.Remove(sockPath)

	vfs := []v1alpha1.VFInfo{
		{ID: 0, Name: "sim_vf0", Allocated: false, PCIAddress: "0000:00:00.0"},
	}
	pool := agent.NewPool(vfs)
	srv := agent.NewServer(sockPath, pool, "netdevice")

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()
	time.Sleep(50 * time.Millisecond)
	defer srv.Stop()

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	req := agent.Request{Command: "allocate", PodRef: "default/test-pod"}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		t.Fatalf("encode: %v", err)
	}

	var resp agent.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "" {
		t.Fatalf("server error: %s", resp.Error)
	}
	if resp.VF == nil || resp.VF.Name != "sim_vf0" {
		t.Errorf("unexpected VF: %+v", resp.VF)
	}
}
