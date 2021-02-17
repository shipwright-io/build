// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"
	"strings"
)

// HandleError returns multiple errors if each error is not nil.
// And its error message.
func HandleError(message string, listOfErrors ...error) error {
	var errSlice []string
	for _, e := range listOfErrors {
		if e != nil {
			errSlice = append(errSlice, e.Error())
		}
	}
	return fmt.Errorf("errors: %s, msg: %s", strings.Join(errSlice, ", "), message)
}
