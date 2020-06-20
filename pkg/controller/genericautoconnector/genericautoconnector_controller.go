package genericautoconnector

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/walmartdigital/go-kaya/pkg/client"
	"github.com/walmartdigital/go-kaya/pkg/kafkaconnect"
	"github.com/walmartdigital/go-kaya/pkg/validator"
	"github.com/walmartdigital/kafka-autoconnector/pkg/apis/skynet/v1alpha1"
	skynetv1alpha1 "github.com/walmartdigital/kafka-autoconnector/pkg/apis/skynet/v1alpha1"
	"github.com/walmartdigital/kafka-autoconnector/pkg/cache"
	operatorconfig "github.com/walmartdigital/kafka-autoconnector/pkg/config"
	"github.com/walmartdigital/kafka-autoconnector/pkg/metrics"
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
	controllerName                     = "controller_genericautoconnector"
	log                                = logf.Log.WithName(controllerName)
	connectorLastHealthyCheckCachePath = "/genericautoconnector/connectors/%s/lasthealthycheck"
	connectorRestartCachePath          = "/genericautoconnector/connectors/%s/restart"
	taskRestartCachePath               = "/genericautoconnector/connectors/%s/tasks/%d/restart"
	connectorHardResetCachePath        = "/genericautoconnector/connectors/%s/hardreset"
	totalTasksCountCachePath           = "/genericautoconnector/connectors/%s/tasks/total/count"
	runningTasksCountCachePath         = "/genericautoconnector/connectors/%s/tasks/running/count"
)

// Add creates a new GenericAutoConnector Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, c cache.Cache, m metrics.Metrics, kcf kafkaconnect.KafkaConnectClientFactory) error {
	return add(mgr, newReconciler(mgr, c, m, kcf))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c cache.Cache, m metrics.Metrics, kcf kafkaconnect.KafkaConnectClientFactory) reconcile.Reconciler {
	return &ReconcileGenericAutoConnector{
		ReconcilerBase:            util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(controllerName)),
		KafkaConnectClientFactory: kcf,
		Cache:                     c,
		Metrics:                   m,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("genericautoconnector-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource GenericAutoConnector
	err = c.Watch(&source.Kind{Type: &skynetv1alpha1.GenericAutoConnector{}}, &handler.EnqueueRequestForObject{}, util.ResourceGenerationOrFinalizerChangedPredicate{})
	//err = c.Watch(&source.Kind{Type: &skynetv1alpha1.GenericAutoConnector{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileGenericAutoConnector implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileGenericAutoConnector{}

// ReconcileGenericAutoConnector reconciles a GenericAutoConnector object
type ReconcileGenericAutoConnector struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	util.ReconcilerBase
	kafkaconnect.KafkaConnectClientFactory
	cache.Cache
	metrics.Metrics
}

// Reconcile reads that state of the cluster for a GenericAutoConnector object and makes changes based on the state read
// and what is in the GenericAutoConnector.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGenericAutoConnector) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling GenericAutoConnector")

	// Fetch the GenericAutoConnector instance
	instance := &skynetv1alpha1.GenericAutoConnector{}
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
		log.Info(fmt.Sprintf("CR %s has not been initialized yet", instance.ObjectMeta.Name))
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

	kafkaConnectHost, cacheErr := operatorconfig.GetKafkaConnectAddress(r.Cache)

	if cacheErr != nil {
		return reconcile.Result{}, cacheErr
	}

	kcc, err0 := r.KafkaConnectClientFactory.Create(kafkaConnectHost, client.RestyClientFactory{})

	if err0 != nil {
		return reconcile.Result{}, err0
	}

	if util.IsBeingDeleted(instance) {
		log.Info(fmt.Sprintf("CR %s is being deleted", instance.ObjectMeta.Name))
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

// HardResetConnector ...
func (r *ReconcileGenericAutoConnector) HardResetConnector(connector *skynetv1alpha1.GenericAutoConnector, kcc kafkaconnect.KafkaConnectClient) error {
	var connectorHardResetCount int
	value, ok := r.Cache.Load(fmt.Sprintf(connectorHardResetCachePath, connector.Spec.Config["name"]))

	maxConnectorHardResets, cacheErr := operatorconfig.GetMaxConnectorHardResets(r.Cache)

	if cacheErr != nil {
		return cacheErr
	}

	if ok {
		connectorHardResetCount = value.(int)
	} else {
		connectorHardResetCount = 0
	}

	if connectorHardResetCount < maxConnectorHardResets {
		resp0, err0 := kcc.Delete(connector.Spec.Config["name"])

		if err0 != nil {
			log.Error(err0, fmt.Sprintf("Failed to delete connector %s", connector.Spec.Config["name"]))
			return err0
		}

		if resp0.Result == "success" {
			log.Error(err0, fmt.Sprintf("Successfully deleted connector %s", connector.Spec.Config["name"]))
			conObj := kafkaconnect.Connector{
				Name:   connector.Spec.Config["name"],
				Config: connector.Spec.Config,
			}

			resp1, err1 := kcc.Create(conObj)

			if err0 != nil {
				log.Error(err1, fmt.Sprintf("Failed to recreate connector %s", connector.Spec.Config["name"]))
				return err1
			}

			if resp1.Result == "success" {
				connectorHardResetCount++
				r.Cache.Store(fmt.Sprintf(connectorHardResetCachePath, connector.Spec.Config["name"]), connectorHardResetCount)
				log.V(1).Info(fmt.Sprintf("Hard reset count for connector %s is now %d", connector.Spec.Config["name"], connectorHardResetCount))
				return nil
			}
			return fmt.Errorf("Failed to recreate connector %s", connector.Spec.Config["name"])
		}
	}
	return nil
}

// RestartConnector restarts a connector. In case maxConnectorRestarts is reached, it will attempt to hard-reset the connector. Returns a boolean to indicate
// whether the maxConnectorRestarts was reached and hard-reset was successfully performed. Returns an error if connector restart or hard-reset fails.
func (r *ReconcileGenericAutoConnector) RestartConnector(connector *skynetv1alpha1.GenericAutoConnector, kcc kafkaconnect.KafkaConnectClient) (bool, error) {
	var connectorRestartCount int
	value, ok := r.Cache.Load(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config["name"]))

	maxConnectorRestarts, cacheErr := operatorconfig.GetMaxConnectorRestarts(r.Cache)

	if cacheErr != nil {
		return false, cacheErr
	}

	if ok {
		connectorRestartCount = value.(int)

		if connectorRestartCount >= maxConnectorRestarts {
			err := r.HardResetConnector(connector, kcc)

			if err != nil {
				log.Error(err, fmt.Sprintf("Failed to hard-reset connector %s", connector.Spec.Config["name"]))
				return false, err
			}
			return true, nil
		}
	} else {
		connectorRestartCount = 0
	}

	log.Info(fmt.Sprintf("Restarting failed connector %s", connector.Spec.Config["name"]))
	response, error := kcc.RestartConnector(connector.Spec.Config["name"])

	if error != nil {
		return false, error
	}

	if response.Result == "success" {
		connectorRestartCount++
		r.Cache.Store(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config["name"]), connectorRestartCount)
		log.V(1).Info(fmt.Sprintf("Restart count for connector %s is now %d", connector.Spec.Config["name"], connectorRestartCount))
		return false, nil
	}
	return false, fmt.Errorf("Error occurred restarting connector %s", connector.Spec.Config["name"])
}

// RestartTask restarts a task given a taskID. If the maxTaskRestarts count has been reached, then it restarts the connector and returns a boolean indicating
// whether the threshold has been met. Returns an error if it fails to restart the task or the connector.
func (r *ReconcileGenericAutoConnector) RestartTask(connector *skynetv1alpha1.GenericAutoConnector, taskID int, kcc kafkaconnect.KafkaConnectClient) (bool, error) {
	var taskRestartCount int
	value, ok := r.Cache.Load(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config["name"], taskID))

	maxTaskRestarts, cacheErr := operatorconfig.GetMaxTaskRestarts(r.Cache)

	if cacheErr != nil {
		return false, cacheErr
	}

	if ok {
		taskRestartCount = value.(int)

		if taskRestartCount >= maxTaskRestarts {
			log.V(1).Info(fmt.Sprintf("Task %d for connector %s has reached failure threshold", taskID, connector.Spec.Config["name"]))
			thresholdReached, err := r.RestartConnector(connector, kcc)

			if err != nil {
				log.Error(err, fmt.Sprintf("Failed to restart connector %s", connector.Spec.Config["name"]))
				return false, err
			}

			if thresholdReached {
				r.Cache.Store(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config["name"]), 0)
			}
			return true, nil
		}
	} else {
		taskRestartCount = 0
	}

	log.Info(fmt.Sprintf("Failed task %d for connector %s is being restarted", taskID, connector.Spec.Config["name"]))
	response, error := kcc.RestartTask(connector.Spec.Config["name"], taskID)

	if error != nil {
		log.Error(error, fmt.Sprintf("Failed to restart task %d for connector %s", taskID, connector.Spec.Config["name"]))
		return false, error
	}

	if response.Result == "success" {
		taskRestartCount++
		r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config["name"], taskID), taskRestartCount)
		log.V(1).Info(fmt.Sprintf("Restart count for task %d of connector %s is now %d", taskID, connector.Spec.Config["name"], taskRestartCount))
	} else {
		return false, fmt.Errorf("Error occurred restarting task %d for connector %s", taskID, connector.Spec.Config["name"])
	}
	return false, nil
}

// UpdateUptimeMetric ...
func (r *ReconcileGenericAutoConnector) UpdateUptimeMetric(connector *skynetv1alpha1.GenericAutoConnector, running bool) {
	if !running {
		r.Cache.Store(fmt.Sprintf(connectorLastHealthyCheckCachePath, connector.Spec.Config["name"]), nil)
		r.Metrics.SetGauge(string(metrics.ConnectorUptime), 0, connector.Namespace, controllerName, connector.Spec.Config["name"])
	} else {
		stored, ok := r.Cache.Load(fmt.Sprintf(connectorLastHealthyCheckCachePath, connector.Spec.Config["name"]))

		now := time.Now()
		if !ok || stored == nil {
			r.Metrics.SetGauge(string(metrics.ConnectorUptime), 0, connector.Namespace, controllerName, connector.Spec.Config["name"])
		} else {
			diff := now.Sub(stored.(time.Time))
			seconds := float64(diff / time.Second)
			log.V(1).Info(fmt.Sprintf("Updating uptime by %fs", seconds))
			r.Metrics.AddToGauge(string(metrics.ConnectorUptime), seconds, connector.Namespace, controllerName, connector.Spec.Config["name"])
		}
		r.Cache.Store(fmt.Sprintf(connectorLastHealthyCheckCachePath, connector.Spec.Config["name"]), now)
	}
}

// CheckAndHealConnector ...
func (r *ReconcileGenericAutoConnector) CheckAndHealConnector(connector *skynetv1alpha1.GenericAutoConnector, kcc kafkaconnect.KafkaConnectClient) (bool, error) {
	var healthy bool

	response, readErr := kcc.GetStatus(connector.Spec.Config["name"])

	if readErr != nil {
		return false, readErr
	}

	status := response.Payload.(kafkaconnect.Status)

	runningTasksCount := status.GetActiveTasksCount()
	totalTaskCount := status.GetTaskCount()
	r.Cache.Store(fmt.Sprintf(totalTasksCountCachePath, connector.Spec.Config["name"]), totalTaskCount)
	r.Cache.Store(fmt.Sprintf(runningTasksCountCachePath, connector.Spec.Config["name"]), runningTasksCount)

	r.Metrics.SetGauge(string(metrics.TotalNumTasks), float64(totalTaskCount), connector.Namespace, controllerName, connector.Spec.Config["name"])
	r.Metrics.SetGauge(string(metrics.NumRunningTasks), float64(runningTasksCount), connector.Namespace, controllerName, connector.Spec.Config["name"])

	failed := status.IsConnectorFailed()
	if failed {
		connRestartThresholdReached, err := r.RestartConnector(connector, kcc)

		if connRestartThresholdReached {
			for t := 0; t < totalTaskCount; t++ {
				r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config["name"], t), 0)
			}
			r.Cache.Store(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config["name"]), 0)
		}
		return false, err
	}

	r.UpdateUptimeMetric(connector, !failed)

	healthy = true
	for i := 0; i < totalTaskCount; i++ {
		failed, err := status.IsTaskFailed(i)
		if err != nil {
			return healthy, err
		}
		if failed {
			healthy = false
			thresholdReached, err := r.RestartTask(connector, i, kcc)

			if err != nil {
				log.Error(err, fmt.Sprintf("Failed to restart task %d for connector %s", i, connector.Spec.Config["name"]))
				return healthy, err
			} else if thresholdReached {
				for t := 0; t < totalTaskCount; t++ {
					r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config["name"], t), 0)
				}
				break
			}
		} else {
			r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config["name"], i), 0)
		}
	}
	return healthy, nil
}

// ManageOperatorLogic ...
func (r *ReconcileGenericAutoConnector) ManageOperatorLogic(obj metav1.Object, kcc kafkaconnect.KafkaConnectClient) error {
	log.V(1).Info("Calling ManageOperatorLogic")
	connector, ok := obj.(*skynetv1alpha1.GenericAutoConnector)

	if !ok {
		return errors.New("Object is not of type GenericAutoConnector")
	}

	if connector.Spec.Config == nil {
		return errors.New("Could not get configuration from GenericAutoConnector")
	}

	conObj := kafkaconnect.Connector{
		Name:   connector.Spec.Config["name"],
		Config: connector.Spec.Config,
	}

	response, readErr := kcc.Read(connector.Spec.Config["name"])
	log.V(1).Info(fmt.Sprintf("Reading connector %s from Kafka Connect", connector.Spec.Config["name"]))

	if response.Result == "success" {
		if !reflect.DeepEqual(connector.Spec.Config, response.Payload) {
			resp, updateErr := kcc.Update(conObj)
			if resp.Result == "success" {
				log.Info(fmt.Sprintf("Connector %s already exists, updating configuration", connector.Spec.Config["name"]))
				return nil
			} else if resp.Result == "conflict" {
				log.Info(fmt.Sprintf("Conflict returned trying to update connector %s ", connector.Spec.Config["name"]))
				return nil
			}
			log.Error(updateErr, fmt.Sprintf("Error occurred updating configuration for connector %s", connector.Spec.Config["name"]))
			return updateErr
		}
		healthy, err := r.CheckAndHealConnector(connector, kcc)
		if err != nil {
			return err
		}
		if healthy {
			log.V(1).Info(fmt.Sprintf("Connector %s is healthy", connector.Spec.Config["name"]))
		} else {
			log.Info(fmt.Sprintf("Connector %s is unhealthy, self-healing in progress", connector.Spec.Config["name"]))
		}
	} else if response.Result == "conflict" {
		log.Info(fmt.Sprintf("Conflict returned trying to update connector %s ", connector.Spec.Config["name"]))
		return nil
	} else if response.Result == "notfound" {
		log.Info(fmt.Sprintf("Connector %s not found, read returned error %s", connector.Spec.Config["name"], readErr.Error()))
		resp3, createErr := kcc.Create(conObj)
		log.Info(fmt.Sprintf("Creating new connector %s", connector.Spec.Config["name"]))
		if resp3.Result == "success" {
			return nil
		}
		log.Error(createErr, fmt.Sprintf("Error occurred creating connector %s", connector.Spec.Config["name"]))
		return createErr
	} else if readErr != nil {
		log.Error(readErr, fmt.Sprintf("Error occurred reading connector %s", connector.Spec.Config["name"]))
		return readErr
	}
	return nil
}

// IsValid ...
func (r *ReconcileGenericAutoConnector) IsValid(obj metav1.Object) (bool, error) {
	connector, ok := obj.(*skynetv1alpha1.GenericAutoConnector)

	log.V(1).Info(fmt.Sprintf("Validating CR %s", obj.GetName()))

	v := validator.New()

	if !ok {
		return false, errors.New("Object is not of type GenericAutoConnector")
	}
	config := connector.Spec.Config

	if config == nil {
		return false, fmt.Errorf("Invalid connector, configuration is nil")
	}

	return v.ValidateMap(config)
}

// IsInitialized ...
func (r *ReconcileGenericAutoConnector) IsInitialized(obj metav1.Object) bool {
	initialized := true

	connector, ok := obj.(*skynetv1alpha1.GenericAutoConnector)

	log.Info(fmt.Sprintf("Checking if CR %s is initialized", obj.GetName()))

	if !ok {
		initialized = false
		log.Error(errors.New("Type assertion error from metav1.Object to GenericAutoConnector"), fmt.Sprintf("Could not retrieve GenericAutoConnector %s", obj.GetName()))
	} else {
		if !util.HasFinalizer(connector, controllerName) {
			log.Info("Adding finalizer to GenericAutoConnector")
			controllerutil.AddFinalizer(connector, controllerName)
			initialized = false
		}

		if connector.Spec.Config["name"] == "" {
			connector.Spec.Config["name"] = connector.ObjectMeta.Name + "_" + utils.RandSeq(8)
			return false
		}
	}
	return initialized
}

func (r *ReconcileGenericAutoConnector) getTasksInfoFromCache(name string) (int, int) {
	v0, ok0 := r.Cache.Load(fmt.Sprintf(totalTasksCountCachePath, name))
	v1, ok1 := r.Cache.Load(fmt.Sprintf(runningTasksCountCachePath, name))

	if !ok0 {
		v0 = 0
	}

	if !ok1 {
		v1 = 0
	}

	return v0.(int), v1.(int)
}

// ManageSuccess ...
func (r *ReconcileGenericAutoConnector) ManageSuccess(obj metav1.Object) (reconcile.Result, error) {
	log.V(1).Info("Executing ManageSuccess")

	if connector, updateStatus := obj.(*skynetv1alpha1.GenericAutoConnector); updateStatus {
		total, running := r.getTasksInfoFromCache(connector.Spec.Config["name"])

		connector.Status = v1alpha1.GenericAutoConnectorStatus{
			ConnectorName: connector.Spec.Config["name"],
			Topics:        connector.Spec.Config["topics"],
			Status:        "Running",
			LastUpdate:    metav1.Now(),
			Tasks:         fmt.Sprintf("%d/%d", running, total),
		}

		err := r.GetClient().Status().Update(context.Background(), connector)
		if err != nil {
			log.Error(err, "unable to update status")
			return reconcile.Result{
				RequeueAfter: time.Second,
				Requeue:      true,
			}, nil
		}
	} else {
		log.Error(errors.New("Type assertion error from metav1.Object to GenericAutoConnector"), "Object is not GenericAutoConnector, not setting status")
	}

	refreshFromKafkaConnectInterval, cacheErr := operatorconfig.GetReconcilePeriod(r.Cache)

	if cacheErr != nil {
		return reconcile.Result{}, cacheErr
	}

	// TODO: If maxConnectorHardReset has been reached, the Status should be set to failed
	// and the resource should not be requeued for reconciliation
	return reconcile.Result{
		RequeueAfter: time.Duration(refreshFromKafkaConnectInterval) * time.Second,
		Requeue:      true,
	}, nil
}

// ManageCleanUpLogic ...
func (r *ReconcileGenericAutoConnector) ManageCleanUpLogic(obj metav1.Object, kcc kafkaconnect.KafkaConnectClient) error {
	log.V(1).Info("Running clean up logic on GenericAutoConnector")
	connector, ok := obj.(*skynetv1alpha1.GenericAutoConnector)

	if !ok {
		return errors.New("Object is not of type GenericAutoConnector")
	}

	if connector.Spec.Config == nil {
		return errors.New("Could not get configuration from GenericAutoConnector")
	}

	if connector.Spec.Config["name"] == "" {
		return errors.New("Connector name is not set")
	}

	response, err := kcc.Delete(connector.Spec.Config["name"])

	if response.Result == "success" {
		log.Info(fmt.Sprintf("Connector %s deleted successfully", connector.Spec.Config["name"]))
		return nil
	}
	log.Error(err, fmt.Sprintf("Failed to delete connector %s", connector.Spec.Config["name"]))
	return err
}

// ManageError ...
func (r *ReconcileGenericAutoConnector) ManageError(obj metav1.Object, issue error) (reconcile.Result, error) {
	log.V(1).Info("Calling ManageError")

	//TODO: What is an error condition in the case of an GenericAutoConnector?
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

	if connector, updateStatus := obj.(*skynetv1alpha1.GenericAutoConnector); updateStatus {
		total, running := r.getTasksInfoFromCache(connector.Spec.Config["name"])

		lastUpdate := connector.Status.LastUpdate.Time
		lastStatus := connector.Status.Status

		connector.Status = v1alpha1.GenericAutoConnectorStatus{
			ConnectorName: connector.Spec.Config["name"],
			Topics:        connector.Spec.Config["topics"],
			Tasks:         fmt.Sprintf("%d/%d", running, total),
			LastUpdate:    metav1.Now(),
			Reason:        issue.Error(),
			Status:        "Failure",
		}

		err := r.GetClient().Status().Update(context.Background(), connector)

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
			retryInterval = connector.Status.LastUpdate.Sub(lastUpdate).Round(time.Second)
		}
	} else {
		log.Error(errors.New("Type assertion error from metav1.Object to GenericAutoConnector"), "Object is not ReconcileStatusAware, not setting status")
		retryInterval = time.Second
	}

	return reconcile.Result{
		RequeueAfter: time.Duration(math.Min(float64(retryInterval.Nanoseconds()*2), float64(time.Hour.Nanoseconds()*6))),
		Requeue:      true,
	}, nil
}
