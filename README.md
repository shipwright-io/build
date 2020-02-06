## Build v2

*Proposal / *Work-in-progress

An API to build images on Kubernetes using popular strategies and tools like source-to-image, buildpack-v3, kaniko and buildah, in an extensible way.

## How

Define the `Build` using a strategy.

```
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: my-java-app-build
spec:
  strategy: buildpacks-v3
  builderImage: quay.io/java8/buildpack-builder
  output: quay.io/my/app
```

