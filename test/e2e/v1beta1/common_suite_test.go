// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

const (
	pollCreateInterval = 1 * time.Second
	pollCreateTimeout  = 10 * time.Second
)

type buildPrototype struct{ build buildv1beta1.Build }
type buildRunPrototype struct{ buildRun buildv1beta1.BuildRun }

func NewBuildPrototype() *buildPrototype {
	return &buildPrototype{
		build: buildv1beta1.Build{},
	}
}

func (b *buildPrototype) Name(name string) *buildPrototype {
	b.build.ObjectMeta.Name = name
	return b
}

func (b *buildPrototype) Namespace(namespace string) *buildPrototype {
	b.build.ObjectMeta.Namespace = namespace
	return b
}

func (b *buildPrototype) BuildStrategy(name string) *buildPrototype {
	var bs = buildv1beta1.NamespacedBuildStrategyKind
	b.build.Spec.Strategy = buildv1beta1.Strategy{
		Kind: &bs,
		Name: name,
	}
	return b
}

func (b *buildPrototype) ClusterBuildStrategy(name string) *buildPrototype {
	var cbs = buildv1beta1.ClusterBuildStrategyKind
	b.build.Spec.Strategy = buildv1beta1.Strategy{
		Kind: &cbs,
		Name: name,
	}
	return b
}

func (b *buildPrototype) SourceCredentials(name string) *buildPrototype {
	if name != "" {
		if b.build.Spec.Source.GitSource == nil {
			b.build.Spec.Source.GitSource = &buildv1beta1.Git{}
		}
		b.build.Spec.Source.GitSource.CloneSecret = &name
	}

	return b
}

func (b *buildPrototype) SourceGit(repository string) *buildPrototype {
	if b.build.Spec.Source.GitSource == nil {
		b.build.Spec.Source.GitSource = &buildv1beta1.Git{}
	}
	b.build.Spec.Source.GitSource.URL = pointer.String(repository)
	b.build.Spec.Source.OCIArtifact = nil
	return b
}

func (b *buildPrototype) SourceBundle(image string) *buildPrototype {
	if b.build.Spec.Source.OCIArtifact == nil {
		b.build.Spec.Source.OCIArtifact = &buildv1beta1.OCIArtifact{}
	}
	b.build.Spec.Source.Type = buildv1beta1.OCIArtifactType
	b.build.Spec.Source.OCIArtifact.Image = image
	return b
}

func (b *buildPrototype) SourceBundlePrune(prune buildv1beta1.PruneOption) *buildPrototype {
	if b.build.Spec.Source.OCIArtifact == nil {
		b.build.Spec.Source.OCIArtifact = &buildv1beta1.OCIArtifact{}
	}
	b.build.Spec.Source.OCIArtifact.Prune = &prune
	return b
}

func (b *buildPrototype) SourceContextDir(contextDir string) *buildPrototype {
	b.build.Spec.Source.ContextDir = pointer.String(contextDir)
	return b
}

func (b *buildPrototype) Dockerfile(dockerfile string) *buildPrototype {
	b.build.Spec.ParamValues = append(
		b.build.Spec.ParamValues,
		buildv1beta1.ParamValue{
			Name: "dockerfile",
			SingleValue: &buildv1beta1.SingleValue{
				Value: &dockerfile,
			},
		},
	)
	return b
}

func (b *buildPrototype) OutputImage(image string) *buildPrototype {
	b.build.Spec.Output.Image = image
	return b
}

func (b *buildPrototype) determineParameterIndex(name string) int {
	index := -1
	for i, paramValue := range b.build.Spec.ParamValues {
		if paramValue.Name == name {
			index = i
			break
		}
	}

	if index == -1 {
		index = len(b.build.Spec.ParamValues)
		b.build.Spec.ParamValues = append(b.build.Spec.ParamValues, buildv1beta1.ParamValue{
			Name: name,
		})
	}

	return index
}

// ArrayParamValue adds an item to an array parameter, if the parameter is not yet present, it is being added
func (b *buildPrototype) ArrayParamValue(name string, value string) *buildPrototype {
	index := b.determineParameterIndex(name)
	b.build.Spec.ParamValues[index].Values = append(b.build.Spec.ParamValues[index].Values, buildv1beta1.SingleValue{
		Value: &value,
	})

	return b
}

// ArrayParamValueFromConfigMap adds an item to an array parameter, if the parameter is not yet present, it is being added
func (b *buildPrototype) ArrayParamValueFromConfigMap(name string, configMapName string, configMapKey string, format *string) *buildPrototype {
	index := b.determineParameterIndex(name)
	b.build.Spec.ParamValues[index].Values = append(b.build.Spec.ParamValues[index].Values, buildv1beta1.SingleValue{
		ConfigMapValue: &buildv1beta1.ObjectKeyRef{
			Name:   configMapName,
			Key:    configMapKey,
			Format: format,
		},
	})

	return b
}

// ArrayParamValueFromSecret adds an item to an array parameter, if the parameter is not yet present, it is being added
func (b *buildPrototype) ArrayParamValueFromSecret(name string, secretName string, secretKey string, format *string) *buildPrototype {
	index := b.determineParameterIndex(name)
	b.build.Spec.ParamValues[index].Values = append(b.build.Spec.ParamValues[index].Values, buildv1beta1.SingleValue{
		SecretValue: &buildv1beta1.ObjectKeyRef{
			Name:   secretName,
			Key:    secretKey,
			Format: format,
		},
	})

	return b
}

func (b *buildPrototype) StringParamValue(name string, value string) *buildPrototype {
	b.build.Spec.ParamValues = append(b.build.Spec.ParamValues, buildv1beta1.ParamValue{
		Name: name,
		SingleValue: &buildv1beta1.SingleValue{
			Value: &value,
		},
	})

	return b
}

func (b *buildPrototype) StringParamValueFromConfigMap(name string, configMapName string, configMapKey string, format *string) *buildPrototype {
	b.build.Spec.ParamValues = append(b.build.Spec.ParamValues, buildv1beta1.ParamValue{
		Name: name,
		SingleValue: &buildv1beta1.SingleValue{
			ConfigMapValue: &buildv1beta1.ObjectKeyRef{
				Name:   configMapName,
				Key:    configMapKey,
				Format: format,
			},
		},
	})

	return b
}

func (b *buildPrototype) StringParamValueFromSecret(name string, secretName string, secretKey string, format *string) *buildPrototype {
	b.build.Spec.ParamValues = append(b.build.Spec.ParamValues, buildv1beta1.ParamValue{
		Name: name,
		SingleValue: &buildv1beta1.SingleValue{
			SecretValue: &buildv1beta1.ObjectKeyRef{
				Name:   secretName,
				Key:    secretKey,
				Format: format,
			},
		},
	})

	return b
}

func (b *buildPrototype) OutputImageCredentials(name string) *buildPrototype {
	if name != "" {
		b.build.Spec.Output.PushSecret = &name
	}
	return b
}

func (b *buildPrototype) OutputImageInsecure(insecure bool) *buildPrototype {
	b.build.Spec.Output.Insecure = &insecure

	return b
}

func (b buildPrototype) Create() (build *buildv1beta1.Build, err error) {
	ctx := context.Background()

	_, err = testBuild.
		BuildClientSet.
		ShipwrightV1beta1().
		Builds(b.build.Namespace).
		Create(ctx, &b.build, meta.CreateOptions{})

	if err != nil {
		return nil, err
	}

	err = wait.PollImmediate(pollCreateInterval, pollCreateTimeout, func() (done bool, err error) {
		build, err = testBuild.BuildClientSet.ShipwrightV1beta1().Builds(b.build.Namespace).Get(ctx, b.build.Name, meta.GetOptions{})
		if err != nil {
			return false, err
		}

		return build.Status.Registered != nil && *build.Status.Registered == core.ConditionTrue, nil
	})

	return
}

// BuildSpec returns the BuildSpec of this Build (no cluster resource is created)
func (b buildPrototype) BuildSpec() (build *buildv1beta1.BuildSpec) {
	return &b.build.Spec
}

func NewBuildRunPrototype() *buildRunPrototype {
	return &buildRunPrototype{buildRun: buildv1beta1.BuildRun{}}
}

func (b *buildRunPrototype) Name(name string) *buildRunPrototype {
	b.buildRun.ObjectMeta.Name = name
	return b
}

func (b *buildRunPrototype) Namespace(namespace string) *buildRunPrototype {
	b.buildRun.Namespace = namespace
	return b
}

func (b *buildRunPrototype) ForBuild(build *buildv1beta1.Build) *buildRunPrototype {
	b.buildRun.Spec.Build.Name = &build.Name
	b.buildRun.ObjectMeta.Namespace = build.Namespace
	return b
}

func (b *buildRunPrototype) WithBuildSpec(buildSpec *buildv1beta1.BuildSpec) *buildRunPrototype {
	b.buildRun.Spec.Build.Build = buildSpec
	return b
}

func (b *buildRunPrototype) GenerateServiceAccount() *buildRunPrototype {
	generate := ".generate"
	if b.buildRun.Spec.ServiceAccount == nil {
		b.buildRun.Spec.ServiceAccount = &generate
	}
	return b
}

func (b *buildRunPrototype) determineParameterIndex(name string) int {
	index := -1
	for i, paramValue := range b.buildRun.Spec.ParamValues {
		if paramValue.Name == name {
			index = i
			break
		}
	}

	if index == -1 {
		index = len(b.buildRun.Spec.ParamValues)
		b.buildRun.Spec.ParamValues = append(b.buildRun.Spec.ParamValues, buildv1beta1.ParamValue{
			Name: name,
		})
	}

	return index
}

// ArrayParamValue adds an item to an array parameter, if the parameter is not yet present, it is being added
func (b *buildRunPrototype) ArrayParamValue(name string, value string) *buildRunPrototype {
	index := b.determineParameterIndex(name)
	b.buildRun.Spec.ParamValues[index].Values = append(b.buildRun.Spec.ParamValues[index].Values, buildv1beta1.SingleValue{
		Value: &value,
	})

	return b
}

// ArrayParamValueFromConfigMap adds an item to an array parameter, if the parameter is not yet present, it is being added
func (b *buildRunPrototype) ArrayParamValueFromConfigMap(name string, configMapName string, configMapKey string, format *string) *buildRunPrototype {
	index := b.determineParameterIndex(name)
	b.buildRun.Spec.ParamValues[index].Values = append(b.buildRun.Spec.ParamValues[index].Values, buildv1beta1.SingleValue{
		ConfigMapValue: &buildv1beta1.ObjectKeyRef{
			Name:   configMapName,
			Key:    configMapKey,
			Format: format,
		},
	})

	return b
}

// ArrayParamValueFromSecret adds an item to an array parameter, if the parameter is not yet present, it is being added
func (b *buildRunPrototype) ArrayParamValueFromSecret(name string, secretName string, secretKey string, format *string) *buildRunPrototype {
	index := b.determineParameterIndex(name)
	b.buildRun.Spec.ParamValues[index].Values = append(b.buildRun.Spec.ParamValues[index].Values, buildv1beta1.SingleValue{
		SecretValue: &buildv1beta1.ObjectKeyRef{
			Name:   secretName,
			Key:    secretKey,
			Format: format,
		},
	})

	return b
}

func (b *buildRunPrototype) StringParamValue(name string, value string) *buildRunPrototype {
	b.buildRun.Spec.ParamValues = append(b.buildRun.Spec.ParamValues, buildv1beta1.ParamValue{
		Name: name,
		SingleValue: &buildv1beta1.SingleValue{
			Value: &value,
		},
	})

	return b
}

func (b *buildRunPrototype) StringParamValueFromConfigMap(name string, configMapName string, configMapKey string, format *string) *buildRunPrototype {
	b.buildRun.Spec.ParamValues = append(b.buildRun.Spec.ParamValues, buildv1beta1.ParamValue{
		Name: name,
		SingleValue: &buildv1beta1.SingleValue{
			ConfigMapValue: &buildv1beta1.ObjectKeyRef{
				Name:   configMapName,
				Key:    configMapKey,
				Format: format,
			},
		},
	})

	return b
}

func (b *buildRunPrototype) StringParamValueFromSecret(name string, secretName string, secretKey string, format *string) *buildRunPrototype {
	b.buildRun.Spec.ParamValues = append(b.buildRun.Spec.ParamValues, buildv1beta1.ParamValue{
		Name: name,
		SingleValue: &buildv1beta1.SingleValue{
			SecretValue: &buildv1beta1.ObjectKeyRef{
				Name:   secretName,
				Key:    secretKey,
				Format: format,
			},
		},
	})

	return b
}

func (b *buildRunPrototype) Create() (*buildv1beta1.BuildRun, error) {
	return testBuild.
		BuildClientSet.
		ShipwrightV1beta1().
		BuildRuns(b.buildRun.Namespace).
		Create(context.Background(), &b.buildRun, meta.CreateOptions{})
}

// Logf logs data
func Logf(format string, args ...interface{}) {
	currentTime := time.Now().UTC().Format(time.RFC3339)

	fmt.Fprintf(
		GinkgoWriter,
		fmt.Sprintf("%s %d %s\n", currentTime, getGinkgoNode(), format),
		args...,
	)
}

func getArg(argName string) (bool, string) {
	for i, arg := range os.Args {
		if arg == argName {
			return true, os.Args[i+1]
		} else if strings.HasPrefix(arg, argName+"=") {
			argAndValue := strings.SplitN(arg, "=", 2)
			return true, argAndValue[1]
		}
	}
	return false, ""
}

func getGinkgoNode() int {
	defined, ginkgoNodeString := getArg("--ginkgo.parallel.node")
	if !defined {
		return 1
	}
	ginkgoNode, err := strconv.Atoi(ginkgoNodeString)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		return 0
	}
	return ginkgoNode
}
