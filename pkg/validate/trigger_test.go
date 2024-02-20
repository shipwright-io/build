// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package validate_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("ValidateBuildTriggers", func() {
	Context("trigger name is not informed", func() {
		b := &build.Build{
			Spec: build.BuildSpec{
				Trigger: &build.Trigger{
					When: []build.TriggerWhen{{
						Name: "",
					}},
				},
			},
		}

		It("should error when name is not set", func() {
			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err.Error()).To(ContainSubstring("name is not set"))
		})
	})

	Context("trigger type github", func() {
		It("should error when github attribute is not set", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "github",
							Type: build.GitHubWebHookTrigger,
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err.Error()).To(ContainSubstring("missing required attribute `.github`"))
		})

		It("should error when github events attribute is empty", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "github",
							Type: build.GitHubWebHookTrigger,
							GitHub: &build.WhenGitHub{
								Events: []build.GitHubEventName{},
							},
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err.Error()).To(ContainSubstring("missing required attribute `.github.events`"))
		})

		It("should pass when github type is complete", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "github",
							Type: build.GitHubWebHookTrigger,
							GitHub: &build.WhenGitHub{
								Events: []build.GitHubEventName{
									build.GitHubPushEvent,
								},
							},
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("trigger type image", func() {
		It("should error when image attribute is not set", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "image",
							Type: build.ImageTrigger,
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err.Error()).To(ContainSubstring("missing required attribute `.image`"))
		})

		It("should error when image names attribute is empty", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "image",
							Type: build.ImageTrigger,
							Image: &build.WhenImage{
								Names: []string{},
							},
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err.Error()).To(ContainSubstring("missing required attribute `.image.names`"))
		})

		It("should pass when github type is complete", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "image",
							Type: build.ImageTrigger,
							Image: &build.WhenImage{
								Names: []string{
									"ghcr.io/shipwright-io/build:latest",
								},
							},
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("trigger type pipeline", func() {
		It("should error when objectRef attribute is not set", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "pipeline",
							Type: build.PipelineTrigger,
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err.Error()).To(ContainSubstring("missing required attribute `.objectRef`"))
		})

		It("should error when status attribute is empty", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "pipeline",
							Type: build.PipelineTrigger,
							ObjectRef: &build.WhenObjectRef{
								Status: []string{},
							},
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err.Error()).To(ContainSubstring("missing required attribute `.objectRef.status`"))
		})

		It("should error when missing required attributes", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "pipeline",
							Type: build.PipelineTrigger,
							ObjectRef: &build.WhenObjectRef{
								Status: []string{"Succeed"},
							},
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err.Error()).To(ContainSubstring(
				"is missing required attributes `.objectRef.name` or `.objectRef.selector`",
			))
		})

		It("should error when declaring conflicting attributes", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "pipeline",
							Type: build.PipelineTrigger,
							ObjectRef: &build.WhenObjectRef{
								Status: []string{"Succeed"},
								Name:   "name",
								Selector: map[string]string{
									"k": "v",
								},
							},
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err.Error()).To(ContainSubstring(
				"contains `.objectRef.name` and `.objectRef.selector`, must be only one",
			))
		})

		It("should pass when objectRef type is complete", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "pipeline",
							Type: build.PipelineTrigger,
							ObjectRef: &build.WhenObjectRef{
								Status: []string{"Succeed"},
								Name:   "name",
							},
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("invalid trigger type", func() {
		It("should error when declaring a invalid trigger type", func() {
			b := &build.Build{
				Spec: build.BuildSpec{
					Trigger: &build.Trigger{
						When: []build.TriggerWhen{{
							Name: "pipeline",
							Type: build.TriggerType("invalid"),
							ObjectRef: &build.WhenObjectRef{
								Name: "name",
							},
						}},
					},
				},
			}

			err := validate.NewTrigger(b).ValidatePath(context.TODO())
			Expect(err.Error()).To(ContainSubstring("contains an invalid type"))
		})
	})
})
