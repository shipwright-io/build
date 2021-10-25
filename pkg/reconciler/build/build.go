// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"
	"encoding/json"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	buildmetrics "github.com/shipwright-io/build/pkg/metrics"
	"github.com/shipwright-io/build/pkg/validate"

	taskrunapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersapi "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

// ReconcileBuild reconciles a Build object
type ReconcileBuild struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	ctx                   context.Context
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(ctx context.Context, c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuild{
		ctx:                   ctx,
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
func (r *ReconcileBuild) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "start reconciling Build", namespace, request.Namespace, name, request.Name)

	b := &build.Build{}
	err := r.client.Get(ctx, request.NamespacedName, b)
	if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	} else if apierrors.IsNotFound(err) {
		ctxlog.Debug(ctx, "finish reconciling build. build was not found", namespace, request.Namespace, name, request.Name)
		return reconcile.Result{}, nil
	}

	// Populate the status struct with default values
	b.Status.Registered = corev1.ConditionFalse
	b.Status.Reason = build.SucceedStatus

	// build a list of current validation types
	validationTypes := []string{
		validate.OwnerReferences,
		validate.SourceURL,
		validate.Secrets,
		validate.Strategies,
		validate.Sources,
		validate.BuildName,
		validate.Envs,
	}

	// trigger all current validations
	for _, validationType := range validationTypes {
		v, err := validate.NewValidation(validationType, b, r.client, r.scheme)
		if err != nil {
			// when the validation type is unknown
			return reconcile.Result{}, err
		}

		if err := v.ValidatePath(ctx); err != nil {
			// We enqueue another reconcile here. This is done only for validation
			// types where the error can be produced from a failed API call.
			if validationType == validate.Secrets || validationType == validate.Strategies {
				return reconcile.Result{}, err
			}
			if validationType == validate.OwnerReferences {
				// we do not want to bail out here if the owerreference validation fails, we ignore this error on purpose
				// In case we just created the Build, we want the Build reconcile logic to continue, in order to
				// validate the Build references ( e.g secrets, strategies )
				ctxlog.Info(ctx, "unexpected error during ownership reference validation", namespace, request.Namespace, name, request.Name, "error", err)
			}
		}
		if b.Status.Reason != build.SucceedStatus {
			return r.UpdateBuildStatusAndRetreat(ctx, b)
		}
	}

	if !strings.Contains(b.Name, "webhook") {
		// If the name of the Build resource contains the substring
		// "webhook", it implies that this was created out of a webhook event.
		// Really, poor man's way of avoiding an infinite loop.

		resourceTemplateTaskRun := taskrunapi.TaskRun{
			ObjectMeta: v1.ObjectMeta{
				GenerateName: b.Name + "-",
				Namespace:    b.Namespace,
			},
			TypeMeta: v1.TypeMeta{
				Kind:       "TaskRun",
				APIVersion: taskrunapi.SchemeGroupVersion.Group + "/" + taskrunapi.SchemeGroupVersion.Version,
			},
			Spec: taskrunapi.TaskRunSpec{
				Params: []taskrunapi.Param{
					{
						Name: "git_revision",
						Value: taskrunapi.ArrayOrString{
							Type:      taskrunapi.ParamTypeString,
							StringVal: "$(tt.params.git-revision)",
						},
					},
					{
						Name: "git_tree",
						Value: taskrunapi.ArrayOrString{
							Type:      taskrunapi.ParamTypeString,
							StringVal: "$(tt.params.git-tree)",
						},
					},
				},

				// ClusterTask "shipwright-executor" is a ClusterTask which
				// creates the appropriate BuildRun ( and Build, since embedded buildSpec is not
				// supported in BuildRuns yet ).
				TaskRef: &taskrunapi.TaskRef{
					Name:       "shipwright-executor",
					Kind:       "ClusterTask",
					APIVersion: "tekton.dev/v1beta1",
				},
			},
		}

		resourceTemplateTaskRunBytes, err := json.Marshal(resourceTemplateTaskRun)
		if err != nil {
			// fail silently for now.
			ctxlog.Error(ctx, err, "Failed to convert TaskRun resource", namespace, request.Namespace, name, request.Name)
		}

		triggerTemplate := triggersapi.TriggerTemplate{}
		triggerTemplate.Name = b.Name
		triggerTemplate.Namespace = b.Namespace
		triggerTemplate.Spec = triggersapi.TriggerTemplateSpec{
			Params: []triggersapi.ParamSpec{
				{
					Name: "git-revision",
				},
				{
					Name: "git-tree",
				},
			},
			ResourceTemplates: []triggersapi.TriggerResourceTemplate{
				{
					RawExtension: runtime.RawExtension{Raw: resourceTemplateTaskRunBytes},
				},
			},
		}

		err = r.client.Create(ctx, &triggerTemplate)
		if err != nil {
			// fail silently for now.
			ctxlog.Error(ctx, err, "Failed to create TriggerTemplate", namespace, request.Namespace, name, request.Name)
		}

		eventListener := triggersapi.EventListener{
			ObjectMeta: v1.ObjectMeta{
				Name:      b.Name,
				Namespace: b.Namespace,
			},
			Spec: triggersapi.EventListenerSpec{
				// If left empty, the "default" service account would be used.
				// Should leave his empty?
				ServiceAccountName: "pipeline",
				Triggers: []triggersapi.EventListenerTrigger{
					{
						Bindings: []*triggersapi.TriggerSpecBinding{
							{
								// ClusterTriggerBinding "shipwright-executor" will need
								// to be shipped as part of the Shipwright installation.
								Ref:  "shipwright-executor",
								Kind: triggersapi.ClusterTriggerBindingKind,
							},
						},
						Template: &triggersapi.TriggerSpecTemplate{
							Ref: &triggerTemplate.Name,
						},
					},
				},
			},
		}

		err = r.client.Create(ctx, &eventListener)
		if err != nil {
			ctxlog.Error(ctx, err, "Failed to create EventListener", namespace, request.Namespace, name, request.Name)
		}
	}

	b.Status.Registered = corev1.ConditionTrue
	b.Status.Message = build.AllValidationsSucceeded
	err = r.client.Status().Update(ctx, b)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Increase Build count in metrics
	buildmetrics.BuildCountInc(b.Spec.Strategy.Name, b.Namespace, b.Name)

	ctxlog.Debug(ctx, "finishing reconciling Build", namespace, request.Namespace, name, request.Name)
	return reconcile.Result{}, nil
}

// UpdateBuildStatusAndRetreat returns an error if an update fails, this should force
// a new reconcile until the API call succeeds. If return is nil, no further reconciliations
// will take place
func (r *ReconcileBuild) UpdateBuildStatusAndRetreat(ctx context.Context, b *build.Build) (reconcile.Result, error) {
	if err := r.client.Status().Update(ctx, b); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}
