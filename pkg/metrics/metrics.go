package metrics

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
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
)

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

// NewPrometheusMetrics ...
func NewPrometheusMetrics() *PrometheusMetrics {
	obj := PrometheusMetrics{
		metrics: make(map[string]interface{}),
		log:     logf.Log.WithName(componentName),
		mutex:   new(sync.Mutex),
	}
	obj.InitMetrics()
	return &obj
}

// PrometheusMetrics ...
type PrometheusMetrics struct {
	mutex   *sync.Mutex
	metrics map[string]interface{}
	log     logr.Logger
}

var componentName string = "kafka_autoconnector_custom_metrics"

// InitMetrics ...
func (p *PrometheusMetrics) InitMetrics() {
	p.mutex.Lock()
	p.metrics[string(TotalNumTasks)] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafkautoconnector_total_connector_tasks",
			Help: "Total number of connector tasks",
		},
		[]string{"ns", "rn", "connector_name"},
	)
	p.metrics[string(TotalNumTasks)].(*prometheus.GaugeVec).WithLabelValues("", "", "")
	prometheus.MustRegister(p.metrics[string(TotalNumTasks)].(*prometheus.GaugeVec))

	p.metrics[string(NumRunningTasks)] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafkautoconnector_running_connector_tasks",
			Help: "Number of running connector tasks",
		},
		[]string{"ns", "rn", "connector_name"},
	)
	p.metrics[string(NumRunningTasks)].(*prometheus.GaugeVec).WithLabelValues("", "", "")
	prometheus.MustRegister(p.metrics[string(NumRunningTasks)].(*prometheus.GaugeVec))

	p.metrics[string(ConnectorUptime)] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafkautoconnector_connector_uptime",
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

// DestroyMetrics ...
func (p PrometheusMetrics) DestroyMetrics() {
	p.mutex.Lock()
	for _, v := range p.metrics {
		prometheus.Unregister(v.(prometheus.Collector))
	}
	p.mutex.Unlock()
}
