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
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

var (
	reservedEnvVars = sets.NewString(
		"TLS_CERT",
		"TLS_KEY",
	)
)

// Validate EventListener.
func (e *EventListener) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError
	if len(e.ObjectMeta.Name) > 60 {
		// Since `el-` is added as the prefix of EventListener services, the name of EventListener must be no more than 60 characters long.
		errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("eventListener name '%s' must be no more than 60 characters long", e.ObjectMeta.Name), "metadata.name"))
	}
	if apis.IsInDelete(ctx) {
		return nil
	}
	return errs.Also(e.Spec.validate(ctx))
}

func (s *EventListenerSpec) validate(ctx context.Context) (errs *apis.FieldError) {
	for i, trigger := range s.Triggers {
		errs = errs.Also(trigger.validate(ctx).ViaField(fmt.Sprintf("spec.triggers[%d]", i)))
	}

	// To be removed in a later release #1020
	if s.DeprecatedReplicas != nil {
		if *s.DeprecatedReplicas < 0 {
			errs = errs.Also(apis.ErrInvalidValue(*s.DeprecatedReplicas, "spec.replicas"))
		}
	}

	// Both Kubernetes and Custom resource can't be present at the same time
	if s.Resources.KubernetesResource != nil && s.Resources.CustomResource != nil {
		return apis.ErrMultipleOneOf("spec.resources.kubernetesResource", "spec.resources.customResource")
	}

	if s.Resources.KubernetesResource != nil {
		errs = errs.Also(validateKubernetesObject(s.Resources.KubernetesResource).ViaField("spec.resources.kubernetesResource"))
	}

	if s.Resources.CustomResource != nil {
		errs = errs.Also(validateCustomObject(s.Resources.CustomResource).ViaField("spec.resources.customResource"))
	}
	return errs
}

func validateCustomObject(customData *CustomResource) (errs *apis.FieldError) {
	orig := duckv1.WithPod{}
	decoder := json.NewDecoder(bytes.NewBuffer(customData.RawExtension.Raw))

	if err := decoder.Decode(&orig); err != nil {
		errs = errs.Also(apis.ErrInvalidValue(err, "spec"))
	}

	if len(orig.Spec.Template.Spec.Containers) > 1 {
		errs = errs.Also(apis.ErrMultipleOneOf("containers").ViaField("spec.template.spec"))
	}
	errs = errs.Also(apis.CheckDisallowedFields(orig.Spec.Template.Spec,
		*podSpecMask(&orig.Spec.Template.Spec)).ViaField("spec.template.spec"))

	// bounded by condition because containers fields are optional so there is a chance that containers can be nil.
	if len(orig.Spec.Template.Spec.Containers) == 1 {
		errs = errs.Also(apis.CheckDisallowedFields(orig.Spec.Template.Spec.Containers[0],
			*containerFieldMask(&orig.Spec.Template.Spec.Containers[0])).ViaField("spec.template.spec.containers[0]"))
		// validate env
		errs = errs.Also(validateEnv(orig.Spec.Template.Spec.Containers[0].Env).ViaField("spec.template.spec.containers[0].env"))
	}

	return errs
}

func validateKubernetesObject(orig *KubernetesResource) (errs *apis.FieldError) {
	if orig.Replicas != nil {
		if *orig.Replicas < 0 {
			errs = errs.Also(apis.ErrInvalidValue(*orig.Replicas, "spec.replicas"))
		}
	}
	if len(orig.Template.Spec.Containers) > 1 {
		errs = errs.Also(apis.ErrMultipleOneOf("containers").ViaField("spec.template.spec"))
	}
	errs = errs.Also(apis.CheckDisallowedFields(orig.Template.Spec,
		*podSpecMask(&orig.Template.Spec)).ViaField("spec.template.spec"))

	// bounded by condition because containers fields are optional so there is a chance that containers can be nil.
	if len(orig.Template.Spec.Containers) == 1 {
		errs = errs.Also(apis.CheckDisallowedFields(orig.Template.Spec.Containers[0],
			*containerFieldMask(&orig.Template.Spec.Containers[0])).ViaField("spec.template.spec.containers[0]"))
		// validate env
		errs = errs.Also(validateEnv(orig.Template.Spec.Containers[0].Env).ViaField("spec.template.spec.containers[0].env"))
	}

	return errs
}

func validateEnv(envVars []corev1.EnvVar) (errs *apis.FieldError) {
	var (
		count    = 0
		envValue string
	)
	for i, env := range envVars {
		errs = errs.Also(validateEnvVar(env).ViaIndex(i))
		if reservedEnvVars.Has(env.Name) {
			count++
			envValue = env.Name
		}
	}
	// This is to make sure both TLS_CERT and TLS_KEY is set for tls connection
	if count == 1 {
		errs = errs.Also(&apis.FieldError{
			Message: fmt.Sprintf("Expected env's are TLS_CERT and TLS_KEY, but got only one env %s", envValue),
		})
	}
	return errs
}

func validateEnvVar(env corev1.EnvVar) (errs *apis.FieldError) {
	errs = errs.Also(apis.CheckDisallowedFields(env, *envVarMask(&env)))

	return errs.Also(validateEnvValueFrom(env.ValueFrom).ViaField("valueFrom"))
}

func validateEnvValueFrom(source *corev1.EnvVarSource) *apis.FieldError {
	if source == nil {
		return nil
	}
	return apis.CheckDisallowedFields(*source, *envVarSourceMask(source))
}

// envVarSourceMask performs a _shallow_ copy of the Kubernetes EnvVarSource object to a new
// Kubernetes EnvVarSource object bringing over only the fields allowed in the Triggers EventListener API.
func envVarSourceMask(in *corev1.EnvVarSource) *corev1.EnvVarSource {
	if in == nil {
		return nil
	}
	out := new(corev1.EnvVarSource)
	// Allowed fields
	out.SecretKeyRef = in.SecretKeyRef

	// Disallowed fields
	out.ConfigMapKeyRef = nil
	out.FieldRef = nil
	out.ResourceFieldRef = nil

	return out
}

// envVarMask performs a _shallow_ copy of the Kubernetes EnvVar object to a new
// Kubernetes EnvVar object bringing over only the fields allowed in the Triggers EventListener API.
func envVarMask(in *corev1.EnvVar) *corev1.EnvVar {
	if in == nil {
		return nil
	}
	out := new(corev1.EnvVar)
	// Allowed fields
	out.Name = in.Name
	out.ValueFrom = in.ValueFrom

	// Disallowed fields
	out.Value = ""

	return out
}

func containerFieldMask(in *corev1.Container) *corev1.Container {
	out := new(corev1.Container)
	out.Resources = in.Resources
	out.Env = in.Env

	// Disallowed fields
	// This list clarifies which all container attributes are not allowed.
	out.Name = ""
	out.Image = ""
	out.Args = nil
	out.Ports = nil
	out.LivenessProbe = nil
	out.ReadinessProbe = nil
	out.StartupProbe = nil
	out.Command = nil
	out.VolumeMounts = nil
	out.ImagePullPolicy = ""
	out.Lifecycle = nil
	out.Stdin = false
	out.StdinOnce = false
	out.TerminationMessagePath = ""
	out.TerminationMessagePolicy = ""
	out.WorkingDir = ""
	out.TTY = false
	out.VolumeDevices = nil
	out.EnvFrom = nil

	return out
}

// podSpecMask performs a _shallow_ copy of the Kubernetes PodSpec object to a new
// Kubernetes PodSpec object bringing over only the fields allowed in the Triggers EvenListener.
func podSpecMask(in *corev1.PodSpec) *corev1.PodSpec {
	out := new(corev1.PodSpec)

	// Allowed fields
	out.ServiceAccountName = in.ServiceAccountName
	out.Containers = in.Containers
	out.Tolerations = in.Tolerations
	out.NodeSelector = in.NodeSelector

	// Disallowed fields
	// This list clarifies which all podspec fields are not allowed.
	out.Volumes = nil
	out.EnableServiceLinks = nil
	out.ImagePullSecrets = nil
	out.InitContainers = nil
	out.RestartPolicy = ""
	out.TerminationGracePeriodSeconds = nil
	out.ActiveDeadlineSeconds = nil
	out.DNSPolicy = ""
	out.AutomountServiceAccountToken = nil
	out.NodeName = ""
	out.HostNetwork = false
	out.HostPID = false
	out.HostIPC = false
	out.ShareProcessNamespace = nil
	out.SecurityContext = nil
	out.Hostname = ""
	out.Subdomain = ""
	out.Affinity = nil
	out.SchedulerName = ""
	out.HostAliases = nil
	out.PriorityClassName = ""
	out.Priority = nil
	out.DNSConfig = nil
	out.ReadinessGates = nil
	out.RuntimeClassName = nil

	return out
}

func (t *EventListenerTrigger) validate(ctx context.Context) (errs *apis.FieldError) {
	if t.Template == nil && t.TriggerRef == "" {
		errs = errs.Also(apis.ErrMissingOneOf("template", "triggerRef"))
	}

	if t.TriggerRef != "" && (t.Template != nil || t.Bindings != nil || t.Interceptors != nil) {
		errs = errs.Also(apis.ErrMultipleOneOf("triggerRef", "template or bindings or interceptors"))
	}

	// Validate optional Bindings
	errs = errs.Also(triggerSpecBindingArray(t.Bindings).validate(ctx))
	if t.Template != nil {
		// Validate required TriggerTemplate
		errs = errs.Also(t.Template.validate(ctx))
	}

	// Validate optional Interceptors
	for i, interceptor := range t.Interceptors {
		errs = errs.Also(interceptor.validate(ctx).ViaField(fmt.Sprintf("interceptors[%d]", i)))
	}

	// The trigger name is added as a label value for 'tekton.dev/trigger' so it must follow the k8s label guidelines:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
	if err := validation.IsValidLabelValue(t.Name); len(err) > 0 {
		errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("trigger name '%s' must be a valid label value", t.Name), "name"))
	}

	return errs
}
