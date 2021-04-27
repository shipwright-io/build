// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
)

var _ = Describe("runtime-image", func() {
	b := &buildv1alpha1.Build{
		Spec: buildv1alpha1.BuildSpec{
			Builder: &buildv1alpha1.Image{
				Image: "test/builder-image:latest",
			},
			Output: buildv1alpha1.Image{
				Image: "test/output-image:latest",
			},
			Runtime: &buildv1alpha1.Runtime{
				Base: buildv1alpha1.Image{
					Image: "test/base-image:latest",
				},
				Env: map[string]string{
					"ENVIRONMENT_VARIABLE": "VALUE",
				},
				Labels: map[string]string{
					"label": "value",
				},
				WorkDir: "/workdir",
				Run:     []string{"command --args"},
				User: &buildv1alpha1.User{
					Name:  "username",
					Group: "1001",
				},
				Paths:      []string{"/path/to/a:/new/path/to/a", "/path/to/b"},
				Entrypoint: []string{"/bin/bash", "-x", "-c"},
			},
		},
	}

	Context("rendering user and group", func() {
		It("expect empty when user is not informed", func() {
			u := renderUserAndGroup(&buildv1alpha1.User{})
			Expect(u).To(BeEmpty())

			// when only group is informed, it also expects empty string
			u = renderUserAndGroup(&buildv1alpha1.User{Group: "group"})
			Expect(u).To(BeEmpty())
		})

		It("expect user and group joined by colon", func() {
			u := renderUserAndGroup(b.Spec.Runtime.User)
			Expect(u).To(Equal("username:1001"))

			u = renderUserAndGroup(&buildv1alpha1.User{Name: "username"})
			Expect(u).To(Equal("username"))
		})
	})

	Context("splitting paths", func() {
		It("expect paths splitted by \":\" ", func() {
			parts := splitPaths("a:b")
			Expect(parts).To(Equal([]string{"a", "b"}))

			parts = splitPaths("a")
			Expect(parts).To(Equal([]string{"a", "a"}))
		})
	})

	Context("rendering entrypoint", func() {
		It("expect entrypoint concatenated", func() {
			entrypoint := renderEntrypoint(b.Spec.Runtime.Entrypoint)

			Expect(entrypoint).To(Equal("\"/bin/bash\", \"-x\", \"-c\""))
		})
	})

	Context("rendering runtime Dockerfile", func() {

		It("expect a complete dockerfile", func() {
			dockerfile, err := renderRuntimeDockerfile(b)

			Expect(err).ToNot(HaveOccurred())
			Expect(dockerfile).ToNot(BeNil())

			Expect(fmt.Sprintf("\n%s", dockerfile)).To(Equal(`
FROM $(params.shp-output-image) as builder

FROM test/base-image:latest
ENV ENVIRONMENT_VARIABLE="VALUE"
LABEL label="value"
RUN command --args
COPY --chown="username:1001" --from=builder "/path/to/a" "/new/path/to/a"
COPY --chown="username:1001" --from=builder "/path/to/b" "/path/to/b"
WORKDIR "/workdir"
USER username:1001
ENTRYPOINT [ "/bin/bash", "-x", "-c" ]`,
			))
		})
	})

	Context("amend build-strategy with extra steps", func() {
		taskSpec := &v1beta1.TaskSpec{
			Steps: []v1beta1.Step{},
		}

		It("expect to have Tekton's Task amended", func() {
			err := AmendTaskSpecWithRuntimeImage(config.NewDefaultConfig(), taskSpec, b)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(taskSpec.Steps)).To(Equal(2))
		})
	})
})
