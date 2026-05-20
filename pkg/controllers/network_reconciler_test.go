// Copyright netdev-cni authors. Apache 2.0 License.
package controllers_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	netattachv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/netdev-cni/netdev-cni/pkg/apis/v1alpha1"
	"github.com/netdev-cni/netdev-cni/pkg/controllers"
)

func TestSriovNetworkCreatesNAD(t *testing.T) {
	g := NewWithT(t)

	_ = netattachv1.AddToScheme(scheme.Scheme)
	_ = v1alpha1.AddToScheme(scheme.Scheme)

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "deploy", "crds"),
		},
	}
	cfg, err := testEnv.Start()
	g.Expect(err).NotTo(HaveOccurred())
	defer testEnv.Stop() //nolint:errcheck

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	g.Expect(err).NotTo(HaveOccurred())

	err = (&controllers.SriovNetworkReconciler{Client: mgr.GetClient()}).SetupWithManager(mgr)
	g.Expect(err).NotTo(HaveOccurred())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = mgr.Start(ctx) }()
	time.Sleep(100 * time.Millisecond)

	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	g.Expect(err).NotTo(HaveOccurred())

	network := &v1alpha1.SriovNetwork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-net",
			Namespace: "default",
		},
		Spec: v1alpha1.SriovNetworkSpec{
			NetworkNamespace: "default",
			ResourceName:     "sriov.io/vf",
			IPAM:             `{"type":"host-local","subnet":"192.168.100.0/24"}`,
		},
	}
	g.Expect(c.Create(context.Background(), network)).To(Succeed())
	time.Sleep(500 * time.Millisecond)

	nad := &netattachv1.NetworkAttachmentDefinition{}
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "test-net",
		Namespace: "default",
	}, nad)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(nad.Spec.Config).To(ContainSubstring("netdev-cni"))
}
