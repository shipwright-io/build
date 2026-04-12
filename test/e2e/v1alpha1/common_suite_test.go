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
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

const (
	pollCreateInterval = 1 * time.Second
	pollCreateTimeout  = 10 * time.Second
)

type buildPrototype struct{ build buildapialpha.Build }
type buildRunPrototype struct{ buildRun buildapialpha.BuildRun }

func NewBuildPrototype() *buildPrototype {
	return &buildPrototype{
		build: buildapialpha.Build{},
	}
}

func (b *buildPrototype) Name(name string) *buildPrototype {
	b.build.Name = name
	return b
}

func (b *buildPrototype) Namespace(namespace string) *buildPrototype {
	b.build.Namespace = namespace
	return b
}

func (b *buildPrototype) BuildStrategy(name string) *buildPrototype {
	var bs = buildapialpha.NamespacedBuildStrategyKind
	b.build.Spec.Strategy = buildapialpha.Strategy{
		Kind: &bs,
		Name: name,
	}
	return b
}

func (b *buildPrototype) ClusterBuildStrategy(name string) *buildPrototype {
	var cbs = buildapialpha.ClusterBuildStrategyKind
	b.build.Spec.Strategy = buildapialpha.Strategy{
		Kind: &cbs,
		Name: name,
	}
	return b
}

func (b *buildPrototype) SourceCredentials(name string) *buildPrototype {
	if name != "" {
		b.build.Spec.Source.Credentials = &core.LocalObjectReference{Name: name}
	}

	return b
}

func (b *buildPrototype) SourceGit(repository string) *buildPrototype {
	b.build.Spec.Source.URL = ptr.To(repository)
	b.build.Spec.Source.BundleContainer = nil
	return b
}

func (b *buildPrototype) SourceGitRevision(revision string) *buildPrototype {
	b.build.Spec.Source.Revision = ptr.To(revision)
	return b
}

func (b *buildPrototype) SourceBundle(image string) *buildPrototype {
	if b.build.Spec.Source.BundleContainer == nil {
		b.build.Spec.Source.BundleContainer = &buildapialpha.BundleContainer{}
	}
	b.build.Spec.Source.BundleContainer.Image = image
	return b
}

func (b *buildPrototype) SourceBundlePrune(prune buildapialpha.PruneOption) *buildPrototype {
	if b.build.Spec.Source.BundleContainer == nil {
		b.build.Spec.Source.BundleContainer = &buildapialpha.BundleContainer{}
	}
	b.build.Spec.Source.BundleContainer.Prune = &prune
	return b
}

func (b *buildPrototype) SourceContextDir(contextDir string) *buildPrototype {
	b.build.Spec.Source.ContextDir = ptr.To(contextDir)
	return b
}

func (b *buildPrototype) Dockerfile(dockerfile string) *buildPrototype {
	b.build.Spec.Dockerfile = &dockerfile
	return b
}

func (b *buildPrototype) Env(key string, value string) *buildPrototype {
	b.build.Spec.Env = append(b.build.Spec.Env, core.EnvVar{
		Name:  key,
		Value: value,
	})
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
		b.build.Spec.ParamValues = append(b.build.Spec.ParamValues, buildapialpha.ParamValue{
			Name: name,
		})
	}

	return index
}

// ArrayParamValue adds an item to an array parameter, if the parameter is not yet present, it is being added
func (b *buildPrototype) ArrayParamValue(name string, value string) *buildPrototype {
	index := b.determineParameterIndex(name)
	b.build.Spec.ParamValues[index].Values = append(b.build.Spec.ParamValues[index].Values, buildapialpha.SingleValue{
		Value: &value,
	})

	return b
}

// ArrayParamValueFromConfigMap adds an item to an array parameter, if the parameter is not yet present, it is being added
func (b *buildPrototype) ArrayParamValueFromConfigMap(name string, configMapName string, configMapKey string, format *string) *buildPrototype {
	index := b.determineParameterIndex(name)
	b.build.Spec.ParamValues[index].Values = append(b.build.Spec.ParamValues[index].Values, buildapialpha.SingleValue{
		ConfigMapValue: &buildapialpha.ObjectKeyRef{
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
	b.build.Spec.ParamValues[index].Values = append(b.build.Spec.ParamValues[index].Values, buildapialpha.SingleValue{
		SecretValue: &buildapialpha.ObjectKeyRef{
			Name:   secretName,
			Key:    secretKey,
			Format: format,
		},
	})

	return b
}

func (b *buildPrototype) StringParamValue(name string, value string) *buildPrototype {
	b.build.Spec.ParamValues = append(b.build.Spec.ParamValues, buildapialpha.ParamValue{
		Name: name,
		SingleValue: &buildapialpha.SingleValue{
			Value: &value,
		},
	})

	return b
}

func (b *buildPrototype) StringParamValueFromConfigMap(name string, configMapName string, configMapKey string, format *string) *buildPrototype {
	b.build.Spec.ParamValues = append(b.build.Spec.ParamValues, buildapialpha.ParamValue{
		Name: name,
		SingleValue: &buildapialpha.SingleValue{
			ConfigMapValue: &buildapialpha.ObjectKeyRef{
				Name:   configMapName,
				Key:    configMapKey,
				Format: format,
			},
		},
	})

	return b
}

func (b *buildPrototype) StringParamValueFromSecret(name string, secretName string, secretKey string, format *string) *buildPrototype {
	b.build.Spec.ParamValues = append(b.build.Spec.ParamValues, buildapialpha.ParamValue{
		Name: name,
		SingleValue: &buildapialpha.SingleValue{
			SecretValue: &buildapialpha.ObjectKeyRef{
				Name:   secretName,
				Key:    secretKey,
				Format: format,
			},
		},
	})

	return b
}

func (b *buildPrototype) OutputImage(image string) *buildPrototype {
	b.build.Spec.Output.Image = image
	return b
}

func (b *buildPrototype) OutputImageCredentials(name string) *buildPrototype {
	if name != "" {
		b.build.Spec.Output.Credentials = &core.LocalObjectReference{Name: name}
	}

	return b
}

func (b *buildPrototype) OutputImageInsecure(insecure bool) *buildPrototype {
	b.build.Spec.Output.Insecure = &insecure

	return b
}

func (b *buildPrototype) OutputTimestamp(timestampString string) *buildPrototype {
	b.build.Spec.Output.Timestamp = &timestampString
	return b
}

func (b buildPrototype) Create() (build *buildapialpha.Build, err error) {
	ctx := context.Background()

	_, err = testBuild.
		BuildClientSet.
		ShipwrightV1alpha1().
		Builds(b.build.Namespace).
		Create(ctx, &b.build, meta.CreateOptions{})

	if err != nil {
		return nil, err
	}

	err = wait.PollUntilContextTimeout(ctx, pollCreateInterval, pollCreateTimeout, true, func(ctx context.Context) (done bool, err error) {
		build, err = testBuild.BuildClientSet.ShipwrightV1alpha1().Builds(b.build.Namespace).Get(ctx, b.build.Name, meta.GetOptions{})
		if err != nil {
			return false, err
		}

		return build.Status.Registered != nil && *build.Status.Registered == core.ConditionTrue, nil
	})

	return
}

// BuildSpec returns the BuildSpec of this Build (no cluster resource is created)
func (b buildPrototype) BuildSpec() (build *buildapialpha.BuildSpec) {
	return &b.build.Spec
}

func NewBuildRunPrototype() *buildRunPrototype {
	return &buildRunPrototype{buildRun: buildapialpha.BuildRun{}}
}

func (b *buildRunPrototype) Name(name string) *buildRunPrototype {
	b.buildRun.Name = name
	return b
}

func (b *buildRunPrototype) Namespace(namespace string) *buildRunPrototype {
	b.buildRun.Namespace = namespace
	return b
}

func (b *buildRunPrototype) ForBuild(build *buildapialpha.Build) *buildRunPrototype {
	b.buildRun.Spec.BuildRef = &buildapialpha.BuildRef{Name: build.Name}
	b.buildRun.Namespace = build.Namespace
	return b
}

func (b *buildRunPrototype) WithBuildSpec(buildSpec *buildapialpha.BuildSpec) *buildRunPrototype {
	b.buildRun.Spec.BuildSpec = buildSpec
	return b
}

func (b *buildRunPrototype) GenerateServiceAccount() *buildRunPrototype {
	if b.buildRun.Spec.ServiceAccount == nil {
		b.buildRun.Spec.ServiceAccount = &buildapialpha.ServiceAccount{}
	}
	b.buildRun.Spec.ServiceAccount.Generate = ptr.To(true)
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
		b.buildRun.Spec.ParamValues = append(b.buildRun.Spec.ParamValues, buildapialpha.ParamValue{
			Name: name,
		})
	}

	return index
}

// ArrayParamValue adds an item to an array parameter, if the parameter is not yet present, it is being added
func (b *buildRunPrototype) ArrayParamValue(name string, value string) *buildRunPrototype {
	index := b.determineParameterIndex(name)
	b.buildRun.Spec.ParamValues[index].Values = append(b.buildRun.Spec.ParamValues[index].Values, buildapialpha.SingleValue{
		Value: &value,
	})

	return b
}

// ArrayParamValueFromConfigMap adds an item to an array parameter, if the parameter is not yet present, it is being added
func (b *buildRunPrototype) ArrayParamValueFromConfigMap(name string, configMapName string, configMapKey string, format *string) *buildRunPrototype {
	index := b.determineParameterIndex(name)
	b.buildRun.Spec.ParamValues[index].Values = append(b.buildRun.Spec.ParamValues[index].Values, buildapialpha.SingleValue{
		ConfigMapValue: &buildapialpha.ObjectKeyRef{
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
	b.buildRun.Spec.ParamValues[index].Values = append(b.buildRun.Spec.ParamValues[index].Values, buildapialpha.SingleValue{
		SecretValue: &buildapialpha.ObjectKeyRef{
			Name:   secretName,
			Key:    secretKey,
			Format: format,
		},
	})

	return b
}

func (b *buildRunPrototype) StringParamValue(name string, value string) *buildRunPrototype {
	b.buildRun.Spec.ParamValues = append(b.buildRun.Spec.ParamValues, buildapialpha.ParamValue{
		Name: name,
		SingleValue: &buildapialpha.SingleValue{
			Value: &value,
		},
	})

	return b
}

func (b *buildRunPrototype) StringParamValueFromConfigMap(name string, configMapName string, configMapKey string, format *string) *buildRunPrototype {
	b.buildRun.Spec.ParamValues = append(b.buildRun.Spec.ParamValues, buildapialpha.ParamValue{
		Name: name,
		SingleValue: &buildapialpha.SingleValue{
			ConfigMapValue: &buildapialpha.ObjectKeyRef{
				Name:   configMapName,
				Key:    configMapKey,
				Format: format,
			},
		},
	})

	return b
}

func (b *buildRunPrototype) StringParamValueFromSecret(name string, secretName string, secretKey string, format *string) *buildRunPrototype {
	b.buildRun.Spec.ParamValues = append(b.buildRun.Spec.ParamValues, buildapialpha.ParamValue{
		Name: name,
		SingleValue: &buildapialpha.SingleValue{
			SecretValue: &buildapialpha.ObjectKeyRef{
				Name:   secretName,
				Key:    secretKey,
				Format: format,
			},
		},
	})

	return b
}

func (b *buildRunPrototype) Create() (*buildapialpha.BuildRun, error) {
	return testBuild.
		BuildClientSet.
		ShipwrightV1alpha1().
		BuildRuns(b.buildRun.Namespace).
		Create(context.Background(), &b.buildRun, meta.CreateOptions{})
}

func (b *buildRunPrototype) MustCreate() *buildapialpha.BuildRun {
	GinkgoHelper()

	buildrun, err := b.Create()
	Expect(err).ToNot(HaveOccurred())
	Expect(buildrun).ToNot(BeNil())

	return buildrun
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
