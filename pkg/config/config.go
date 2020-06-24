package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/kelseyhightower/envconfig"
	configcache "github.com/walmartdigital/kafka-autoconnector/pkg/cache"
)

// LabelsMap ...
type LabelsMap struct {
	Labels map[string]string
}

var (
	kafkaConnectHost                = "192.168.64.5:30256"
	refreshFromKafkaConnectInterval = 5
	maxConnectorRestarts            = 5
	maxTaskRestarts                 = 5
	maxConnectorHardResets          = 3
	kafkaConnectAddrCacheKey        = "/config/global/kafkaconnect/address"
	reconcilePeriodCacheKey         = "/config/global/reconcile/period"
	maxConnectorRestartsCacheKey    = "/config/global/connectors/maxrestarts"
	maxConnectorHardResetsCacheKey  = "/config/global/connectors/maxhardresets"
	maxTaskRestartsCacheKey         = "/config/global/tasks/maxrestarts"
	customMetricsPort               = 10000
	customMetricsPortCacheKey       = "/config/global/metrics/port/number"
	customMetricsPortName           = "custom-metrics"
	customMetricsPortNameCacheKey   = "/config/global/metrics/port/name"
	customMetricsSMLabels           LabelsMap
	customMetricsSMLabelsCacheKey   = "/config/global/metrics/servicemonitor/labels"
)

// LoadFromEnvironment loads configuration parameters from environment variables and
// store them in the provided cache
func LoadFromEnvironment(configCache configcache.Cache) {
	if configCache == nil {
		panic(errors.New("Could not load config because cache is nil"))
	}

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

	port := os.Getenv("CUSTOM_METRICS_PORT")
	if port != "" {
		val, err := strconv.Atoi(port)
		if err == nil {
			customMetricsPort = val
		}
	}

	portName := os.Getenv("CUSTOM_METRICS_PORT_NAME")
	if portName != "" {
		customMetricsPortName = portName
	}

	err := envconfig.Process("service_monitor", &customMetricsSMLabels)

	if err != nil {
		panic(err)
	}

	if customMetricsSMLabels.Labels != nil {
		configCache.Store(customMetricsSMLabelsCacheKey, customMetricsSMLabels.Labels)
	}

	configCache.Store(kafkaConnectAddrCacheKey, kafkaConnectHost)
	configCache.Store(reconcilePeriodCacheKey, refreshFromKafkaConnectInterval)
	configCache.Store(maxConnectorRestartsCacheKey, maxConnectorRestarts)
	configCache.Store(maxConnectorHardResetsCacheKey, maxConnectorHardResets)
	configCache.Store(maxTaskRestartsCacheKey, maxTaskRestarts)
	configCache.Store(customMetricsPortCacheKey, customMetricsPort)
	configCache.Store(customMetricsPortNameCacheKey, customMetricsPortName)
}

// GetServiceMonitorLabels returns the KafkaConnect address stored in the provided cache
func GetServiceMonitorLabels(configCache configcache.Cache) (map[string]string, error) {
	labels, ok := configCache.Load(customMetricsSMLabelsCacheKey)
	if !ok {
		return nil, errors.New("Could not retrieve Service Monitor labels from cache")
	}
	return labels.(map[string]string), nil
}

// GetKafkaConnectAddress returns the KafkaConnect address stored in the provided cache
func GetKafkaConnectAddress(configCache configcache.Cache) (string, error) {
	addr, ok := configCache.Load(kafkaConnectAddrCacheKey)
	if !ok {
		return "", errors.New("Could not retrieve KafkaConnect address from cache")
	}
	return addr.(string), nil
}

// GetCustomMetricsPortName returns the custom metrics port name stored in the provided cache
func GetCustomMetricsPortName(configCache configcache.Cache) (string, error) {
	name, ok := configCache.Load(customMetricsPortNameCacheKey)
	if !ok {
		return "", errors.New("Could not retrieve the custom metrics port name from cache")
	}
	return name.(string), nil
}

// GetCustomMetricsPort returns the custom metrics port in the provided cache
func GetCustomMetricsPort(configCache configcache.Cache) (int, error) {
	port, ok := configCache.Load(customMetricsPortCacheKey)
	if !ok {
		return -1, errors.New("Could not retrieve custom metrics port from cache")
	}
	return port.(int), nil
}

// GetReconcilePeriod returns the reconciliation interval stored in the provided cache
func GetReconcilePeriod(configCache configcache.Cache) (int, error) {
	addr, ok := configCache.Load(reconcilePeriodCacheKey)
	if !ok {
		return -1, errors.New("Could not retrieve reconciliation interval from cache")
	}
	return addr.(int), nil
}

// GetMaxConnectorRestarts returns the maximum allowed connector restart count stored
// in the provided cache
func GetMaxConnectorRestarts(configCache configcache.Cache) (int, error) {
	addr, ok := configCache.Load(maxConnectorRestartsCacheKey)
	if !ok {
		return -1, errors.New("Could not retrieve maximum connector restart count from cache")
	}
	return addr.(int), nil
}

// GetMaxConnectorHardResets returns the maximum allowed connector hard reset count stored
// in the provided cache
func GetMaxConnectorHardResets(configCache configcache.Cache) (int, error) {
	addr, ok := configCache.Load(maxConnectorHardResetsCacheKey)
	if !ok {
		return -1, errors.New("Could not retrieve maximum connector hard reset count from cache")
	}
	return addr.(int), nil
}

// GetMaxTaskRestarts returns the maximum allowed task restart count stored in the provided cache
func GetMaxTaskRestarts(configCache configcache.Cache) (int, error) {
	addr, ok := configCache.Load(maxTaskRestartsCacheKey)
	if !ok {
		return -1, errors.New("Could not retrieve maximum task restart count from cache")
	}
	return addr.(int), nil
}
