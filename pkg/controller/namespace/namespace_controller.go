package namespace

import (
	"context"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	securityv1 "github.com/openshift/api/security/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_namespace")

// Add creates a new Namespace Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNamespace{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("namespace-controller", mgr, controller.Options{Reconciler: r})
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

// blank assignment to verify that ReconcileNamespace implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNamespace{}

// ReconcileNamespace reconciles a Namespace object
type ReconcileNamespace struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Namespace object and makes changes based on the state read
// and what is in the Namespace.Spec
func (r *ReconcileNamespace) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Name", request.Name)
	reqLogger.Info("Reconciling Namespace")

	// Fetch the MapRScc instance
	reqLogger.Info("Fetch the Namespace instance")
	instance := &corev1.Namespace{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Request object not found - not requeuing")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Info("Error reading the object - requeueing request")
		return reconcile.Result{}, err
	}

	// Reconcile SCC
	reqLogger.Info("Defining a new SCC object")
	scc := r.newSCCForNS(instance)
	reqLogger.Info("Checking if this SCC already exists")
	// Get the SCC instance
	sccfound := &securityv1.SecurityContextConstraints{}
	err = r.client.Get(context.TODO(), client.ObjectKey{Name: "mapr-" + instance.Name}, sccfound)
	if err != nil && errors.IsNotFound(err) {
		// Define a new SCC object
		reqLogger.Info("SCC not found - reating a new SCC")
		err = r.client.Create(context.TODO(), scc)
		if err != nil {
			return reconcile.Result{}, err
		}
		// SCC created successfully - don't requeue
		reqLogger.Info("SCC created successfully - not requeuing")
		return reconcile.Result{}, nil
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
		return reconcile.Result{}, nil
	}

	// Objects already exist - don't requeue
	reqLogger.Info("Skipping reconcile: objects already exist")
	return reconcile.Result{}, nil
}

// newSCCForNS returns an SCC with the name mapr-{namespace}
func (r *ReconcileNamespace) newSCCForNS(cr *corev1.Namespace) *securityv1.SecurityContextConstraints {
	labels := map[string]string{
		"namespace": cr.Name,
	}
	var uid int64 = 908000261
	var prio int32 = 42
	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "mapr-" + cr.Name,
			Labels: labels,
		},
		AllowPrivilegedContainer: false,
		AllowHostNetwork:         false,
		AllowHostPorts:           false,
		AllowHostPID:             false,
		AllowHostIPC:             false,
		Priority:                 &prio,
		FSGroup: securityv1.FSGroupStrategyOptions{
			Type: securityv1.FSGroupStrategyMustRunAs,
			Ranges: []securityv1.IDRange{
				securityv1.IDRange{
					Min: 908000261,
					Max: 908000262,
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
			UID:  &uid,
		},
		SELinuxContext: securityv1.SELinuxContextStrategyOptions{
			Type: securityv1.SELinuxStrategyMustRunAs,
		},
		SupplementalGroups: securityv1.SupplementalGroupsStrategyOptions{
			Type: securityv1.SupplementalGroupsStrategyRunAsAny,
			Ranges: []securityv1.IDRange{
				securityv1.IDRange{
					Min: 908000261,
					Max: 908000262,
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
		Users:  []string{"system:serviceaccount:" + cr.Name + ":default"},
		Groups: []string{"mapr-sas"},
	}
	controllerutil.SetControllerReference(cr, scc, r.scheme)
	return scc
}

// newSCCForNS only returns true if all fields except TypeMeta and ObjectMeta in sccfound and scc are equal
func equalSCCs(sccfound *securityv1.SecurityContextConstraints, scc *securityv1.SecurityContextConstraints) bool {
	return (cmp.Equal(sccfound, scc, cmpopts.IgnoreFields(securityv1.SecurityContextConstraints{}, "TypeMeta", "ObjectMeta")))
}
