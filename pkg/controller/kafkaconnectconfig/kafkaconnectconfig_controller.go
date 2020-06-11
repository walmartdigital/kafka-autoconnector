package kafkaconnectconfig

import (
	"context"

	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/walmartdigital/go-kaya/pkg/kafkaconnect"
	skynetv1alpha1 "github.com/walmartdigital/kafka-autoconnector/pkg/apis/skynet/v1alpha1"
	"github.com/walmartdigital/kafka-autoconnector/pkg/cache"
	"github.com/walmartdigital/kafka-autoconnector/pkg/metrics"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	controllerName = "controller_kafkaconnectconfig"
	log            = logf.Log.WithName(controllerName)
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new KafkaConnectConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, c cache.Cache, m metrics.Metrics, kcf kafkaconnect.KafkaConnectClientFactory) error {
	return add(mgr, newReconciler(mgr, c, m))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c cache.Cache, m metrics.Metrics) reconcile.Reconciler {
	return &ReconcileKafkaConnectConfig{
		ReconcilerBase: util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(controllerName)),
		Cache:          c,
		Metrics:        m,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kafkaconnectconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KafkaConnectConfig
	err = c.Watch(&source.Kind{Type: &skynetv1alpha1.KafkaConnectConfig{}}, &handler.EnqueueRequestForObject{}, util.ResourceGenerationOrFinalizerChangedPredicate{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKafkaConnectConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKafkaConnectConfig{}

// ReconcileKafkaConnectConfig reconciles a KafkaConnectConfig object
type ReconcileKafkaConnectConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	util.ReconcilerBase
	cache.Cache
	metrics.Metrics
}

// Reconcile reads that state of the cluster for a KafkaConnectConfig object and makes changes based on the state read
// and what is in the KafkaConnectConfig.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKafkaConnectConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KafkaConnectConfig")

	// Fetch the KafkaConnectConfig instance
	instance := &skynetv1alpha1.KafkaConnectConfig{}
	err := r.GetClient().Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
