// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package test

// MinimalBuildahBuildRun defines a simple
// BuildRun with a referenced Build
const MinimalBuildahBuildRun = `
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: buildah-run
spec:
  buildRef:
    name: buildah
`

// BuildahBuildRunWithSA defines a BuildRun
// with a service-account
const BuildahBuildRunWithSA = `
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: buildah-run
  namespace: build-test
spec:
  buildRef:
    name: buildah
  serviceAccount:
    name: buildpacks-v3-serviceaccount
`

// BuildahBuildRunWithSAAndOutput defines a BuildRun
// with a service-account and output overrides
const BuildahBuildRunWithSAAndOutput = `
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: buildah-run
  namespace: build-test
spec:
  buildRef:
    name: buildah
  serviceAccount:
    name: buildpacks-v3-serviceaccount
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app-v2
`

// BuildpacksBuildRunWithSA defines a BuildRun
// with a service-account
const BuildpacksBuildRunWithSA = `
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: buildpacks-v3-run
  namespace: build-test
spec:
  buildRef:
    name: buildpacks-v3
  serviceAccount:
    name: buildpacks-v3-serviceaccount
    generate: false
`

// BuildahBuildRunWithTimeOutAndSA defines a BuildRun
// with a service-account and timeout
const BuildahBuildRunWithTimeOutAndSA = `
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: buildah-run
  namespace: build-test
spec:
  buildRef:
    name: buildah
  serviceAccount:
    name: buildpacks-v3-serviceaccount
  timeout: 1m
`

// MinimalBuildRun defines a minimal BuildRun
// with a reference to a not existing Build
const MinimalBuildRun = `
apiVersion: build.dev/v1alpha1
kind: BuildRun
spec:
  buildRef:
    name: foobar
`

// MinimalBuildRunWithSpecifiedServiceAccount defines a minimal BuildRun
// with a reference to a not existing serviceAccount
const MinimalBuildRunWithSpecifiedServiceAccount = `
apiVersion: build.dev/v1alpha1
kind: BuildRun
spec:
  buildRef:
    name: buildah
  serviceAccount:
    name: foobar
`

// MinimalBuildRunWithSAGeneration defines a minimal BuildRun
// with a reference to a not existing Build
const MinimalBuildRunWithSAGeneration = `
apiVersion: build.dev/v1alpha1
kind: BuildRun
spec:
  serviceAccount:
    generate: true
  buildRef:
    name: foobar
`

// MinimalBuildRunWithTimeOut defines a BuildRun with
// an override for the Build Timeout
const MinimalBuildRunWithTimeOut = `
apiVersion: build.dev/v1alpha1
kind: BuildRun
spec:
  timeout: 1s
  buildRef:
    name: foobar
`

// MinimalBuildRunWithOutput defines a BuildRun with
// an override for the Build Output
const MinimalBuildRunWithOutput = `
apiVersion: build.dev/v1alpha1
kind: BuildRun
spec:
  output:
    image: foobar.registry.com
  buildRef:
    name: foobar
`
