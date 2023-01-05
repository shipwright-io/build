// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	. "github.com/shipwright-io/build/cmd/git"
	shpgit "github.com/shipwright-io/build/pkg/git"
)

func TestGitCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Git Command Suite")
}

type errorClassMatcher struct{ expected shpgit.ErrorClass }

func FailWith(expected shpgit.ErrorClass) types.GomegaMatcher {
	return &errorClassMatcher{expected: expected}
}

func (m *errorClassMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, nil
	}

	switch obj := actual.(type) {
	case *ExitError:
		return obj.Reason == m.expected, nil

	case shpgit.ErrorClass:
		return obj == m.expected, nil

	default:
		return false, fmt.Errorf("type mismatch: %T", actual)
	}
}

func (m *errorClassMatcher) asStrings(actual interface{}) (string, string) {
	switch obj := actual.(type) {
	case *ExitError:
		return obj.Reason.String(), m.expected.String()

	case shpgit.ErrorClass:
		return obj.String(), m.expected.String()

	default:
		return fmt.Sprintf("%v", obj), m.expected.String()
	}
}

func (m *errorClassMatcher) FailureMessage(actual interface{}) string {
	act, exp := m.asStrings(actual)
	return fmt.Sprintf("\nExpected\n\t%s\nto equal\n\t%s", act, exp)
}

func (m *errorClassMatcher) NegatedFailureMessage(actual interface{}) string {
	act, exp := m.asStrings(actual)
	return fmt.Sprintf("\nExpected\n\t%s\nto not equal\n\t%s", act, exp)
}
