package essinkconnector

import (
	"context"
	"errors"
	"fmt"

	"github.com/chinniehendrix/go-kaya/pkg/client"
	"github.com/chinniehendrix/go-kaya/pkg/kafkaconnect"
	"github.com/chinniehendrix/go-kaya/pkg/validator"
	"github.com/google/go-cmp/cmp"
	"github.com/redhat-cop/operator-utils/pkg/util"
	skynetv1alpha1 "github.com/walmartdigital/kafka-autoconnector/pkg/apis/skynet/v1alpha1"
	"github.com/walmartdigital/kafka-autoconnector/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var controllerName = "controller_essinkconnector"
var log = logf.Log.WithName(controllerName)
var kafkaConnectHost = "192.168.64.5:30256"

var opt cmp.Option

func init() {
	opt = cmp.Comparer(func(a, b kafkaconnect.ConnectorConfig) bool {
		if a == (kafkaconnect.ConnectorConfig{}) && b == (kafkaconnect.ConnectorConfig{}) {
			return true
		} else if a != (kafkaconnect.ConnectorConfig{}) && b != (kafkaconnect.ConnectorConfig{}) {
			if a.Name == b.Name &&
				a.ConnectorClass == b.ConnectorClass &&
				a.DocumentType == b.DocumentType &&
				a.Topics == b.Topics &&
				a.TopicIndexMap == b.TopicIndexMap &&
				a.BatchSize == b.BatchSize &&
				a.ConnectionURL == b.ConnectionURL &&
				a.KeyIgnore == b.KeyIgnore &&
				a.SchemaIgnore == b.SchemaIgnore &&
				a.BehaviorOnMalformedDocuments == b.BehaviorOnMalformedDocuments {
				return true
			}
		}
		return false
	})
}

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ESSinkConnector Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, factory kafkaconnect.KafkaConnectClientFactory) error {
	return add(mgr, newReconciler(mgr, factory))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, factory kafkaconnect.KafkaConnectClientFactory) reconcile.Reconciler {
	return &ReconcileESSinkConnector{
		ReconcilerBase:            util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(controllerName)),
		KafkaConnectClientFactory: factory,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("essinkconnector-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ESSinkConnector
	// err = c.Watch(&source.Kind{Type: &examplev1alpha1.MyCRD{}}, &handler.EnqueueRequestForObject{}, util.ResourceGenerationOrFinalizerChangedPredicate{})
	err = c.Watch(&source.Kind{Type: &skynetv1alpha1.ESSinkConnector{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileESSinkConnector implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileESSinkConnector{}

// ReconcileESSinkConnector reconciles a ESSinkConnector object
type ReconcileESSinkConnector struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	util.ReconcilerBase
	kafkaconnect.KafkaConnectClientFactory
}

// Reconcile reads that state of the cluster for a ESSinkConnector object and makes changes based on the state read
// and what is in the ESSinkConnector.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileESSinkConnector) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ESSinkConnector")

	// Fetch the ESSinkConnector instance
	instance := &skynetv1alpha1.ESSinkConnector{}
	err := r.GetClient().Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if ok := r.IsInitialized(instance); !ok {
		log.Info("CR has not been initialized yet")
		err := r.GetClient().Update(context.TODO(), instance)
		if err != nil {
			log.Error(err, "unable to update instance", "instance", instance)
			return r.ManageError(instance, err)
		}
		return reconcile.Result{}, nil
	}

	if ok, err := r.IsValid(instance); !ok {
		return r.ManageError(instance, err)
	}

	kcc, err0 := r.KafkaConnectClientFactory.Create(kafkaConnectHost, client.RestyClientFactory{})

	if err0 != nil {
		// TODO: need to understand how to convey outcome as part of Result struct
		return reconcile.Result{}, err0
	}

	if util.IsBeingDeleted(instance) {
		log.Info("CR is being deleted")
		if !util.HasFinalizer(instance, controllerName) {
			return reconcile.Result{}, nil
		}
		err := r.ManageCleanUpLogic(instance, kcc)
		if err != nil {
			log.Error(err, "unable to delete instance", "instance", instance)
			return r.ManageError(instance, err)
		}
		util.RemoveFinalizer(instance, controllerName)
		err = r.GetClient().Update(context.TODO(), instance)
		if err != nil {
			log.Error(err, "unable to update instance", "instance", instance)
			return r.ManageError(instance, err)
		}
		return reconcile.Result{}, nil
	}

	err = r.ManageOperatorLogic(instance, kcc)
	if err != nil {
		return r.ManageError(instance, err)
	}
	return r.ManageSuccess(instance)
}

func (r *ReconcileESSinkConnector) ManageOperatorLogic(obj metav1.Object, kcc kafkaconnect.KafkaConnectClient) error {
	log.Info("Calling ManageOperatorLogic")
	connector, ok := obj.(*skynetv1alpha1.ESSinkConnector)

	if !ok {
		return errors.New("Object is not of type ESSinkConnector")
	}

	config := connector.Spec.Config

	if config == (kafkaconnect.ConnectorConfig{}) {
		return errors.New("Could not get configuration from ESSinkConnector")
	}

	conObj := kafkaconnect.Connector{
		Name:   connector.Spec.Config.Name,
		Config: &connector.Spec.Config,
	}

	response, _ := kcc.Read(config.Name)

	if response.Result == "success" {
		log.Info(fmt.Sprintf("Connector %s already exists, updating configuration", config.Name))

		if !cmp.Equal(connector.Spec.Config, response.Config, opt) {
			resp, err2 := kcc.Update(conObj)
			if resp.Result == "success" {
				return nil
			} else {
				return err2
			}
		}
	} else {
		log.Info(fmt.Sprintf("Failed to get connector %s", config.Name))
		resp3, err3 := kcc.Create(conObj)
		if resp3.Result == "success" {
			return nil
		} else {
			return err3
		}
	}
	return nil
}

func (r *ReconcileESSinkConnector) IsValid(obj metav1.Object) (bool, error) {
	log.Info("Validating CR")

	connector, ok := obj.(*skynetv1alpha1.ESSinkConnector)
	v := validator.New()

	if !ok {
		return false, errors.New("Object is not of type ESSinkConnector")
	} else {
		config := connector.Spec.Config

		err := v.Struct(config)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *ReconcileESSinkConnector) IsInitialized(obj metav1.Object) bool {
	log.Info("Checking if CR is initialized")
	initialized := true

	connector, ok := obj.(*skynetv1alpha1.ESSinkConnector)

	if !ok {
		initialized = false
		log.Info("Could not retrieve ESSinkConnector")
	} else {
		if !util.HasFinalizer(connector, controllerName) {
			log.Info("Adding finalizer to ESSinkConnector")
			controllerutil.AddFinalizer(connector, controllerName)
			initialized = false
		}

		if connector.Spec.Config.Name == "" {
			connector.Spec.Config.Name = connector.ObjectMeta.Name + "_" + utils.RandSeq(8)
			return false
		}
	}

	return initialized
}

func (r *ReconcileESSinkConnector) ManageSuccess(obj metav1.Object) (reconcile.Result, error) {
	log.Info("Calling ManageSuccess")
	return reconcile.Result{}, nil
}

func (r *ReconcileESSinkConnector) ManageCleanUpLogic(obj metav1.Object, kcc kafkaconnect.KafkaConnectClient) error {
	log.Info("Running clean up logic on ESSinkConnector")
	connector, ok := obj.(*skynetv1alpha1.ESSinkConnector)

	if !ok {
		return errors.New("Object is not of type ESSinkConnector")
	}

	config := connector.Spec.Config

	if config == (kafkaconnect.ConnectorConfig{}) {
		return errors.New("Could not get configuration from ESSinkConnector")
	}

	if config.Name == "" {
		return errors.New("Connector name is not set")
	}

	response, err := kcc.Delete(config.Name)

	if response.Result == "success" {
		log.Info(fmt.Sprintf("Connector %s deleted successfully", config.Name))
		return nil
	} else {
		log.Info(fmt.Sprintf("Failed to delete connector %s", config.Name))
		return err
	}
}

func (r *ReconcileESSinkConnector) ManageError(obj metav1.Object, err error) (reconcile.Result, error) {
	log.Info("Calling ManageError")
	return reconcile.Result{}, nil
}
