// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package test

// MinimalBuildahBuild defines a simple
// Build with a source and a strategy
const MinimalBuildahBuild = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
`

// BuildahBuildWithOutput defines a simple
// Build with a source, strategy and output
const BuildahBuildWithOutput = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah
  namespace: build-test
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildpacksBuildWithBuilderAndTimeOut defines a Build with
// source, strategy, builder, output and
// timeout
const BuildpacksBuildWithBuilderAndTimeOut = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildpacks-v3
  namespace: build-test
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
    contextDir: docker-build
  strategy:
    name: buildpacks-v3
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
  builder:
    image: heroku/buildpacks:18
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 30s
`

// BuildahBuildWithTimeOut defines a Build for
// Buildah with source, strategy, output and
// timeout
const BuildahBuildWithTimeOut = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah
  namespace: build-test
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 30s
`

// BuildBSMinimal defines a Build with a BuildStrategy
const BuildBSMinimal = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSMinimal defines a Build with a
// ClusterBuildStrategy
const BuildCBSMinimal = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
    contextDir: docker-build
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSMinimalWithFakeSecret defines a Build with a
// ClusterBuildStrategy and an not existing secret
const BuildCBSMinimalWithFakeSecret = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    credentials:
      name: fake-secret
`

// BuildWithOutputRefSecret defines a Build with a
// referenced secret under spec.output
const BuildWithOutputRefSecret = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    credentials:
      name: output-secret
  timeout: 5s
`

// BuildWithSourceRefSecret defines a Build with a
// referenced secret under spec.source
const BuildWithSourceRefSecret = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
    credentials:
      name: source-secret
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildWithBuilderRefSecret defines a Build with a
// referenced secret under spec.builder
const BuildWithBuilderRefSecret = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  builder:
    image: heroku/buildpacks:18
    credentials:
      name: builder-secret
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildWithMultipleRefSecrets defines a Build with
// multiple referenced secrets under spec
const BuildWithMultipleRefSecrets = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
    credentials:
      name: source-secret
  builder:
    image: heroku/buildpacks:18
    credentials:
      name: builder-secret
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildCBSWithShortTimeOut defines a Build with a
// ClusterBuildStrategy and a short timeout
const BuildCBSWithShortTimeOut = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildCBSWithWrongURL defines a Build with a
// ClusterBuildStrategy and a non-existing url
const BuildCBSWithWrongURL = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  annotations:
    build.shipwright.io/verify.repository: "true"
spec:
  source:
    url: "https://github.foobar.com/sbose78/taxi"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSWithVerifyRepositoryAnnotation defines a Build
// with the verify repository annotation key
const BuildCBSWithVerifyRepositoryAnnotation = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  annotations:
    build.shipwright.io/verify.repository: ""
spec:
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSWithoutVerifyRepositoryAnnotation defines a minimal
// Build without source url and annotation
const BuildCBSWithoutVerifyRepositoryAnnotation = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSWithBuildRunDeletion defines a Build with a
// ClusterBuildStrategy and the annotation for automatic BuildRun
// deletion
const BuildCBSWithBuildRunDeletion = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  annotations:
    build.shipwright.io/build-run-deletion: "true"
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`
