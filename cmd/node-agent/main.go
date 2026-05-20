// Copyright netdev-cni authors. Apache 2.0 License.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/netdev-cni/netdev-cni/pkg/agent"
	"github.com/netdev-cni/netdev-cni/pkg/apis/v1alpha1"
	"github.com/netdev-cni/netdev-cni/pkg/simulation"
	"go.uber.org/zap"
)

const (
	socketPath = "/var/run/netdev-cni/agent.sock"
	sriovSysfs = "/sys/class/net"
	numVfs     = 4
	vfPrefix   = "sim"
	deviceType = "netdevice"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync() //nolint:errcheck

	if err := os.MkdirAll("/var/run/netdev-cni", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	simMode := agent.IsSimulationMode(sriovSysfs)
	log.Info("node agent starting", zap.Bool("simulationMode", simMode))

	var vfs []v1alpha1.VFInfo
	if simMode {
		backend := simulation.NewVethBackend(vfPrefix)
		var err error
		vfs, err = backend.CreateVFs(numVfs)
		if err != nil {
			log.Fatal("create simulated VFs", zap.Error(err))
		}
		log.Info("created simulated VFs", zap.Int("count", len(vfs)))
	} else {
		log.Fatal("real SR-IOV mode not yet implemented")
	}

	pool := agent.NewPool(vfs)
	srv := agent.NewServer(socketPath, pool, deviceType)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info("listening on socket", zap.String("path", socketPath))
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("server error", zap.Error(err))
	}
}
