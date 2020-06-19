package config

import (
	"errors"
	"os"
	"strconv"

	configcache "github.com/walmartdigital/kafka-autoconnector/pkg/cache"
)

var (
	kafkaConnectHost                = "192.168.64.5:30256"
	refreshFromKafkaConnectInterval = 5
	maxConnectorRestarts            = 5
	maxTaskRestarts                 = 5
	maxConnectorHardResets          = 3
	kafkaConnectAddrKey             = "/config/global/kafkaconnect/address"
	reconcilePeriodKey              = "/config/global/reconcile/period"
	maxConnectorRestartsKey         = "/config/global/connectors/maxrestarts"
	maxConnectorHardResetsKey       = "/config/global/connectors/maxhardresets"
	maxTaskRestartsKey              = "/config/global/tasks/maxrestarts"
)

// CacheConfig ...
func CacheConfig(configCache *configcache.Cache) {
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

	(*configCache).Store(kafkaConnectAddrKey, kafkaConnectHost)
	(*configCache).Store(reconcilePeriodKey, refreshFromKafkaConnectInterval)
	(*configCache).Store(maxConnectorRestartsKey, maxConnectorRestarts)
	(*configCache).Store(maxConnectorHardResetsKey, maxConnectorHardResets)
	(*configCache).Store(maxTaskRestartsKey, kafkaConnectHost)
}
