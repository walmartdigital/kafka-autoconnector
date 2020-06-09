package metrics

import (
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var metrics map[string]interface{}

// Metrics ...
type Metrics interface {
	InitMetrics()
	IncrementCounter(string, ...string)
	ResetCounter(string, ...string)
	SetGauge(string, float64, ...string)
	DestroyMetrics()
}

// MetricsFactory ...
type MetricsFactory interface {
	Create() Metrics
}

// PrometheusMetricsFactory ...
type PrometheusMetricsFactory struct {
}

// Create ...
func (p PrometheusMetricsFactory) Create() Metrics {
	metrics := new(PrometheusMetrics)
	metrics.InitMetrics()
	return metrics
}

// PrometheusMetrics ...
type PrometheusMetrics struct {
	metrics *sync.Map
	log     logr.Logger
}

var componentName string = "kafka_autoconnector_custom_metrics"

// InitMetrics ...
func (p PrometheusMetrics) InitMetrics() {
	p.metrics = new(sync.Map)
	p.log = logf.Log.WithName(componentName)

	totalNumTasksMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafkautoconnector_total_connector_tasks",
			Help: "Total number of connector tasks",
		},
		[]string{"ns", "rn"},
	)
	prometheus.MustRegister(totalNumTasksMetric)
	p.metrics.Store(
		"totalNumTasks",
		totalNumTasksMetric,
	)

	runningNumTasksMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafkautoconnector_running_connector_tasks",
			Help: "Number of running connector tasks",
		},
		[]string{"ns", "rn"},
	)
	prometheus.MustRegister(runningNumTasksMetric)
	p.metrics.Store(
		"numRunningTasks",
		runningNumTasksMetric,
	)

	uptimeMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafkautoconnector_connector_uptime",
			Help: "Time that connector has been in RUNNING state",
		},
		[]string{"ns", "rn"},
	)
	prometheus.MustRegister(uptimeMetric)
	p.metrics.Store(
		"connectorUptime",
		uptimeMetric,
	)
}

// IncrementCounter ...
func (p PrometheusMetrics) IncrementCounter(key string, labels ...string) {
	m, ok := p.metrics.Load(key)

	if !ok {
		err := fmt.Errorf("Panicking, could not retrieve metric %s", key)
		p.log.Error(err, "Error occurred while incrementing metric counter")
		panic(err.Error())
	}
	m.(*prometheus.CounterVec).WithLabelValues(labels...).Inc()
}

// ResetCounter ...
func (p PrometheusMetrics) ResetCounter(key string, labels ...string) {
	m, ok := p.metrics.Load(key)

	if !ok {
		err := fmt.Errorf("Panicking, could not retrieve metric %s", key)
		p.log.Error(err, "Error occurred while incrementing metric counter")
		panic(err.Error())
	}
	m.(*prometheus.CounterVec).Reset()
}

// SetGauge ...
func (p PrometheusMetrics) SetGauge(key string, value float64, labels ...string) {
	m, ok := p.metrics.Load(key)

	if !ok {
		err := fmt.Errorf("Panicking, could not retrieve metric %s", key)
		p.log.Error(err, "Error occurred while incrementing metric counter")
		panic(err.Error())
	}
	m.(*prometheus.GaugeVec).WithLabelValues(labels...).Set(value)
}

// DestroyMetrics ...
func (p PrometheusMetrics) DestroyMetrics() {
	p.metrics.Range(func(key interface{}, value interface{}) bool {
		prometheus.Unregister(value.(prometheus.Collector))
		p.metrics.Delete(key)
		return true
	})
}
