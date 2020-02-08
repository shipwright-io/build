## Build v2

*Proposal / *Work-in-progress

An API to build images on Kubernetes using popular strategies and tools like source-to-image, buildpack-v3, kaniko and buildah, in an extensible way.

## How

### Buildpacks v3

```
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: example-build
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy: "buildpacks-v3"
  builderImage: "cloudfoundry/cnb:bionic"
  outputImage: "image-registry.openshift-image-registry.svc:5000/sbose/nodejs-ex"
```

### Source-to-Image (s2i )

```
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: example-build
spec:
  # Add fields here
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy: "s2i"
  builderImage: "docker.io/centos/nodejs-10-centos7"
  outputImage: "image-registry.openshift-image-registry.svc:5000/sbose/nodejs-ex"
```


Uses Golang 1.13 and operator-sdk 1.15.1