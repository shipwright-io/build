package tektonrun

import (
	"context"
	"fmt"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	knativeapis "knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ReconcileTektonRun struct {
	config       *config.Config
	client       client.Client
	scheme       *runtime.Scheme
	ownerRefFunc setOwnerReferenceFunc
}

func NewReconciler(c *config.Config, mgr manager.Manager, ownerRefFunc setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileTektonRun{
		config:       c,
		client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		ownerRefFunc: ownerRefFunc,
	}
}

func (r *ReconcileTektonRun) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	ctxlog.Debug(ctx, "received reconcile request for tekton run")
	tektonRun := &tektonv1alpha1.Run{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: request.Namespace,
			Name:      request.Name,
		},
	}
	err := r.client.Get(ctx, client.ObjectKeyFromObject(tektonRun), tektonRun)
	if errors.IsNotFound(err) {
		ctxlog.Debug(ctx, "could not find tekton run, no status to update")
		return reconcile.Result{Requeue: false}, nil
	}
	if err != nil {
		ctxlog.Error(ctx, err, "failed to get tekton run")
		return reconcile.Result{Requeue: true}, err
	}

	if tektonRun.IsDone() {
		ctxlog.Debug(ctx, "tekton run is done")
		return reconcile.Result{Requeue: false}, nil
	}

	ctxlog.Info(ctx, "reconciling status for tekton run")

	if err := ValidateTektonRun(tektonRun); err != nil {
		return r.handleRunFailure(ctx, tektonRun, "InvalidRun", err)
	}

	var status ExtraFields
	err = tektonRun.Status.DecodeExtraFields(&status)
	if err != nil {
		ctxlog.Error(ctx, err, "failed to decode extra status fields")
		return reconcile.Result{Requeue: true}, err
	}

	buildrun := &buildv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tektonRun.Namespace,
		},
	}

	if status.IsEmpty() {
		buildrun, err = r.createBuildRun(ctx, tektonRun)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		ctxlog.Info(ctx, "creating builrun for tekton run", "buildrun", buildrun.GetName())

		// recording the BuildRun name created using ExtraFields
		fields := ExtraFields{BuildRunName: buildrun.GetName()}
		if err = tektonRun.Status.EncodeExtraFields(&fields); err != nil {
			ctxlog.Error(ctx, err, "failed to encode extra status fields", "tekton.run", tektonRun.GetName())
			return reconcile.Result{Requeue: true}, err
		}
		now := metav1.Now()
		tektonRun.Status.StartTime = &now
	} else {
		buildrun.Name = status.BuildRunName
		err = r.client.Get(ctx, client.ObjectKeyFromObject(buildrun), buildrun)
		if err != nil {
			ctxlog.Error(ctx, err, "failed to get BuildRun", "buildrun", status.BuildRunName, "tekton.run", tektonRun.GetName())
			return reconcile.Result{Requeue: true}, err
		}
		ctxlog.Info(ctx, "Updating Tekton Run status with BuildRun", "tekton.run", tektonRun.GetName(), "buildrun", buildrun.GetName())
	}

	result, err := r.updateRunStatus(ctx, tektonRun, buildrun)
	ctxlog.Info(ctx, "finished reconcile request for tekton Run", "tekton.run", request.Name)
	return result, err
}

func (r *ReconcileTektonRun) handleRunFailure(ctx context.Context, tektonRun *tektonv1alpha1.Run, reason string, err error) (reconcile.Result, error) {
	ctxlog.Debug(ctx, "handling run failure")
	now := metav1.Now()
	if tektonRun.Status.StartTime == nil {
		tektonRun.Status.StartTime = &now
	}
	if tektonRun.Status.CompletionTime == nil {
		tektonRun.Status.CompletionTime = &now
	}
	tektonRun.Status.SetCondition(&knativeapis.Condition{
		Type:    knativeapis.ConditionSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: err.Error(),
	})
	if err := r.client.Status().Update(ctx, tektonRun, &client.UpdateOptions{}); err != nil {
		ctxlog.Error(ctx, err, "failed to update run status")
		return reconcile.Result{Requeue: true}, err
	}
	ctxlog.Info(ctx, "updated run status due to failure", "reason", reason)
	return reconcile.Result{Requeue: false}, nil
}

func (r *ReconcileTektonRun) createBuildRun(ctx context.Context, tektonRun *tektonv1alpha1.Run) (*buildv1alpha1.BuildRun, error) {
	buildRun := &buildv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", tektonRun.Name),
			Namespace:    tektonRun.Namespace,
		},
	}
	if tektonRun.Spec.Ref != nil {
		buildRun.Spec.BuildRef = &buildv1alpha1.BuildRef{
			Name: tektonRun.Spec.Ref.Name,
		}
	}
	err := r.ownerRefFunc(tektonRun, buildRun, r.scheme)
	if err != nil {
		return nil, err
	}
	err = r.client.Create(ctx, buildRun, &client.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return buildRun, nil
}

// updateRunStatus reflect the BuildRun status into the Tekton Run resource.
func (r *ReconcileTektonRun) updateRunStatus(ctx context.Context, tektonRun *tektonv1alpha1.Run, buildrun *buildv1alpha1.BuildRun) (reconcile.Result, error) {
	tektonRun.Status.CompletionTime = buildrun.Status.CompletionTime

	for _, condition := range buildrun.Status.Conditions {
		severity := knativeapis.ConditionSeverityInfo
		if condition.Status == corev1.ConditionFalse {
			severity = knativeapis.ConditionSeverityError
		}
		tektonRun.Status.SetCondition(&knativeapis.Condition{
			Type:               knativeapis.ConditionType(string(condition.Type)),
			Status:             condition.Status,
			LastTransitionTime: knativeapis.VolatileTime{Inner: condition.LastTransitionTime},
			Reason:             condition.Reason,
			Message:            condition.Message,
			Severity:           severity,
		})
	}

	if len(tektonRun.Status.Conditions) == 0 {
		tektonRun.Status.SetCondition(&knativeapis.Condition{
			Type:   knativeapis.ConditionSucceeded,
			Status: corev1.ConditionUnknown,
		})
	}

	err := r.client.Status().Update(ctx, tektonRun, &client.UpdateOptions{})
	if err != nil {
		ctxlog.Error(ctx, err, "failed to update run status")
		return reconcile.Result{Requeue: true}, err
	}
	ctxlog.Debug(ctx, "updated run status")
	return reconcile.Result{Requeue: false}, nil
}
