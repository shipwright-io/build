/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"encoding/json"
)

// ToEventListenerTrigger converts a TriggerSpec into an EventListenerTrigger.
// This is primarily for compatibility between CRD and non-CRD types so that
// underlying libraries can reuse existing code.
func ToEventListenerTrigger(in TriggerSpec) (EventListenerTrigger, error) {
	var out EventListenerTrigger

	// Use json Marshalling in order to be field agnostic. Since TriggerSpec
	// is a subset of the existing EventListenerTrigger type, and should always
	// contain the same field labels, this should be safe to do.
	b, err := json.Marshal(in)
	if err != nil {
		return out, err
	}

	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}
