// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Create(ctx, &c.clusterBuildStrategy, metav1.CreateOptions{})

	if err != nil {
		return nil, err
	}

	err = wait.PollUntilContextTimeout(ctx, pollCreateInterval, pollCreateTimeout, true, func(context.Context) (done bool, err error) {
		cbs, err = testBuild.BuildClientSet.ShipwrightV1beta1().ClusterBuildStrategies().Get(ctx, c.clusterBuildStrategy.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return true, nil
	})

	return
}

func (c *clusterBuildStrategyPrototype) TestMe(f func(clusterBuildStrategy *buildv1beta1.ClusterBuildStrategy)) {
	GinkgoHelper()
	cbs, err := c.Create()
	Expect(err).ToNot(HaveOccurred())

	f(cbs)

	Expect(testBuild.
		BuildClientSet.
		ShipwrightV1beta1().
		ClusterBuildStrategies().
		Delete(context.Background(), cbs.Name, metav1.DeleteOptions{}),
	).To(Succeed())
}
