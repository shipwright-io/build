package e2e

import (
	"fmt"

	operatorapis "github.com/redhat-developer/build/pkg/apis"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"k8s.io/kubectl/pkg/scheme"
)

// buildahBuild Test data setup
func buildpackBuildTestData(ns string, identifier string) (*operator.Build, *operator.BuildStrategy) {

	decode := scheme.Codecs.UniversalDeserializer().Decode
	operatorapis.AddToScheme(scheme.Scheme)

	yaml := getBuildpacksBuildStrategyYaml()
	obj, _, err := decode([]byte(yaml), nil, nil)
	if err != nil {
		fmt.Printf("%#v", err)
	}

	buildpackBuildStrategy := obj.(*operator.BuildStrategy)

	buildpackBuildStrategy.SetNamespace(ns)

	yaml = getBuildpacksBuildYaml()
	obj, _, err = decode([]byte(yaml), nil, nil)
	if err != nil {
		fmt.Printf("%#v", err)
	}
	buildpackBuild := obj.(*operator.Build)

	buildpackBuild.SetNamespace(ns)
	buildpackBuild.SetName(identifier)

	return buildpackBuild, buildpackBuildStrategy
}

// TODO: read from the yaml files once we start running these on CI.
// Keeping these here as a baseline of what ALWAYS works.
func getBuildpacksBuildStrategyYaml() string {
	buildpacksYaml := `
apiVersion: build.dev/v1alpha1
kind: BuildStrategy
metadata:
  name: buildpacks-v3
spec:
    buildSteps:
    - name: step-prepare
      image: $(build.builderImage)
      securityContext:
        runAsUser: 0
      command:
        - /bin/bash
      args:
        - -c
        - chown -R "1000:1000" "/workspace/source"
    - name: step-detect
      image: $(build.builderImage)
      securityContext:
        runAsUser: 1000
      command:
        - /cnb/lifecycle/detector
      args:
        - -app=/workspace/source
        - -group=/layers/group.toml
        - -plan=/layers/plan.toml
      volumeMounts:
        - name: layers-dir
          mountPath: /layers
    - name: step-restore
      image: $(build.builderImage)
      securityContext:
        runAsUser: 1000
      command:
        - /cnb/lifecycle/restorer
      args:
        - -layers=/layers
        - -cache-dir=/cache
        - -group=/layers/group.toml
      volumeMounts:
        - name: cache-dir
          mountPath: /cache
        - name: layers-dir
          mountPath: /layers
    - name: step-build
      image: $(build.builderImage)
      securityContext:
        runAsUser: 1000
      command:
        - /cnb/lifecycle/builder
      args:
        - -app=/workspace/source
        - -layers=/layers
        - -group=/layers/group.toml
        - -plan=/layers/plan.toml
      volumeMounts:
        - name: layers-dir
          mountPath: /layers
    - name: step-export
      image: $(build.builderImage)
      securityContext:
        runAsUser: 0
      command:
        - /cnb/lifecycle/exporter
      args:
        - -app=/workspace/source
        - -layers=/layers
        - -cache-dir=/cache
        - -group=/layers/group.toml
        - -helpers=true
        - $(build.output.image)
      volumeMounts:
        - name: cache-dir
          mountPath: /cache
        - name: layers-dir
          mountPath: /layers
`
	return buildpacksYaml
}

func getBuildpacksBuildYaml() string {
	buildpacksYaml := `
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildpack-nodejs-build
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy:
    name: buildpacks-v3
  builderImage: heroku/buildpacks:18
  output:
    image: quay.io/example/nodejs-ex:latest
`
	return buildpacksYaml
}
