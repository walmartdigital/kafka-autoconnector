package essinkconnector

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/walmartdigital/go-kaya/pkg/client"
	"github.com/walmartdigital/go-kaya/pkg/kafkaconnect"
	"github.com/walmartdigital/go-kaya/pkg/validator"
	"github.com/walmartdigital/kafka-autoconnector/pkg/apis/skynet/v1alpha1"
	skynetv1alpha1 "github.com/walmartdigital/kafka-autoconnector/pkg/apis/skynet/v1alpha1"
	"github.com/walmartdigital/kafka-autoconnector/pkg/cache"
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
	controllerName                  = "controller_essinkconnector"
	log                             = logf.Log.WithName(controllerName)
	kafkaConnectHost                = "192.168.64.5:30256"
	connectorRestartCachePath       = "/essinkconnector/connectors/%s/restart"
	taskRestartCachePath            = "/essinkconnector/connectors/%s/tasks/%d/restart"
	connectorHardResetCachePath     = "/essinkconnector/connectors/%s/hardreset"
	totalTasksCountCachePath        = "/essinkconnector/connectors/%s/tasks/total/count"
	runningTasksCountCachePath      = "/essinkconnector/connectors/%s/tasks/running/count"
	refreshFromKafkaConnectInterval = 5
	maxConnectorRestarts            = 5
	maxTaskRestarts                 = 5
	maxConnectorHardResets          = 3
)

func init() {
	addr := os.Getenv("KAFKA_CONNECT_ADDR")
	if addr != "" {
		kafkaConnectHost = addr
	}

	interval := os.Getenv("REFRESH_INTERVAL")
	if interval != "" {
		val, err := strconv.Atoi(interval)
		if err == nil {
			refreshFromKafkaConnectInterval = val
		}
	}

	connectorThreshold := os.Getenv("MAX_CONNECTOR_RESTARTS")
	if connectorThreshold != "" {
		val, err := strconv.Atoi(connectorThreshold)
		if err == nil {
			maxConnectorRestarts = val
		}
	}

	taskThreshold := os.Getenv("MAX_TASK_RESTARTS")
	if taskThreshold != "" {
		val, err := strconv.Atoi(taskThreshold)
		if err == nil {
			maxTaskRestarts = val
		}
	}

	hardReset := os.Getenv("MAX_CONNECT_HARD_RESETS")
	if hardReset != "" {
		val, err := strconv.Atoi(hardReset)
		if err == nil {
			maxConnectorHardResets = val
		}
	}
}

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ESSinkConnector Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, c cache.Cache, m metrics.Metrics, kcf kafkaconnect.KafkaConnectClientFactory) error {
	return add(mgr, newReconciler(mgr, c, m, kcf))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c cache.Cache, m metrics.Metrics, kcf kafkaconnect.KafkaConnectClientFactory) reconcile.Reconciler {
	return &ReconcileESSinkConnector{
		ReconcilerBase:            util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(controllerName)),
		KafkaConnectClientFactory: kcf,
		Cache:                     c,
		Metrics:                   m,
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
	//err = c.Watch(&source.Kind{Type: &skynetv1alpha1.ESSinkConnector{}}, &handler.EnqueueRequestForObject{})
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
	metrics.Metrics
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

// HardResetConnector ...
func (r *ReconcileESSinkConnector) HardResetConnector(connector *skynetv1alpha1.ESSinkConnector, kcc kafkaconnect.KafkaConnectClient) error {
	var connectorHardResetCount int
	value, ok := r.Cache.Load(fmt.Sprintf(connectorHardResetCachePath, connector.Spec.Config.Name))

	if ok {
		connectorHardResetCount = value.(int)
	} else {
		connectorHardResetCount = 0
	}

	if connectorHardResetCount < maxConnectorHardResets {
		resp0, err0 := kcc.Delete(connector.Spec.Config.Name)

		if err0 != nil {
			log.Error(err0, fmt.Sprintf("Failed to delete connector %s", connector.Spec.Config.Name))
			return err0
		}

		if resp0.Result == "success" {
			log.Error(err0, fmt.Sprintf("Successfully deleted connector %s", connector.Spec.Config.Name))
			conObj := kafkaconnect.Connector{
				Name:   connector.Spec.Config.Name,
				Config: &connector.Spec.Config,
			}

			resp1, err1 := kcc.Create(conObj)

			if err0 != nil {
				log.Error(err1, fmt.Sprintf("Failed to recreate connector %s", connector.Spec.Config.Name))
				return err1
			}

			if resp1.Result == "success" {
				connectorHardResetCount++
				r.Cache.Store(fmt.Sprintf(connectorHardResetCachePath, connector.Spec.Config.Name), connectorHardResetCount)
				log.Info(fmt.Sprintf("Hard reset count for connector %s is now %d", connector.Spec.Config.Name, connectorHardResetCount))
				return nil
			}
			return fmt.Errorf("Failed to recreate connector %s", connector.Spec.Config.Name)
		}
	}
	return nil
}

// RestartConnector restarts a connector. In case maxConnectorRestarts is reached, it will attempt to hard-reset the connector. Returns a boolean to indicate
// whether the maxConnectorRestarts was reached and hard-reset was successfully performed. Returns an error if connector restart or hard-reset fails.
func (r *ReconcileESSinkConnector) RestartConnector(connector *skynetv1alpha1.ESSinkConnector, kcc kafkaconnect.KafkaConnectClient) (bool, error) {
	var connectorRestartCount int
	value, ok := r.Cache.Load(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config.Name))

	if ok {
		connectorRestartCount = value.(int)

		if connectorRestartCount >= maxConnectorRestarts {
			err := r.HardResetConnector(connector, kcc)

			if err != nil {
				log.Error(err, fmt.Sprintf("Failed to hard-reset connector %s", connector.Spec.Config.Name))
				return false, err
			}
			return true, nil
		}
	} else {
		connectorRestartCount = 0
	}

	log.Info(fmt.Sprintf("Restarting failed connector %s", connector.Spec.Config.Name))
	response, error := kcc.RestartConnector(connector.Spec.Config.Name)

	if error != nil {
		return false, error
	}

	if response.Result == "success" {
		connectorRestartCount++
		r.Cache.Store(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config.Name), connectorRestartCount)
		log.Info(fmt.Sprintf("Restart count for connector %s is now %d", connector.Spec.Config.Name, connectorRestartCount))
		return false, nil
	}
	return false, fmt.Errorf("Error occurred restarting connector %s", connector.Spec.Config.Name)
}

// RestartTask restarts a task given a taskID. If the maxTaskRestarts count has been reached, then it restarts the connector and returns a boolean indicating
// whether the threshold has been met. Returns an error if it fails to restart the task or the connector.
func (r *ReconcileESSinkConnector) RestartTask(connector *skynetv1alpha1.ESSinkConnector, taskID int, kcc kafkaconnect.KafkaConnectClient) (bool, error) {
	var taskRestartCount int
	value, ok := r.Cache.Load(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config.Name, taskID))

	if ok {
		taskRestartCount = value.(int)

		if taskRestartCount >= maxTaskRestarts {
			log.Info(fmt.Sprintf("Task %d for connector %s has reached failure threshold", taskID, connector.Spec.Config.Name))
			thresholdReached, err := r.RestartConnector(connector, kcc)

			if err != nil {
				log.Error(err, fmt.Sprintf("Failed to restart connector %s", connector.Spec.Config.Name))
				return false, err
			}

			if thresholdReached {
				r.Cache.Store(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config.Name), 0)
			}
			return true, nil
		}
	} else {
		taskRestartCount = 0
	}

	log.Info(fmt.Sprintf("Failed task %d for connector %s is being restarted", taskID, connector.Spec.Config.Name))
	response, error := kcc.RestartTask(connector.Spec.Config.Name, taskID)

	if error != nil {
		log.Error(error, fmt.Sprintf("Failed to restart task %d for connector %s", taskID, connector.Spec.Config.Name))
		return false, error
	}

	if response.Result == "success" {
		taskRestartCount++
		r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config.Name, taskID), taskRestartCount)
		log.Info(fmt.Sprintf("Restart count for task %d of connector %s is now %d", taskID, connector.Spec.Config.Name, taskRestartCount))
	} else {
		return false, fmt.Errorf("Error occurred restarting task %d for connector %s", taskID, connector.Spec.Config.Name)
	}
	return false, nil
}

// CheckAndHealConnector ...
func (r *ReconcileESSinkConnector) CheckAndHealConnector(connector *skynetv1alpha1.ESSinkConnector, kcc kafkaconnect.KafkaConnectClient) (bool, error) {
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

	r.Metrics.SetGauge(string(metrics.TotalNumTasks), float64(totalTaskCount), connector.Namespace, controllerName, connector.Spec.Config.Name)
	r.Metrics.SetGauge(string(metrics.NumRunningTasks), float64(runningTasksCount), connector.Namespace, controllerName, connector.Spec.Config.Name)

	if status.IsConnectorFailed() {
		connRestartThresholdReached, err := r.RestartConnector(connector, kcc)

		if connRestartThresholdReached {
			for t := 0; t < totalTaskCount; t++ {
				r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config.Name, t), 0)
			}
			r.Cache.Store(fmt.Sprintf(connectorRestartCachePath, connector.Spec.Config.Name), 0)
		}

		return false, err
	}
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
				log.Error(err, fmt.Sprintf("Failed to restart task %d for connector %s", i, connector.Spec.Config.Name))
				return healthy, err
			} else if thresholdReached {
				for t := 0; t < totalTaskCount; t++ {
					r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config.Name, t), 0)
				}
				break
			}
		} else {
			r.Cache.Store(fmt.Sprintf(taskRestartCachePath, connector.Spec.Config.Name, i), 0)
		}
	}
	return healthy, nil
}

// ManageOperatorLogic ...
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
	log.Info(fmt.Sprintf("Reading connector %s from Kafka Connect", config.Name))

	if response.Result == "success" {
		if !cmp.Equal(connector.Spec.Config, response.Payload, kafkaconnect.ConnectorConfigComparer) {
			resp, updateErr := kcc.Update(conObj)
			if resp.Result == "success" {
				log.Info(fmt.Sprintf("Connector %s already exists, updating configuration", config.Name))
				return nil
			}
			log.Error(updateErr, fmt.Sprintf("Error occurred updating configuration for connector %s", config.Name))
			return updateErr
		}
		healthy, err := r.CheckAndHealConnector(connector, kcc)
		if err != nil {
			return err
		}
		if healthy {
			log.Info(fmt.Sprintf("Connector %s is healthy", config.Name))
		} else {
			log.Info(fmt.Sprintf("Connector %s is unhealthy, self-healing in progress", config.Name))
		}
	} else if response.Result == "notfound" {
		log.Info(fmt.Sprintf("Connector %s not found", config.Name))
		resp3, createErr := kcc.Create(conObj)
		log.Info(fmt.Sprintf("Creating new connector %s", config.Name))
		if resp3.Result == "success" {
			return nil
		}
		log.Error(createErr, fmt.Sprintf("Error occurred creating connector %s", config.Name))
		return createErr
	} else if readErr != nil {
		log.Error(readErr, fmt.Sprintf("Error occurred reading connector %s", config.Name))
		return readErr
	}
	return nil
}

// IsValid ...
func (r *ReconcileESSinkConnector) IsValid(obj metav1.Object) (bool, error) {
	log.Info("Validating CR")

	connector, ok := obj.(*skynetv1alpha1.ESSinkConnector)
	v := validator.New()

	if !ok {
		return false, errors.New("Object is not of type ESSinkConnector")
	}
	config := connector.Spec.Config

	err := v.Struct(config)
	if err != nil {
		return false, err
	}
	return true, nil
}

// IsInitialized ...
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

func (r *ReconcileESSinkConnector) getTasksInfoFromCache(name string) (int, int) {
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
func (r *ReconcileESSinkConnector) ManageSuccess(obj metav1.Object) (reconcile.Result, error) {
	log.Info("Executing ManageSuccess")

	if connector, updateStatus := obj.(*skynetv1alpha1.ESSinkConnector); updateStatus {
		total, running := r.getTasksInfoFromCache(connector.Spec.Config.Name)

		connector.Status = v1alpha1.ESSinkConnectorStatus{
			ConnectorName: connector.Spec.Config.Name,
			Topics:        connector.Spec.Config.Topics,
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
		log.Info("object is not ReconcileStatusAware, not setting status")
	}

	// TODO: If maxConnectorHardReset has been reached, the Status should be set to failed
	// and the resource should not be requeued for reconciliation
	return reconcile.Result{
		RequeueAfter: time.Duration(refreshFromKafkaConnectInterval) * time.Second,
		Requeue:      true,
	}, nil
}

// ManageCleanUpLogic ...
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
	}
	log.Info(fmt.Sprintf("Failed to delete connector %s", config.Name))
	return err
}

// ManageError ...
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
		total, running := r.getTasksInfoFromCache(connector.Spec.Config.Name)

		lastUpdate := connector.Status.LastUpdate.Time
		lastStatus := connector.Status.Status

		connector.Status = v1alpha1.ESSinkConnectorStatus{
			ConnectorName: connector.Spec.Config.Name,
			Topics:        connector.Spec.Config.Topics,
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
		log.Info("object is not ReconcileStatusAware, not setting status")
		retryInterval = time.Second
	}

	return reconcile.Result{
		RequeueAfter: time.Duration(math.Min(float64(retryInterval.Nanoseconds()*2), float64(time.Hour.Nanoseconds()*6))),
		Requeue:      true,
	}, nil
}
