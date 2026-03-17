// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
)

var _ = Describe("Multi-Arch PipelineRun Generation", func() {
	var (
		cfg                  *config.Config
		build                *buildv1beta1.Build
		buildRun             *buildv1beta1.BuildRun
		clusterBuildStrategy *buildv1beta1.ClusterBuildStrategy
		serviceAccountName   string
		ctl                  test.Catalog
	)

	BeforeEach(func() {
		cfg = config.NewDefaultConfig()
		serviceAccountName = "test-sa"

		var err error
		build, err = ctl.LoadBuildYAML([]byte(test.MinimalBuild))
		Expect(err).ToNot(HaveOccurred())

		buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.MinimalBuildRun))
		Expect(err).ToNot(HaveOccurred())

		clusterBuildStrategy, err = ctl.LoadCBSWithName("noop", []byte(test.ClusterBuildStrategyNoOp))
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when multiArch is configured on the Build", func() {
		BeforeEach(func() {
			build.Spec.Output.MultiArch = &buildv1beta1.MultiArch{
				Platforms: []buildv1beta1.ImagePlatform{
					{OS: "linux", Arch: "amd64"},
					{OS: "linux", Arch: "arm64"},
				},
			}
		})

		It("generates a PipelineRun with per-platform build tasks", func() {
			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())
			Expect(pr).ToNot(BeNil())

			tasks := pr.Spec.PipelineSpec.Tasks
			taskNames := make([]string, len(tasks))
			for i, t := range tasks {
				taskNames[i] = t.Name
			}

			Expect(taskNames).To(ContainElement("source-acquisition"))
			Expect(taskNames).To(ContainElement("build-linux-amd64"))
			Expect(taskNames).To(ContainElement("build-linux-arm64"))
			Expect(taskNames).To(ContainElement("assemble-index"))
		})

		It("sets per-platform build tasks to run after source-acquisition", func() {
			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			for _, task := range pr.Spec.PipelineSpec.Tasks {
				if task.Name == "build-linux-amd64" || task.Name == "build-linux-arm64" {
					Expect(task.RunAfter).To(ContainElement("source-acquisition"))
				}
			}
		})

		It("sets assemble-index to run after all build tasks", func() {
			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			for _, task := range pr.Spec.PipelineSpec.Tasks {
				if task.Name == "assemble-index" {
					Expect(task.RunAfter).To(ContainElements("build-linux-amd64", "build-linux-arm64"))
				}
			}
		})

		It("uses EmptyDir workspace bindings for multi-arch", func() {
			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			for _, ws := range pr.Spec.Workspaces {
				Expect(ws.EmptyDir).ToNot(BeNil(), "workspace %s should use EmptyDir", ws.Name)
				Expect(ws.VolumeClaimTemplate).To(BeNil(), "workspace %s should not use PVC", ws.Name)
			}
		})

		It("generates TaskRunSpecs with per-platform nodeSelector", func() {
			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			Expect(pr.Spec.TaskRunSpecs).To(HaveLen(2))

			specMap := make(map[string]interface{})
			for _, s := range pr.Spec.TaskRunSpecs {
				specMap[s.PipelineTaskName] = s
			}

			Expect(specMap).To(HaveKey("build-linux-amd64"))
			Expect(specMap).To(HaveKey("build-linux-arm64"))

			for _, s := range pr.Spec.TaskRunSpecs {
				Expect(s.PodTemplate).ToNot(BeNil())
				Expect(s.PodTemplate.NodeSelector).To(HaveKey("kubernetes.io/os"))
				Expect(s.PodTemplate.NodeSelector).To(HaveKey("kubernetes.io/arch"))
			}
		})

		It("includes a source bundle push step in source-acquisition", func() {
			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			var found bool
			for _, t := range pr.Spec.PipelineSpec.Tasks {
				if t.Name == "source-acquisition" {
					for _, s := range t.TaskSpec.Steps {
						if s.Name == "push-source-bundle" {
							found = true
						}
					}
				}
			}
			Expect(found).To(BeTrue(), "source-acquisition should have push-source-bundle step")
		})

		It("overrides shp-output-image for per-platform tasks", func() {
			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			for _, task := range pr.Spec.PipelineSpec.Tasks {
				if task.Name == "build-linux-amd64" {
					for _, p := range task.Params {
						if p.Name == "shp-output-image" {
							Expect(p.Value.StringVal).To(ContainSubstring("-linux-amd64"))
						}
					}
				}
				if task.Name == "build-linux-arm64" {
					for _, p := range task.Params {
						if p.Name == "shp-output-image" {
							Expect(p.Value.StringVal).To(ContainSubstring("-linux-arm64"))
						}
					}
				}
			}
		})

		It("includes an image-processing step in each per-platform task", func() {
			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			for _, task := range pr.Spec.PipelineSpec.Tasks {
				if task.Name == "build-linux-amd64" || task.Name == "build-linux-arm64" {
					var hasImageProcessing bool
					for _, s := range task.TaskSpec.Steps {
						if s.Name == "image-processing" {
							hasImageProcessing = true
							Expect(s.Args).To(ContainElement("--result-file-image-digest"))
							Expect(s.Args).To(ContainElement("--result-file-image-size"))
						}
					}
					Expect(hasImageProcessing).To(BeTrue(),
						"per-platform task %s must have an image-processing step", task.Name)
				}
			}
		})

		It("mounts push secret on source-bundle push, per-platform pull, and assemble-index when PushSecret is set", func() {
			secretName := "registry-creds"
			build.Spec.Output.PushSecret = &secretName

			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			for _, task := range pr.Spec.PipelineSpec.Tasks {
				switch task.Name {
				case "source-acquisition":
					for _, step := range task.TaskSpec.Steps {
						if step.Name == "push-source-bundle" {
							Expect(step.Args).To(ContainElement("--secret-path"),
								"push-source-bundle should have --secret-path arg")
							var hasMount bool
							for _, vm := range step.VolumeMounts {
								if vm.Name == "shp-registry-creds" && vm.ReadOnly {
									hasMount = true
								}
							}
							Expect(hasMount).To(BeTrue(),
								"push-source-bundle should mount the push secret volume")
						}
					}

				case "build-linux-amd64", "build-linux-arm64":
					for _, step := range task.TaskSpec.Steps {
						if step.Name == "pull-source-bundle" {
							Expect(step.Args).To(ContainElement("--secret-path"),
								"pull-source-bundle in %s should have --secret-path arg", task.Name)
							var hasMount bool
							for _, vm := range step.VolumeMounts {
								if vm.Name == "shp-registry-creds" && vm.ReadOnly {
									hasMount = true
								}
							}
							Expect(hasMount).To(BeTrue(),
								"pull-source-bundle in %s should mount the push secret volume", task.Name)
						}
					}

				case "assemble-index":
					for _, step := range task.TaskSpec.Steps {
						if step.Name == "assemble-index" {
							Expect(step.Args).To(ContainElement("--secret-path"),
								"assemble-index should have --secret-path arg")
							var hasMount bool
							for _, vm := range step.VolumeMounts {
								if vm.Name == "shp-registry-creds" && vm.ReadOnly {
									hasMount = true
								}
							}
							Expect(hasMount).To(BeTrue(),
								"assemble-index step should mount the push secret volume")
						}
					}
				}
			}
		})

		It("sets up home and tmp volumes on all multi-arch custom steps", func() {
			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			checkStepHasHomeTmp := func(taskName, stepName string) {
				var found bool
				for _, task := range pr.Spec.PipelineSpec.Tasks {
					if task.Name == taskName {
						for _, step := range task.TaskSpec.Steps {
							if step.Name == stepName {
								found = true
								var hasHome, hasTmpDir bool
								for _, e := range step.Env {
									if e.Name == "HOME" && e.Value == "/shp-writable-home" {
										hasHome = true
									}
									if e.Name == "TMPDIR" && e.Value == "/shp-tmp" {
										hasTmpDir = true
									}
								}
								Expect(hasHome).To(BeTrue(),
									"%s/%s should have HOME=/shp-writable-home", taskName, stepName)
								Expect(hasTmpDir).To(BeTrue(),
									"%s/%s should have TMPDIR=/shp-tmp", taskName, stepName)

								var hasHomeMnt, hasTmpMnt bool
								for _, vm := range step.VolumeMounts {
									if vm.MountPath == "/shp-writable-home" {
										hasHomeMnt = true
									}
									if vm.MountPath == "/shp-tmp" {
										hasTmpMnt = true
									}
								}
								Expect(hasHomeMnt).To(BeTrue(),
									"%s/%s should have volume mount at /shp-writable-home", taskName, stepName)
								Expect(hasTmpMnt).To(BeTrue(),
									"%s/%s should have volume mount at /shp-tmp", taskName, stepName)
							}
						}
					}
				}
				Expect(found).To(BeTrue(), "step %s not found in task %s", stepName, taskName)
			}

			checkStepHasHomeTmp("source-acquisition", "push-source-bundle")
			checkStepHasHomeTmp("build-linux-amd64", "pull-source-bundle")
			checkStepHasHomeTmp("build-linux-arm64", "pull-source-bundle")
			checkStepHasHomeTmp("assemble-index", "assemble-index")
		})

		It("propagates tolerations to per-platform TaskRunSpecs", func() {
			build.Spec.Tolerations = []corev1.Toleration{
				{Key: "dedicated", Value: "build", Effect: corev1.TaintEffectNoSchedule},
			}

			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			Expect(pr.Spec.TaskRunSpecs).To(HaveLen(2))
			for _, spec := range pr.Spec.TaskRunSpecs {
				Expect(spec.PodTemplate).ToNot(BeNil())
				Expect(spec.PodTemplate.Tolerations).To(HaveLen(1))
				Expect(spec.PodTemplate.Tolerations[0].Key).To(Equal("dedicated"))
				Expect(spec.PodTemplate.Tolerations[0].Value).To(Equal("build"))
				Expect(spec.PodTemplate.Tolerations[0].Effect).To(Equal(corev1.TaintEffectNoSchedule))
			}
		})

		Context("when the strategy defines a SecurityContext", func() {
			BeforeEach(func() {
				clusterBuildStrategy.Spec.SecurityContext = &buildv1beta1.BuildStrategySecurityContext{
					RunAsUser:  1000,
					RunAsGroup: 1000,
				}
			})

			It("applies security context volumes to per-platform and assemble-index tasks", func() {
				pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
				Expect(err).ToNot(HaveOccurred())

				for _, task := range pr.Spec.PipelineSpec.Tasks {
					switch task.Name {
					case "build-linux-amd64", "build-linux-arm64":
						var hasSecCtxVolume bool
						for _, v := range task.TaskSpec.Volumes {
							if v.Name == "shp-security-context" {
								hasSecCtxVolume = true
							}
						}
						Expect(hasSecCtxVolume).To(BeTrue(),
							"per-platform task %s should have shp-security-context volume", task.Name)

						for _, step := range task.TaskSpec.Steps {
							if step.Name != "step-no-and-op" {
								var hasPasswd bool
								for _, vm := range step.VolumeMounts {
									if vm.MountPath == "/etc/passwd" && vm.Name == "shp-security-context" {
										hasPasswd = true
									}
								}
								Expect(hasPasswd).To(BeTrue(),
									"non-strategy step %s in %s should have /etc/passwd mount", step.Name, task.Name)
							}
						}

					case "assemble-index":
						var hasSecCtxVolume bool
						for _, v := range task.TaskSpec.Volumes {
							if v.Name == "shp-security-context" {
								hasSecCtxVolume = true
							}
						}
						Expect(hasSecCtxVolume).To(BeTrue(),
							"assemble-index should have shp-security-context volume")

						for _, step := range task.TaskSpec.Steps {
							var hasPasswd bool
							for _, vm := range step.VolumeMounts {
								if vm.MountPath == "/etc/passwd" && vm.Name == "shp-security-context" {
									hasPasswd = true
								}
							}
							Expect(hasPasswd).To(BeTrue(),
								"step %s in assemble-index should have /etc/passwd mount", step.Name)
						}
					}
				}
			})
		})

		It("wires SourceTimestamp from source-acquisition to per-platform tasks via pipeline parameter", func() {
			sourceTimestamp := buildv1beta1.OutputImageSourceTimestamp
			build.Spec.Output.Timestamp = &sourceTimestamp

			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			for _, task := range pr.Spec.PipelineSpec.Tasks {
				if task.Name == "build-linux-amd64" || task.Name == "build-linux-arm64" {
					// TaskSpec must declare the source-timestamp param
					var hasParam bool
					for _, p := range task.TaskSpec.Params {
						if p.Name == "source-timestamp" {
							hasParam = true
						}
					}
					Expect(hasParam).To(BeTrue(),
						"%s TaskSpec should declare source-timestamp param", task.Name)

					// PipelineTask param must reference source-acquisition result
					var paramVal string
					for _, p := range task.Params {
						if p.Name == "source-timestamp" {
							paramVal = p.Value.StringVal
						}
					}
					Expect(paramVal).To(ContainSubstring("tasks.source-acquisition.results.shp-source-default-source-timestamp"),
						"%s should wire source-timestamp from source-acquisition", task.Name)

					// Image-processing step must use --image-timestamp with the param
					for _, step := range task.TaskSpec.Steps {
						if step.Name == "image-processing" {
							Expect(step.Args).To(ContainElement("--image-timestamp"),
								"%s image-processing should have --image-timestamp", task.Name)
							Expect(step.Args).To(ContainElement("$(params.source-timestamp)"),
								"%s image-processing should reference $(params.source-timestamp)", task.Name)
						}
					}
				}
			}
		})

		It("includes vulnerability scanning args in per-platform image-processing when enabled", func() {
			build.Spec.Output.VulnerabilityScan = &buildv1beta1.VulnerabilityScanOptions{
				Enabled: true,
			}

			pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred())

			for _, task := range pr.Spec.PipelineSpec.Tasks {
				if task.Name == "build-linux-amd64" || task.Name == "build-linux-arm64" {
					for _, step := range task.TaskSpec.Steps {
						if step.Name == "image-processing" {
							Expect(step.Args).To(ContainElement("--vuln-settings"),
								"per-platform image-processing in %s should have --vuln-settings", task.Name)
							Expect(step.Args).To(ContainElement("--result-file-image-vulnerabilities"),
								"per-platform image-processing in %s should have --result-file-image-vulnerabilities", task.Name)
					}
				}
			}
		}
	})

		Context("when the strategy references output-directory", func() {
			BeforeEach(func() {
				var err error
				clusterBuildStrategy, err = ctl.LoadCBSWithName("crane-pull", []byte(test.ClusterBuildStrategyForVulnerabilityScanning))
				Expect(err).ToNot(HaveOccurred())
			})

			It("adds output-directory param to the pipeline spec", func() {
				pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
				Expect(err).ToNot(HaveOccurred())

				var found bool
				for _, p := range pr.Spec.PipelineSpec.Params {
					if p.Name == "shp-output-directory" {
						found = true
					}
				}
				Expect(found).To(BeTrue(),
					"pipeline spec should have shp-output-directory param when strategy uses it")
			})

			It("includes --push with output-directory in per-platform image-processing args", func() {
				pr, err := resources.GeneratePipelineRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)
				Expect(err).ToNot(HaveOccurred())

				for _, task := range pr.Spec.PipelineSpec.Tasks {
					if task.Name == "build-linux-amd64" || task.Name == "build-linux-arm64" {
						for _, step := range task.TaskSpec.Steps {
							if step.Name == "image-processing" {
								Expect(step.Args).To(ContainElement("--push"),
									"per-platform image-processing in %s should have --push arg", task.Name)
							}
						}
					}
				}
			})
		})
	})

})
