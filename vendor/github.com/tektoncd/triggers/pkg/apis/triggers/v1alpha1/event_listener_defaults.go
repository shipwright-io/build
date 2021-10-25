/*
Copyright 2019 The Tekton Authors

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
	"context"

	"knative.dev/pkg/logging"
	"knative.dev/pkg/ptr"
)

// SetDefaults sets the defaults on the object.
func (el *EventListener) SetDefaults(ctx context.Context) {
	if IsUpgradeViaDefaulting(ctx) {
		// set defaults
		if el.Spec.Resources.KubernetesResource != nil {
			if el.Spec.Resources.KubernetesResource.Replicas != nil && *el.Spec.Resources.KubernetesResource.Replicas == 0 {
				*el.Spec.Resources.KubernetesResource.Replicas = 1
			}
		}

		for i, t := range el.Spec.Triggers {
			triggerSpecBindingArray(el.Spec.Triggers[i].Bindings).defaultBindings()
			for _, ti := range t.Interceptors {
				ti.defaultInterceptorKind()
				if err := ti.updateCoreInterceptors(); err != nil {
					// The err only happens due to malformed JSON and should never really happen
					// We can't return an error here, so print out the error
					logger := logging.FromContext(ctx)
					logger.Errorf("failed to setDefaults for trigger: %s; err: %s", t.Name, err)
				}
			}
		}
		el.Spec.updatePodTemplate()
		// To be removed in a later release #1020
		el.Spec.updateReplicas()
	}
}

func (spec *EventListenerSpec) updatePodTemplate() {
	if spec.DeprecatedPodTemplate != nil {
		if spec.DeprecatedPodTemplate.NodeSelector != nil {
			if spec.Resources.KubernetesResource == nil {
				spec.Resources.KubernetesResource = &KubernetesResource{}
			}
			spec.Resources.KubernetesResource.Template.Spec.NodeSelector = spec.DeprecatedPodTemplate.NodeSelector
			spec.DeprecatedPodTemplate.NodeSelector = nil
		}
		if spec.DeprecatedPodTemplate.Tolerations != nil {
			if spec.Resources.KubernetesResource == nil {
				spec.Resources.KubernetesResource = &KubernetesResource{}
			}
			spec.Resources.KubernetesResource.Template.Spec.Tolerations = spec.DeprecatedPodTemplate.Tolerations
			spec.DeprecatedPodTemplate.Tolerations = nil
		}
		spec.DeprecatedPodTemplate = nil
	}
}

// To be Removed in a later release #1020
func (spec *EventListenerSpec) updateReplicas() {
	if spec.DeprecatedReplicas != nil {
		if *spec.DeprecatedReplicas == 0 {
			if spec.Resources.KubernetesResource == nil {
				spec.Resources.KubernetesResource = &KubernetesResource{}
			}
			spec.Resources.KubernetesResource.Replicas = ptr.Int32(1)
			spec.DeprecatedReplicas = nil
		} else if *spec.DeprecatedReplicas > 0 {
			if spec.Resources.KubernetesResource == nil {
				spec.Resources.KubernetesResource = &KubernetesResource{}
			}
			spec.Resources.KubernetesResource.Replicas = spec.DeprecatedReplicas
			spec.DeprecatedReplicas = nil
		}
	}
}
