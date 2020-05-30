package controller

import (
	"github.com/walmartdigital/go-kaya/pkg/kafkaconnect"
	"github.com/walmartdigital/kafka-autoconnector/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, cache.Cache, kafkaconnect.KafkaConnectClientFactory) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, c cache.Cache, fact kafkaconnect.KafkaConnectClientFactory) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, c, fact); err != nil {
			return err
		}
	}
	return nil
}
