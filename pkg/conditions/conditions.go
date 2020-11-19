// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package conditions

//
// This class is heavily based on
// https://github.com/knative/pkg/blob/master/apis/condition_set.go
// but contains a more simplified approach.
//
// This class is intended to enable any Build CRD that might require to
// operate and support Conditions.
//
// For any Build CRD that requires to operate on Conditions, they should
// only need to implement the StatusConditions interface.
//

// StatusConditions provides access to the conditions of an
// object that have a Status field
type StatusConditions interface {
	GetConditions() *Conditions
	SetConditions(Conditions)
}

// Access is the interface that allows retrieval
// of a particular condition
type Access interface {
	GetCondition(t Type) *Condition
}

// Manager is the interface that allows to operate
// on a particular condition, by getting or setting it
type Manager interface {
	Access
	SetCondition(*Condition)
}

// Implementor implements the Manager interface
type Implementor struct {
	Connect StatusConditions
}

// Verify that Implementor implements Manager
var _ Manager = (*Implementor)(nil)

// Manage enables an object that implements the
// StatusConditions interface to get access to the Manager
func Manage(status StatusConditions) Manager {
	return Implementor{
		Connect: status,
	}
}

// GetCondition retrieves a particular condition based
// on the type
func (i Implementor) GetCondition(t Type) *Condition {
	if i.Connect == nil {
		return nil
	}

	for _, c := range *i.Connect.GetConditions() {
		if c.Type == t {
			return &c
		}
	}

	return nil
}

// SetCondition updates a condition by generating the same list
// of conditions with the provided one
// This does not preserve the order when multiple conditions exist.
func (i Implementor) SetCondition(aCondition *Condition) {
	if i.Connect == nil {
		return
	}

	var conditions Conditions

	for _, c := range *i.Connect.GetConditions() {
		if c.Type != aCondition.Type {
			conditions = append(conditions, c)
		}
	}
	conditions = append(conditions, *aCondition)

	i.Connect.SetConditions(conditions)

}
