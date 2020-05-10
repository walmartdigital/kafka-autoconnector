package controller

import (
	"github.com/chinniehendrix/go-kaya/pkg/kafkaconnect"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, kafkaconnect.KafkaConnectClientFactory) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, fact kafkaconnect.KafkaConnectClientFactory) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, fact); err != nil {
			return err
		}
	}
	return nil
}
