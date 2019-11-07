package controller

import (
	"github.com/aerdei/namespace-scc-operator/pkg/controller/namespacescc"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, namespacescc.Add)
}
