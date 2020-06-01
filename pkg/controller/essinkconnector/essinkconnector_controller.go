package essinkconnector

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/walmartdigital/go-kaya/pkg/client"
	"github.com/walmartdigital/go-kaya/pkg/kafkaconnect"
	"github.com/walmartdigital/go-kaya/pkg/validator"
	"github.com/walmartdigital/kafka-autoconnector/pkg/apis/skynet/v1alpha1"
	skynetv1alpha1 "github.com/walmartdigital/kafka-autoconnector/pkg/apis/skynet/v1alpha1"
	"github.com/walmartdigital/kafka-autoconnector/pkg/cache"
	"github.com/walmartdigital/kafka-autoconnector/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	controllerName             = "controller_essinkconnector"
	log                        = logf.Log.WithName(controllerName)
	kafkaConnectHost           = "192.168.64.5:30256"
	connectorRestartCachePath  = "/essinkconnector/connectors/%s/restart"
	taskRestartCachePath       = "/essinkconnector/connectors/%s/tasks/%d/restart"
	totalTasksCountCachePath   = "/essinkconnector/connectors/%s/tasks/total/count"
	runningTasksCountCachePath = "/essinkconnector/connectors/%s/tasks/running/count"
	opt                        cmp.Option
)

func init() {
	// This should probably be moved to the go-kaya repo for domain/cohesion purposes
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
func Add(mgr manager.Manager, c cache.Cache, kcf kafkaconnect.KafkaConnectClientFactory) error {
	return add(mgr, newReconciler(mgr, c, kcf))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c cache.Cache, kcf kafkaconnect.KafkaConnectClientFactory) reconcile.Reconciler {
	return &ReconcileESSinkConnector{
		ReconcilerBase:            util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(controllerName)),
		KafkaConnectClientFactory: kcf,
		Cache:                     c,
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
	err = c.Watch(&source.Kind{Type: &skynetv1alpha1.ESSinkConnector{}}, &handler.EnqueueRequestForObject{}, util.ResourceGenerationOrFinalizerChangedPredicate{})
	// err = c.Watch(&source.Kind{Type: &skynetv1alpha1.ESSinkConnector{}}, &handler.EnqueueRequestForObject{})
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
	cache.Cache
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

func (r *ReconcileESSinkConnector) CheckAndHealConnector(connector *skynetv1alpha1.ESSinkConnector, kcc kafkaconnect.KafkaConnectClient) (bool, error) {
	var connectorRestartCount, taskRestartCount int
	var healthy bool

	response, readErr := kcc.GetStatus(connector.Spec.Config.Name)

	if readErr != nil {
		return false, readErr
	}

	status := response.Payload.(kafkaconnect.Status)

	runningTasksCount := status.GetActiveTasksCount()
	totalTaskCount := status.GetTaskCount()
	r.Cache.Store(fmt.Sprintf(totalTasksCountCachePath, connector.Spec.Config.Name), totalTaskCount)
	r.Cache.Store(fmt.Sprintf(runningTasksCountCachePath, connector.Spec.Config.Name), runningTasksCount)

	if status.IsConnectorFailed() {
		response, error := kcc.RestartConnector(connector.Spec.Config.Name)

		if error != nil {
			return false, error
		}

		if response.Result == "success" {
			log.Info(fmt.Sprintf("Failed connector %s is being restarted", connector.Spec.Config.Name))
			value, ok := r.Cache.Load(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config.Name))

			if ok {
				connectorRestartCount = value.(int)
				connectorRestartCount++
				r.Cache.Store(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config.Name), connectorRestartCount)
			} else {
				connectorRestartCount = 1
				r.Cache.Store(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config.Name), connectorRestartCount)
			}
			log.Info(fmt.Sprintf("Restart count for connector %s is now %d", connector.Spec.Config.Name, connectorRestartCount))
			return false, nil
		} else {
			return false, fmt.Errorf("Error occurred restarting connector %s", connector.Spec.Config.Name)
		}
	} else {
		healthy = true
		for i := 0; i < totalTaskCount; i++ {
			failed, err := status.IsTaskFailed(i)
			if err != nil {
				return false, err
			}
			if failed {
				healthy = false
				response, error := kcc.RestartTask(connector.Spec.Config.Name, i)

				if error != nil {
					return false, error
				}

				if response.Result == "success" {
					log.Info(fmt.Sprintf("Failed task %d for connector %s is being restarted", i, connector.Spec.Config.Name))
					value, ok := r.Cache.Load(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config.Name, i))

					if ok {
						taskRestartCount = value.(int)
						taskRestartCount++
						r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config.Name, i), taskRestartCount)
					} else {
						taskRestartCount = 1
						r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config.Name, i), taskRestartCount)
					}
					log.Info(fmt.Sprintf("Restart count for task %d of connector %s is now %d", i, connector.Spec.Config.Name, taskRestartCount))
				} else {
					return false, fmt.Errorf("Error occurred restarting connector %s", connector.Spec.Config.Name)
				}
			} else {
				r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config.Name, i), 0)
			}
		}
		return healthy, nil
	}
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

	response, readErr := kcc.Read(config.Name)

	if readErr != nil {
		return readErr
	}

	if response.Result == "success" {
		log.Info(fmt.Sprintf("Connector %s already exists, updating configuration", config.Name))

		if !cmp.Equal(connector.Spec.Config, response.Payload, opt) {
			resp, updateErr := kcc.Update(conObj)
			if resp.Result == "success" {
				return nil
			} else {
				return updateErr
			}
		} else {
			healthy, err := r.CheckAndHealConnector(connector, kcc)
			if err != nil {
				return err
			}
			if healthy {
				log.Info(fmt.Sprintf("Connector %s is healthy", config.Name))
			} else {
				log.Info(fmt.Sprintf("Connector %s is unhealthy, self-healing in progress", config.Name))
			}
		}
	} else {
		log.Info(fmt.Sprintf("Failed to get connector %s", config.Name))
		resp3, createErr := kcc.Create(conObj)
		if resp3.Result == "success" {
			return nil
		} else {
			return createErr
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
	log.Info("Executing ManageSuccess")

	runtimeObj, ok := (obj).(runtime.Object)
	if !ok {
		log.Error(errors.New("not a runtime.Object"), "passed object was not a runtime.Object", "object", obj)
		return reconcile.Result{}, nil
	}
	if connector, updateStatus := obj.(*skynetv1alpha1.ESSinkConnector); updateStatus {
		status := v1alpha1.ESSinkConnectorStatus{
			ConnectorName: connector.Spec.Config.Name,
			Topics:        connector.Spec.Config.Topics,
			Status:        "Running",
			LastUpdate:    metav1.Now(),
			Tasks:         0,
		}
		connector.SetStatus(status)
		err := r.GetClient().Status().Update(context.Background(), runtimeObj)
		if err != nil {
			log.Error(err, "unable to update status")
			return reconcile.Result{
				RequeueAfter: time.Second,
				Requeue:      true,
			}, nil
		}
	} else {
		log.Info("object is not RecocileStatusAware, not setting status")
	}
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

func (r *ReconcileESSinkConnector) ManageError(obj metav1.Object, issue error) (reconcile.Result, error) {
	log.Info("Calling ManageError")

	//TODO: What is an error condition in the case of an ESSinkConnector?
	// A number of active tasks that is less than a certain threshold relative to the max number of tasks
	// A connector that is not running (get from status endpoint)
	// Can we get any other information from the tasks endpoint to detect any deviations from the desired state?

	runtimeObj, ok := (obj).(runtime.Object)

	if !ok {
		log.Error(errors.New("not a runtime.Object"), "passed object was not a runtime.Object", "object", obj)
		return reconcile.Result{}, nil
	}

	var retryInterval time.Duration

	r.GetRecorder().Event(runtimeObj, "Warning", "ProcessingError", issue.Error())

	if connector, updateStatus := obj.(*skynetv1alpha1.ESSinkConnector); updateStatus {
		lastUpdate := connector.GetStatus().LastUpdate.Time
		lastStatus := connector.GetStatus().Status

		status := v1alpha1.ESSinkConnectorStatus{
			ConnectorName: connector.Spec.Config.Name,
			Topics:        connector.Spec.Config.Topics,
			Tasks:         0,
			LastUpdate:    metav1.Now(),
			Reason:        issue.Error(),
			Status:        "Failure",
		}

		connector.SetStatus(status)

		err := r.GetClient().Status().Update(context.Background(), runtimeObj)

		if err != nil {
			log.Error(err, "unable to update status")
			return reconcile.Result{
				RequeueAfter: time.Second,
				Requeue:      true,
			}, nil
		}

		if lastUpdate.IsZero() || lastStatus == "Success" {
			retryInterval = time.Second
		} else {
			retryInterval = status.LastUpdate.Sub(lastUpdate).Round(time.Second)
		}
	} else {
		log.Info("object is not RecocileStatusAware, not setting status")
		retryInterval = time.Second
	}

	return reconcile.Result{
		RequeueAfter: time.Duration(math.Min(float64(retryInterval.Nanoseconds()*2), float64(time.Hour.Nanoseconds()*6))),
		Requeue:      true,
	}, nil
}
