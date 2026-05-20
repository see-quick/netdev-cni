// Copyright netdev-cni authors. Apache 2.0 License.
package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/netdev-cni/netdev-cni/pkg/apis/v1alpha1"
)

// SriovNetworkReconciler reconciles SriovNetwork objects.
type SriovNetworkReconciler struct {
	client.Client
}

func (r *SriovNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SriovNetwork{}).
		Complete(r)
}

func (r *SriovNetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	network := &v1alpha1.SriovNetwork{}
	if err := r.Get(ctx, req.NamespacedName, network); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	nadConfig, err := buildNADConfig(network)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("build NAD config: %w", err)
	}

	// Store NAD config as an annotation on the SriovNetwork until the Multus NAD
	// CRD is available in the cluster (Task 11 adds the full NAD reconciliation).
	if network.Annotations == nil {
		network.Annotations = map[string]string{}
	}
	existing, notFound := network.Annotations["netdev.io/nad-config"], errors.IsNotFound(nil)
	_ = notFound
	if existing == nadConfig {
		return ctrl.Result{}, nil
	}
	network.Annotations["netdev.io/nad-config"] = nadConfig
	log.Info("updated NAD config annotation", "network", network.Name)
	return ctrl.Result{}, r.Update(ctx, network)
}

type nadCNIConfig struct {
	CniVersion   string          `json:"cniVersion"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	IPAM         json.RawMessage `json:"ipam"`
	Capabilities struct {
		DPDK bool `json:"dpdk,omitempty"`
		RoCE bool `json:"roce,omitempty"`
	} `json:"capabilities,omitempty"`
}

func buildNADConfig(network *v1alpha1.SriovNetwork) (string, error) {
	cfg := nadCNIConfig{
		CniVersion: "1.0.0",
		Name:       network.Name,
		Type:       "netdev-cni",
		IPAM:       json.RawMessage(network.Spec.IPAM),
	}
	cfg.Capabilities.DPDK = network.Spec.Capabilities.DPDK
	cfg.Capabilities.RoCE = network.Spec.Capabilities.RoCE

	b, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// nadMeta is used when creating NAD objects (added in Task 11 with Multus dep).
type nadMeta struct {
	metav1.ObjectMeta
}
