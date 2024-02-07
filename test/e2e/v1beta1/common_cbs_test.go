// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"

	. "github.com/onsi/gomega"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

type clusterBuildStrategyPrototype struct {
	clusterBuildStrategy buildv1beta1.ClusterBuildStrategy
}

func NewClusterBuildStrategyPrototype() *clusterBuildStrategyPrototype {
	return &clusterBuildStrategyPrototype{
		clusterBuildStrategy: buildv1beta1.ClusterBuildStrategy{},
	}
}

func (c *clusterBuildStrategyPrototype) Name(name string) *clusterBuildStrategyPrototype {
	c.clusterBuildStrategy.ObjectMeta.Name = name
	return c
}

func (c *clusterBuildStrategyPrototype) BuildStep(buildStep buildv1beta1.Step) *clusterBuildStrategyPrototype {
	c.clusterBuildStrategy.Spec.Steps = append(c.clusterBuildStrategy.Spec.Steps, buildStep)
	return c
}

func (c *clusterBuildStrategyPrototype) Parameter(parameter buildv1beta1.Parameter) *clusterBuildStrategyPrototype {
	c.clusterBuildStrategy.Spec.Parameters = append(c.clusterBuildStrategy.Spec.Parameters, parameter)
	return c
}

func (c *clusterBuildStrategyPrototype) Volume(volume buildv1beta1.BuildStrategyVolume) *clusterBuildStrategyPrototype {
	c.clusterBuildStrategy.Spec.Volumes = append(c.clusterBuildStrategy.Spec.Volumes, volume)
	return c
}

func (c *clusterBuildStrategyPrototype) Create() (cbs *buildv1beta1.ClusterBuildStrategy, err error) {
	ctx := context.Background()

	_, err = testBuild.
		BuildClientSet.
		ShipwrightV1beta1().
		ClusterBuildStrategies().
		Create(ctx, &c.clusterBuildStrategy, meta.CreateOptions{})

	if err != nil {
		return nil, err
	}

	err = wait.PollImmediate(pollCreateInterval, pollCreateTimeout, func() (done bool, err error) {
		cbs, err = testBuild.BuildClientSet.ShipwrightV1beta1().ClusterBuildStrategies().Get(ctx, c.clusterBuildStrategy.Name, meta.GetOptions{})
		if err != nil {
			return false, err
		}

		return true, nil
	})

	return
}

func (c *clusterBuildStrategyPrototype) TestMe(f func(clusterBuildStrategy *buildv1beta1.ClusterBuildStrategy)) {
	cbs, err := c.Create()
	Expect(err).ToNot(HaveOccurred())

	f(cbs)

	Expect(testBuild.
		BuildClientSet.
		ShipwrightV1beta1().
		ClusterBuildStrategies().
		Delete(context.Background(), cbs.Name, meta.DeleteOptions{}),
	).To(Succeed())
}
