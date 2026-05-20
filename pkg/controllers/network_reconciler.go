// Copyright netdev-cni authors. Apache 2.0 License.
package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	netattachv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
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
	// Register NAD type with the scheme so the client can handle it.
	if err := netattachv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
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

	nad := &netattachv1.NetworkAttachmentDefinition{}
	err = r.Get(ctx, client.ObjectKey{
		Name:      network.Name,
		Namespace: network.Spec.NetworkNamespace,
	}, nad)

	if errors.IsNotFound(err) {
		nad = &netattachv1.NetworkAttachmentDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      network.Name,
				Namespace: network.Spec.NetworkNamespace,
				Labels:    map[string]string{"netdev.io/managed-by": "netdev-cni"},
			},
			Spec: netattachv1.NetworkAttachmentDefinitionSpec{Config: nadConfig},
		}
		log.Info("creating NetworkAttachmentDefinition", "name", network.Name)
		return ctrl.Result{}, r.Create(ctx, nad)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	// Skip NADs owned by something else.
	if v, ok := nad.Labels["netdev.io/managed-by"]; !ok || v != "netdev-cni" {
		log.Info("NAD exists with different owner, skipping", "name", network.Name)
		return ctrl.Result{}, nil
	}

	nad.Spec.Config = nadConfig
	return ctrl.Result{}, r.Update(ctx, nad)
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
