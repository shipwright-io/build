// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

type containNamedElementMatcher struct {
	Name string
}

func (matcher *containNamedElementMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to contain element with name", matcher.Name)
}

func (matcher *containNamedElementMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to contain element with name", matcher.Name)
}

func (matcher *containNamedElementMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, nil
	}

	kind := reflect.TypeOf(actual).Kind()
	if kind == reflect.Array || kind == reflect.Slice {
		value := reflect.ValueOf(actual)

		for i := 0; i < value.Len(); i++ {
			vItem := value.Index(i)
			vName := vItem.FieldByName("Name")
			if !vName.IsZero() && matcher.Name == vName.String() {
				return true, nil
			}
		}
	}

	return false, nil
}

// ContainNamedElement can be applied for an array or slice of objects which have a Name field, to check if any item has a matching name
func ContainNamedElement(name string) types.GomegaMatcher {
	return &containNamedElementMatcher{
		Name: name,
	}
}

type containNamedWithValueElementMatcher struct {
	Name  string
	Value string
}

func (matcher *containNamedWithValueElementMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to contain element with name and value", fmt.Sprintf("%s=%s", matcher.Name, matcher.Value))
}

func (matcher *containNamedWithValueElementMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to contain element with name and value", fmt.Sprintf("%s=%s", matcher.Name, matcher.Value))
}

func (matcher *containNamedWithValueElementMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, nil
	}

	kind := reflect.TypeOf(actual).Kind()
	if kind == reflect.Array || kind == reflect.Slice {
		value := reflect.ValueOf(actual)

		for i := 0; i < value.Len(); i++ {
			vItem := value.Index(i)
			vName := vItem.FieldByName("Name")
			if !vName.IsZero() && matcher.Name == vName.String() {
				vValue := vItem.FieldByName("Value")
				if !vValue.IsZero() && matcher.Value == vValue.String() {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// ContainNamedElementWithValue can be applied for an array or slice of objects which have a Name and Value field, to check if any item has a matching name and value
func ContainNamedElementWithValue(name string, value string) types.GomegaMatcher {
	return &containNamedWithValueElementMatcher{
		Name:  name,
		Value: value,
	}
}

type returnMatcher struct {
	actualStatusCode   int
	expectedStatusCode int
}

func (matcher *returnMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(matcher.expectedStatusCode, fmt.Sprintf("to be the HTTP response for %s, but received", actual), matcher.actualStatusCode)
}

func (matcher *returnMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(matcher.expectedStatusCode, fmt.Sprintf("to not be the HTTP response for %s, but received", actual), matcher.actualStatusCode)
}

func (matcher *returnMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, nil
	}

	kind := reflect.TypeOf(actual).Kind()
	if kind == reflect.String {
		url := reflect.ValueOf(actual).String()

		// #nosec:G107 test code
		resp, err := http.Get(url)
		if err != nil {
			return false, err
		}

		matcher.actualStatusCode = resp.StatusCode

		return resp.StatusCode == matcher.expectedStatusCode, nil
	}

	return false, nil
}

// Return can be applied for a string, it will call the URL and check the status code
func Return(statusCode int) types.GomegaMatcher {
	return &returnMatcher{
		expectedStatusCode: statusCode,
	}
}
