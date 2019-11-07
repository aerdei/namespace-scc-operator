package namespacescc

import (
	"context"

	namespacesccv1alpha1 "github.com/aerdei/namespace-scc-operator/pkg/apis/namespacescc/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	securityv1 "github.com/openshift/api/security/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_namespacescc")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new NamespaceSCC Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNamespaceSCC{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("namespacescc-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NamespaceSCC
	err = c.Watch(&source.Kind{Type: &namespacesccv1alpha1.NamespaceSCC{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Namespace
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource SCCs and requeue the owner MapRScc
	err = c.Watch(&source.Kind{Type: &securityv1.SecurityContextConstraints{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &corev1.Namespace{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNamespaceSCC implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNamespaceSCC{}

// ReconcileNamespaceSCC reconciles a NamespaceSCC object
type ReconcileNamespaceSCC struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a NamespaceSCC object and makes changes based on the state read
// and what is in the NamespaceSCC.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNamespaceSCC) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NamespaceSCC")

	// Fetch the NamespaceSCC CR list
	sccList := &namespacesccv1alpha1.NamespaceSCCList{}
	if err := r.client.List(context.TODO(), sccList); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Fetch the Namespace list
	nsList := &corev1.NamespaceList{}
	if err := r.client.List(context.TODO(), nsList); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	// For every SCC, check if every namespace has a corresponding scc. If not, create one. If present but differs, update.
	for _, sccElement := range sccList.Items {
		for _, nsElement := range nsList.Items {
			whiteListedNamespace := false
			for _, whiteListed := range sccElement.Spec.WhiteList {
				if nsElement.Name == whiteListed {
					whiteListedNamespace = true
				}
			}
			if !whiteListedNamespace {
				// Reconcile SCC
				reqLogger.Info("Defining a new SCC object")
				scc := r.newSCCForNS(&sccElement, &nsElement)
				reqLogger.Info("Checking if this SCC already exists")
				// Get the SCC instance
				sccfound := &securityv1.SecurityContextConstraints{}
				if err := r.client.Get(context.TODO(), client.ObjectKey{Name: "mapr-" + nsElement.Name}, sccfound); err != nil && errors.IsNotFound(err) {
					// Define a new SCC object
					reqLogger.Info("SCC not found - reating a new SCC")
					err = r.client.Create(context.TODO(), scc)
					if err != nil {
						return reconcile.Result{}, err
					}
					// SCC created successfully - don't requeue
					reqLogger.Info("SCC created successfully - not requeuing")
					//return reconcile.Result{}, nil
				} else if err != nil {
					reqLogger.Info("SCC get error - requeueing")
					return reconcile.Result{}, err
				} else if !equalSCCs(sccfound, scc) {
					reqLogger.Info("SCC mismatch - updating")
					sccfound = scc
					err = r.client.Update(context.TODO(), sccfound)
					if err != nil {
						reqLogger.Error(err, "Failed to update SCC")
						return reconcile.Result{}, err
					}
					//return reconcile.Result{}, nil
				}
			}
		}
	}
	// Pod already exists - don't requeue
	return reconcile.Result{}, nil
}

// newSCCForNS returns an SCC with the name mapr-{namespace}
func (r *ReconcileNamespaceSCC) newSCCForNS(cr *namespacesccv1alpha1.NamespaceSCC, ns *corev1.Namespace) *securityv1.SecurityContextConstraints {
	labels := map[string]string{
		"namespace": ns.Name,
	}
	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "mapr-" + ns.Name,
			Labels: labels,
		},
		AllowPrivilegedContainer: false,
		AllowHostNetwork:         false,
		AllowHostPorts:           false,
		AllowHostPID:             false,
		AllowHostIPC:             false,
		Priority:                 &cr.Spec.SccPriority,
		FSGroup: securityv1.FSGroupStrategyOptions{
			Type: securityv1.FSGroupStrategyMustRunAs,
			Ranges: []securityv1.IDRange{
				securityv1.IDRange{
					Min: cr.Spec.UUID,
					Max: cr.Spec.UUID,
				},
			},
		},
		ReadOnlyRootFilesystem: false,
		RequiredDropCapabilities: []corev1.Capability{
			"KILL",
			"MKNOD",
			"SETUID",
			"SETGID",
		},
		RunAsUser: securityv1.RunAsUserStrategyOptions{
			Type: securityv1.RunAsUserStrategyMustRunAs,
			UID:  &cr.Spec.UUID,
		},
		SELinuxContext: securityv1.SELinuxContextStrategyOptions{
			Type: securityv1.SELinuxStrategyMustRunAs,
		},
		SupplementalGroups: securityv1.SupplementalGroupsStrategyOptions{
			Type: securityv1.SupplementalGroupsStrategyRunAsAny,
			Ranges: []securityv1.IDRange{
				securityv1.IDRange{
					Min: cr.Spec.UUID,
					Max: cr.Spec.UUID,
				},
			},
		},
		Volumes: []securityv1.FSType{
			securityv1.FSTypeConfigMap,
			securityv1.FSTypeDownwardAPI,
			securityv1.FSTypeEmptyDir,
			securityv1.FSTypePersistentVolumeClaim,
			securityv1.FSProjected,
			securityv1.FSTypeSecret,
		},
		Users:  []string{"system:serviceaccount:" + ns.Name + ":default"},
		Groups: []string{"mapr-sas"},
	}
	controllerutil.SetControllerReference(ns, scc, r.scheme)
	return scc
}

// newSCCForNS only returns true if all fields except TypeMeta and ObjectMeta in sccfound and scc are equal
func equalSCCs(sccfound *securityv1.SecurityContextConstraints, scc *securityv1.SecurityContextConstraints) bool {
	return (cmp.Equal(sccfound, scc, cmpopts.IgnoreFields(securityv1.SecurityContextConstraints{}, "TypeMeta", "ObjectMeta")))
}
