package controller

import (
	"github.com/walmartdigital/kafka-autoconnector/pkg/controller/genericautoconnector"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, genericautoconnector.Add)
}
