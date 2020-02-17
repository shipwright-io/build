## Build v2

*Proposal / *Work-in-progress

An API to build images on Kubernetes using popular strategies and tools like source-to-image,
buildpack-v3, kaniko and buildah, in an extensible way.

## How

The following are the `BuildStrategy`s supported by this operator, out-of-the-box:

* [Source-to-Image](samples/buildstrategy/buildstrategy_source-to-image_cr.yaml);
* [Buildpacks-v3](samples/buildstrategy/buildstrategy_buildpacksv3-cr.yaml);
* [Buildah](samples/buildstrategy/buildstrategy_buildah_cr.yaml);
* [Kaniko](samples/buildstrategy/buildstrategy_kaniko_cr.yaml);


Users have the option to define their own `BuildStrategy`s and make them available for consumption
by `Build`s.

## Strategies

Create resources and configuration in order to implement the following strategies.

### Buildpacks v3

Create the below CR for starting a buildpacks-v3 `Build`

```yml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: example-build
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy: buildpacks-v3
  builderImage: cloudfoundry/cnb:bionic
  outputImage: image-registry.openshift-image-registry.svc:5000/example/nodejs-ex
```

### Source-to-Image (`s2i`)

Create the below CR for starting an s2i `Build`

```yml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: example-build
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy: source-to-image
  builderImage: registry.redhat.io/rhscl/nodejs-12-rhel7:latest
  outputImage: image-registry.openshift-image-registry.svc:5000/example/nodejs-ex
```

### Buildah

Create the below CR for starting a Buildah `Build`

```yml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/sbose78/taxi
  strategy: buildah
  dockerfile: Dockerfile
  outputImage: image-registry.openshift-image-registry.svc:5000/example/taxi-app
```

### Kaniko

Create the below CR for starting a Kaniko `Build`

```yml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: kaniko-golang-build
spec:
  source:
    url: https://github.com/sbose78/taxi
  strategy: kaniko
  dockerfile: Dockerfile
  pathContext: .
  outputImage: image-registry.openshift-image-registry.svc:5000/example/taxi-app
```

On **Reconcile**, the `Build` CR's `Status` gets updated,

```yml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: example-build
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy: source-to-image
  builderImage: docker.io/centos/nodejs-10-centos7
  outputImage: image-registry.openshift-image-registry.svc:5000/sbose/nodejs-ex
status:
  status: Running
```

----


## Running the Operator

Build, test & run using [HACK.md](HACK.md).

----


## Roadmap


### Status of support for build strategies

| Build Strategy  | Alpha | Beta | GA Support
| ------------- | ------------- | ------------- | ------------- |
| [Source-to-Image](samples/buildstrategy/buildstrategy_source-to-image_cr.yaml)  | ☑️ | 
| [Buildpacks-v3](samples/buildstrategy/buildstrategy_buildpacksv3-cr.yaml)  | ⚪️ |
| [Kaniko](samples/buildstrategy/buildstrategy_kaniko_cr.yaml)  | ☑️ |
| [Buildah](samples/buildstrategy/buildstrategy_buildah_cr.yaml)  | ☑️  |


### Status of support for generic features

------

| Feature  | Alpha | Beta | GA Support
| ------------- | ------------- | ------------- | ------------- |
| Private Git Repos  | ☑️ |  |
| Runtime Base Image  | ⚪️ |  |
| Binary builds  |  | |
| Image Caching  |  |  |
| ImageStreams support  |  | |
| Entitlements  |  | |


------

### Key

⚪️  Initial work is in progress

☑️ Validated to be working

✅ Can be shipped

