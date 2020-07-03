package test

// MinimalBuildahBuild defines a simple
// Build with a source and a strategy
const MinimalBuildahBuild = `
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildah
spec:
  source:
    url: "https://github.com/sbose78/taxi"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
`

// BuildahBuildWithOutput defines a simple
// Build with a source, strategy and output
const BuildahBuildWithOutput = `
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildah
  namespace: build-test
spec:
  source:
    url: "https://github.com/sbose78/taxi"
  strategy:
    name: buildah
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildpacksBuildWithBuilderAndTimeOut defines a Build with
// source, strategy, builder, output and
// timeout
const BuildpacksBuildWithBuilderAndTimeOut = `
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildpacks-v3
  namespace: build-test
spec:
  source:
    url: "https://github.com/sbose78/taxi"
    revision: master
    contextDir: src
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
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildah
  namespace: build-test
spec:
  source:
    url: "https://github.com/sbose78/taxi"
  strategy:
    name: buildah
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 30s
`
