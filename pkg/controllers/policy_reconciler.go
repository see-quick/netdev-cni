// Copyright netdev-cni authors. Apache 2.0 License.
package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/netdev-cni/netdev-cni/pkg/apis/v1alpha1"
)

// SriovNetworkNodePolicyReconciler reconciles SriovNetworkNodePolicy objects.
type SriovNetworkNodePolicyReconciler struct {
	client.Client
}

func (r *SriovNetworkNodePolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SriovNetworkNodePolicy{}).
		Complete(r)
}

func (r *SriovNetworkNodePolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	policy := &v1alpha1.SriovNetworkNodePolicy{}
	if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	nodeList := &corev1.NodeList{}
	selector := labels.SelectorFromSet(policy.Spec.NodeSelector)
	if err := r.List(ctx, nodeList, &client.ListOptions{LabelSelector: selector}); err != nil {
		return ctrl.Result{}, fmt.Errorf("list nodes: %w", err)
	}

	for _, node := range nodeList.Items {
		if err := r.ensureNodeState(ctx, policy, node.Name); err != nil {
			log.Error(err, "ensure node state", "node", node.Name)
		}
	}
	return ctrl.Result{}, nil
}

func (r *SriovNetworkNodePolicyReconciler) ensureNodeState(
	ctx context.Context, policy *v1alpha1.SriovNetworkNodePolicy, nodeName string,
) error {
	state := &v1alpha1.SriovNetworkNodeState{}
	err := r.Get(ctx, client.ObjectKey{Name: nodeName}, state)
	if errors.IsNotFound(err) {
		state = &v1alpha1.SriovNetworkNodeState{
			ObjectMeta: metav1.ObjectMeta{Name: nodeName},
			Spec: v1alpha1.SriovNetworkNodeStateSpec{
				Interfaces: []v1alpha1.InterfaceSpec{
					{
						Name:       policy.Spec.NicSelector.PfNames[0],
						NumVfs:     policy.Spec.NumVfs,
						DeviceType: policy.Spec.DeviceType,
					},
				},
			},
		}
		return r.Create(ctx, state)
	}
	if err != nil {
		return err
	}
	state.Spec.Interfaces = []v1alpha1.InterfaceSpec{
		{
			Name:       policy.Spec.NicSelector.PfNames[0],
			NumVfs:     policy.Spec.NumVfs,
			DeviceType: policy.Spec.DeviceType,
		},
	}
	return r.Update(ctx, state)
}
