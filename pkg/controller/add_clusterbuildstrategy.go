package controller

import (
	"github.com/redhat-developer/build/pkg/controller/clusterbuildstrategy"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, clusterbuildstrategy.Add)
}
