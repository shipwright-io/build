// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

type ContainNamedElementMatcher struct {
	Name string
}

func (matcher *ContainNamedElementMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to contain element with name", matcher.Name)
}

func (matcher *ContainNamedElementMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to contain element with name", matcher.Name)
}

func (matcher *ContainNamedElementMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, nil
	}

	kind := reflect.TypeOf(actual).Kind()
	if kind == reflect.Array || kind == reflect.Slice {
		value := reflect.ValueOf(actual)

		for i := 0; i < value.Len(); i++ {
			vItem := value.Index(i)
			vName := vItem.FieldByName("Name")
			if matcher.Name == vName.String() {
				return true, nil
			}
		}
	}

	return false, nil
}

// ContainNamedElement can be applied for an array or slice of objects which have a Name field, to check if any item has a matching name
func ContainNamedElement(name string) types.GomegaMatcher {
	return &ContainNamedElementMatcher{
		Name: name,
	}
}
