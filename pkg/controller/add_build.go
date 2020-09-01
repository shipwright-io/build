// Copyright The Shipwright Contributors
// 
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"github.com/shipwright-io/build/pkg/controller/build"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, build.Add)
}
