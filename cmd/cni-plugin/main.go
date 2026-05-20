//go:build linux

// Copyright netdev-cni authors. Apache 2.0 License.
package main

import (
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/netdev-cni/netdev-cni/pkg/cni"
)

func main() {
	skel.PluginMain(
		cni.CmdAdd,
		cni.CmdCheck,
		cni.CmdDel,
		version.All,
		"netdev-cni v0.1.0",
	)
}
