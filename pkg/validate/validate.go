package validate

import (
	"context"
	"fmt"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Secrets for validating secret references in Build objects
	Secrets = "secrets"
	// Strategies for validating strategy references in Build objects
	Strategies = "strategy"
	// SourceURL for validating the source URL in Build objects
	SourceURL = "sourceurl"
	// Runtime for validating the runtime definition in Build objects
	Runtime = "runtime"
	// OwnerReferences for validating the ownerreferences between a Build
	// and BuildRun objects
	OwnerReferences = "ownerreferences"
	namespace       = "namespace"
	name            = "name"
)

// BuildPath is an interface that holds a ValidaPath() function
// for validating different Build spec paths
type BuildPath interface {
	ValidatePath(ctx context.Context) error
}

// NewValidation returns an specific Structure that implements
// BuildPath interface
func NewValidation(
	validationType string,
	build *build.Build,
	client client.Client,
	scheme *runtime.Scheme,
) (BuildPath, error) {
	secretRef := SecretRef{
		Build:  build,
		Client: client,
	}
	strategyRef := StrategyRef{
		Build:  build,
		Client: client,
	}
	sourceURLRef := SourceURLRef{
		Build:  build,
		Client: client,
	}
	runtimeSpecRef := RuntimeRef{
		Build:  build,
		Client: client,
	}
	ownerRef := OwnerRef{
		Build:  build,
		Client: client,
		Scheme: scheme,
	}
	switch validationType {
	case Secrets:
		return secretRef, nil
	case Strategies:
		return strategyRef, nil
	case SourceURL:
		return sourceURLRef, nil
	case Runtime:
		return runtimeSpecRef, nil
	case OwnerReferences:
		return ownerRef, nil
	default:
		return nil, fmt.Errorf("unknown validation type")
	}
}
