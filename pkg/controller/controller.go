package controller

import (
	"github.com/walmartdigital/go-kaya/pkg/kafkaconnect"
	"github.com/walmartdigital/kafka-autoconnector/pkg/cache"
	"github.com/walmartdigital/kafka-autoconnector/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, cache.Cache, metrics.Metrics, kafkaconnect.KafkaConnectClientFactory) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, c cache.Cache, met metrics.Metrics, fact kafkaconnect.KafkaConnectClientFactory) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, c, met, fact); err != nil {
			return err
		}
	}
	return nil
}
