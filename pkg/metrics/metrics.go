package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	monclientv1 "github.com/coreos/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	"github.com/gorilla/mux"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/prometheus/client_golang/prometheus"
	controller_http "github.com/walmartdigital/kafka-autoconnector/pkg/http"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// MetricsKey ...
type MetricsKey string

var (
	// TotalNumTasks ...
	TotalNumTasks MetricsKey = "totalNumTasks"
	// NumRunningTasks ...
	NumRunningTasks MetricsKey = "numRunningTasks"
	// ConnectorUptime ...
	ConnectorUptime MetricsKey = "connectorUptime"

	componentName = "kafka_autoconnector_custom_metrics"
	log           = logf.Log.WithName(componentName)
)

// Metrics ...
type Metrics interface {
	InitMetrics()
	IncrementCounter(string, ...string)
	ResetCounter(string, ...string)
	SetGauge(string, float64, ...string)
	AddToGauge(string, float64, ...string)
	DestroyMetrics()
}

// MetricsFactory ...
type MetricsFactory interface {
	Create() Metrics
}

// PrometheusMetricsFactory ...
type PrometheusMetricsFactory struct {
}

// NewPrometheusMetrics ...
func NewPrometheusMetrics() *PrometheusMetrics {
	obj := PrometheusMetrics{
		metrics: make(map[string]interface{}),
		mutex:   new(sync.Mutex),
	}
	obj.InitMetrics()
	return &obj
}

// PrometheusMetrics ...
type PrometheusMetrics struct {
	mutex   *sync.Mutex
	metrics map[string]interface{}
	// log     logr.Logger
}

// InitMetrics ...
func (p *PrometheusMetrics) InitMetrics() {
	p.mutex.Lock()
	p.metrics[string(TotalNumTasks)] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_autoconnector_total_connector_tasks",
			Help: "Total number of connector tasks",
		},
		[]string{"ns", "rn", "connector_name"},
	)
	p.metrics[string(TotalNumTasks)].(*prometheus.GaugeVec).WithLabelValues("", "", "")
	prometheus.MustRegister(p.metrics[string(TotalNumTasks)].(*prometheus.GaugeVec))

	p.metrics[string(NumRunningTasks)] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_autoconnector_running_connector_tasks",
			Help: "Number of running connector tasks",
		},
		[]string{"ns", "rn", "connector_name"},
	)
	p.metrics[string(NumRunningTasks)].(*prometheus.GaugeVec).WithLabelValues("", "", "")
	prometheus.MustRegister(p.metrics[string(NumRunningTasks)].(*prometheus.GaugeVec))

	p.metrics[string(ConnectorUptime)] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_autoconnector_connector_uptime",
			Help: "Time that connector has been in RUNNING state",
		},
		[]string{"ns", "rn", "connector_name"},
	)
	p.metrics[string(ConnectorUptime)].(*prometheus.GaugeVec).WithLabelValues("", "", "")
	prometheus.MustRegister(p.metrics[string(ConnectorUptime)].(*prometheus.GaugeVec))
	p.mutex.Unlock()
}

// IncrementCounter ...
func (p *PrometheusMetrics) IncrementCounter(key string, labels ...string) {
	p.mutex.Lock()
	p.metrics[key].(*prometheus.CounterVec).WithLabelValues(labels...).Inc()
	p.mutex.Unlock()
}

// ResetCounter ...
func (p *PrometheusMetrics) ResetCounter(key string, labels ...string) {
	p.mutex.Lock()
	p.metrics[key].(*prometheus.CounterVec).Reset()
	p.mutex.Unlock()
}

// SetGauge ...
func (p *PrometheusMetrics) SetGauge(key string, value float64, labels ...string) {
	p.mutex.Lock()
	p.metrics[key].(*prometheus.GaugeVec).WithLabelValues(labels...).Set(value)
	p.mutex.Unlock()
}

// AddToGauge ...
func (p *PrometheusMetrics) AddToGauge(key string, value float64, labels ...string) {
	p.mutex.Lock()
	p.metrics[key].(*prometheus.GaugeVec).WithLabelValues(labels...).Add(value)
	p.mutex.Unlock()
}

// DestroyMetrics ...
func (p PrometheusMetrics) DestroyMetrics() {
	p.mutex.Lock()
	for _, v := range p.metrics {
		prometheus.Unregister(v.(prometheus.Collector))
	}
	p.mutex.Unlock()
}

// AddCustomMetrics ...
func (p PrometheusMetrics) AddCustomMetrics(ctx context.Context, cfg *rest.Config, wg *sync.WaitGroup, port int, portName string, labels map[string]string) {
	addCustomMetrics(ctx, cfg, wg, port, portName, labels)
}

func serveCustomMetrics(wg *sync.WaitGroup, port int) error {
	router := mux.NewRouter().StrictSlash(true)
	routerWrapper := &controller_http.RouterWrapper{Router: router}
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}
	metricsServer := controller_http.CreateServer(httpServer, routerWrapper)
	log.Info("Starting the custom metrics server.")
	go metricsServer.Run(wg)
	return nil
}

// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addCustomMetrics(ctx context.Context, cfg *rest.Config, wg *sync.WaitGroup, port int, portName string, labels map[string]string) {
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		if errors.Is(err, k8sutil.ErrRunLocal) {
			log.Info("Skipping CR metrics server creation; not running in a cluster.")
			return
		}
	}

	if err := serveCustomMetrics(wg, port); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: int32(port), Name: portName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(port)}},
	}

	// Create Service object to expose the metrics port(s).
	service, err := createCustomMetricsService(ctx, cfg, servicePorts, labels)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}

	// The ServiceMonitor is created in the same namespace where the operator is deployed
	_, err = createServiceMonitors(cfg, operatorNs, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == errServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}

func createCustomMetricsService(ctx context.Context, cfg *rest.Config, servicePorts []v1.ServicePort, labels map[string]string) (*v1.Service, error) {
	if len(servicePorts) < 1 {
		return nil, fmt.Errorf("failed to create metrics Serice; service ports were empty")
	}
	client, err := crclient.New(cfg, crclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create new client: %w", err)
	}
	s, err := initOperatorService(ctx, client, servicePorts, labels)
	if err != nil {
		if err == k8sutil.ErrNoNamespace || err == k8sutil.ErrRunLocal {
			log.Info("Skipping metrics Service creation; not running in a cluster.")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to initialize service object for metrics: %w", err)
	}
	service, err := createOrUpdateService(ctx, client, s)
	if err != nil {
		return nil, fmt.Errorf("failed to create or get service for metrics: %w", err)
	}
	return service, nil
}

// initOperatorService returns the static service which exposes specified port(s).
func initOperatorService(ctx context.Context, client crclient.Client, sp []v1.ServicePort, labels map[string]string) (*v1.Service, error) {
	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		return nil, err
	}
	namespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return nil, err
	}

	selectorLabel := map[string]string{"name": operatorName}

	selfLabels := make(map[string]string)
	selfLabels["name"] = operatorName

	if labels != nil {
		for k, v := range labels {
			selfLabels[k] = v
		}
	}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-custom-metrics", operatorName),
			Namespace: namespace,
			Labels:    selfLabels,
		},
		Spec: v1.ServiceSpec{
			Ports:    sp,
			Selector: selectorLabel,
		},
	}

	ownRef, err := getPodOwnerRef(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	service.SetOwnerReferences([]metav1.OwnerReference{*ownRef})
	return service, nil
}

func getPodOwnerRef(ctx context.Context, client crclient.Client, ns string) (*metav1.OwnerReference, error) {
	// Get current Pod the operator is running in
	pod, err := k8sutil.GetPod(ctx, client, ns)
	if err != nil {
		return nil, err
	}
	podOwnerRefs := metav1.NewControllerRef(pod, pod.GroupVersionKind())
	// Get Owner that the Pod belongs to
	ownerRef := metav1.GetControllerOf(pod)
	finalOwnerRef, err := findFinalOwnerRef(ctx, client, ns, ownerRef)
	if err != nil {
		return nil, err
	}
	if finalOwnerRef != nil {
		return finalOwnerRef, nil
	}

	// Default to returning Pod as the Owner
	return podOwnerRefs, nil
}

// findFinalOwnerRef tries to locate the final controller/owner based on the owner reference provided.
func findFinalOwnerRef(ctx context.Context, client crclient.Client, ns string,
	ownerRef *metav1.OwnerReference) (*metav1.OwnerReference, error) {
	if ownerRef == nil {
		return nil, nil
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(ownerRef.APIVersion)
	obj.SetKind(ownerRef.Kind)
	err := client.Get(ctx, types.NamespacedName{Namespace: ns, Name: ownerRef.Name}, obj)
	if err != nil {
		return nil, err
	}
	newOwnerRef := metav1.GetControllerOf(obj)
	if newOwnerRef != nil {
		return findFinalOwnerRef(ctx, client, ns, newOwnerRef)
	}

	log.V(1).Info("Pods owner found", "Kind", ownerRef.Kind, "Name",
		ownerRef.Name, "Namespace", ns)
	return ownerRef, nil
}

func createOrUpdateService(ctx context.Context, client crclient.Client, s *v1.Service) (*v1.Service, error) {
	if err := client.Create(ctx, s); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
		// Service already exists, we want to update it
		// as we do not know if any fields might have changed.
		existingService := &v1.Service{}
		err := client.Get(ctx, types.NamespacedName{
			Name:      s.Name,
			Namespace: s.Namespace,
		}, existingService)
		if err != nil {
			return nil, err
		}

		s.ResourceVersion = existingService.ResourceVersion
		if existingService.Spec.Type == v1.ServiceTypeClusterIP {
			s.Spec.ClusterIP = existingService.Spec.ClusterIP
		}
		err = client.Update(ctx, s)
		if err != nil {
			return nil, err
		}
		log.Info("Metrics Service object updated", "Service.Name",
			s.Name, "Service.Namespace", s.Namespace)
		return s, nil
	}

	log.Info("Metrics Service object created", "Service.Name",
		s.Name, "Service.Namespace", s.Namespace)
	return s, nil
}

var errServiceMonitorNotPresent = fmt.Errorf("no ServiceMonitor registered with the API")

// ServiceMonitorUpdater ...
type ServiceMonitorUpdater func(*monitoringv1.ServiceMonitor) error

func createServiceMonitors(config *rest.Config, ns string, services []*v1.Service,
	updaters ...ServiceMonitorUpdater) ([]*monitoringv1.ServiceMonitor, error) {
	// check if we can even create ServiceMonitors
	exists, err := hasServiceMonitor(config)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errServiceMonitorNotPresent
	}

	var serviceMonitors []*monitoringv1.ServiceMonitor
	mclient := monclientv1.NewForConfigOrDie(config)

	for _, s := range services {
		if s == nil {
			continue
		}
		sm := generateServiceMonitor(s)
		for _, update := range updaters {
			if err := update(sm); err != nil {
				return nil, err
			}
		}

		smc, err := mclient.ServiceMonitors(ns).Create(sm)
		if err != nil {
			return serviceMonitors, err
		}
		serviceMonitors = append(serviceMonitors, smc)
	}

	return serviceMonitors, nil
}

func generateServiceMonitor(s *v1.Service) *monitoringv1.ServiceMonitor {
	labels := make(map[string]string)
	for k, v := range s.ObjectMeta.Labels {
		labels[k] = v
	}

	endpoints := populateEndpointsFromServicePorts(s)
	boolTrue := true

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.ObjectMeta.Name,
			Namespace: s.ObjectMeta.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					BlockOwnerDeletion: &boolTrue,
					Controller:         &boolTrue,
					Kind:               "Service",
					Name:               s.Name,
					UID:                s.UID,
				},
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Endpoints: endpoints,
		},
	}
}

func populateEndpointsFromServicePorts(s *v1.Service) []monitoringv1.Endpoint {
	var endpoints []monitoringv1.Endpoint
	for _, port := range s.Spec.Ports {
		endpoints = append(endpoints, monitoringv1.Endpoint{Port: port.Name})
	}
	return endpoints
}

// hasServiceMonitor checks if ServiceMonitor is registered in the cluster.
func hasServiceMonitor(config *rest.Config) (bool, error) {
	dc := discovery.NewDiscoveryClientForConfigOrDie(config)
	apiVersion := "monitoring.coreos.com/v1"
	kind := "ServiceMonitor"

	return k8sutil.ResourceExists(dc, apiVersion, kind)
}
