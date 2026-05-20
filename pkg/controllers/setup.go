// Copyright netdev-cni authors. Apache 2.0 License.
package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupWithManager registers all reconcilers with the manager.
func SetupWithManager(mgr manager.Manager) error {
	if err := (&SriovNetworkNodePolicyReconciler{Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&SriovNetworkReconciler{Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
		return err
	}
	return nil
}
