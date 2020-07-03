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
