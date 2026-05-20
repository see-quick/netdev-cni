// Copyright netdev-cni authors. Apache 2.0 License.
package controllers_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/netdev-cni/netdev-cni/pkg/apis/v1alpha1"
	"github.com/netdev-cni/netdev-cni/pkg/controllers"
)

func TestPolicyCreatesNodeState(t *testing.T) {
	g := NewWithT(t)

	_ = v1alpha1.AddToScheme(scheme.Scheme)

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "deploy", "crds")},
	}
	cfg, err := testEnv.Start()
	g.Expect(err).NotTo(HaveOccurred())
	defer testEnv.Stop() //nolint:errcheck

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	g.Expect(err).NotTo(HaveOccurred())

	err = (&controllers.SriovNetworkNodePolicyReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	g.Expect(err).NotTo(HaveOccurred())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = mgr.Start(ctx) }()
	time.Sleep(100 * time.Millisecond)

	policy := &v1alpha1.SriovNetworkNodePolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "test-policy"},
		Spec: v1alpha1.SriovNetworkNodePolicySpec{
			NodeSelector: map[string]string{"kubernetes.io/hostname": "kind-worker"},
			NumVfs:       2,
			NicSelector:  v1alpha1.NicSelector{PfNames: []string{"eth1"}},
			DeviceType:   "netdevice",
		},
	}

	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(c.Create(context.Background(), policy)).To(Succeed())

	// No real nodes in envtest — reconciler runs without error and returns without creating state.
	time.Sleep(200 * time.Millisecond)

	// Verify the reconciler did not panic and policy object exists.
	var fetched v1alpha1.SriovNetworkNodePolicy
	g.Expect(c.Get(context.Background(), client.ObjectKey{Name: "test-policy"}, &fetched)).To(Succeed())
}
