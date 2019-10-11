package e2e

import (
	goctx "context"
	"testing"
	"time"

	securityv1 "github.com/openshift/api/security/v1"
	f "github.com/operator-framework/operator-sdk/pkg/test"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 60
)

func TestSccOperator(t *testing.T) {
	// run subtests
	t.Run("scc-group", func(t *testing.T) {
		t.Run("Cluster", SccOperatorCluster)
	})
}

func sccHandlingTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {

	exampleNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "example-scc-op-test",
			Annotations: map[string]string{
				"openshift.io/description":                "",
				"openshift.io/display-name":               "",
				"openshift.io/requester":                  "system:admin",
				"openshift.io/sa.scc.mcs":                 "s0:c13,c12",
				"openshift.io/sa.scc.supplemental-groups": "1000180000/10000",
				"openshift.io/sa.scc.uid-range":           "1000180000/10000",
			},
			Labels: map[string]string{
				"usernamespace": "true",
			},
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{
				corev1.FinalizerKubernetes,
			},
		},
	}
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err := f.Client.Create(goctx.TODO(), exampleNamespace, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	// get the namespace
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: "example-scc-op-test", Namespace: ""}, exampleNamespace)
	if err != nil {
		return err
	}

	// get the scc to see if it was created
	scc := &securityv1.SecurityContextConstraints{}
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: "mapr-example-scc-op-test", Namespace: ""}, scc)
		if err != nil {
			if errors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s scc\n", scc.Name)
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	// modify the scc to see it gets updated back to its original state
	scc.AllowHostNetwork = true
	err = f.Client.Update(goctx.TODO(), scc)
	if err != nil {
		return err
	}

	// poll the scc to see if it gets reconciled
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: "mapr-example-scc-op-test", Namespace: ""}, scc)
		if err != nil {
			if errors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s scc\n", scc.Name)
				return false, nil
			}
			return false, err
		} else if scc.AllowHostNetwork == true {
			t.Logf("Waiting for reconciliation of %s scc\n", scc.Name)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	// delete namespace to see if scc is deleted
	err = f.Client.Delete(goctx.TODO(), exampleNamespace)
	if err != nil {
		return err
	}

	// poll the scc to see if it gets deleted
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: "mapr-example-scc-op-test", Namespace: ""}, scc)
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		t.Logf("Waiting for deletion of %s scc\n", scc.Name)
		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func SccOperatorCluster(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)

	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}

	sccl := &securityv1.SecurityContextConstraintsList{}
	err = f.AddToFrameworkScheme(securityv1.AddToScheme, sccl)
	if err != nil {
		t.Fatalf("Could not add scc to scheme: %v", err)
	}

	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	// get global framework variables
	f := framework.Global
	// wait for operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "default", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if err = sccHandlingTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}

}
