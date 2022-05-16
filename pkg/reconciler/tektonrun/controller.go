package tektonrun

import (
	"context"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type setOwnerReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "tektonrun-controller")
	return add(ctx, mgr, NewReconciler(c, mgr, controllerutil.SetOwnerReference))
}

func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler) error {
	err := builder.ControllerManagedBy(mgr).
		Named("tektonrun-controller").
		For(&tektonv1alpha1.Run{}).
		WithEventFilter(predicate.NewPredicateFuncs(filterTaskRun)).
		Owns(&buildv1alpha1.BuildRun{}).
		Complete(r)
	return err
}

func filterTaskRun(o client.Object) bool {
	r, isRun := o.(*tektonv1alpha1.Run)
	// If this is not a TaskRun, it should be an owned BuildRun event that is processed.
	if !isRun {
		return true
	}
	if r.Spec.Ref != nil {
		return r.Spec.Ref.Kind == "Build" && r.Spec.Ref.APIVersion == "shipwright.io/v1alpha1"
	}
	if r.Spec.Spec != nil {
		return r.Spec.Spec.Kind == "Build" && r.Spec.Spec.APIVersion == "shipwright.io/v1alpha1"
	}
	return true
}
