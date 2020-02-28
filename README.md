## Build v2

*Proposal / *Work-in-progress

An API to build images on Kubernetes using popular strategies and tools like source-to-image,
buildpack-v3, kaniko and buildah, in an extensible way.

## How

The following are the `BuildStrategies` supported by this operator, out-of-the-box:

* [Source-to-Image](samples/buildstrategy/buildstrategy_source-to-image_cr.yaml)
* [Buildpacks-v3](samples/buildstrategy/buildstrategy_buildpacks-v3_cr.yaml)
* [Buildah](samples/buildstrategy/buildstrategy_buildah_cr.yaml)
* [Kaniko](samples/buildstrategy/buildstrategy_kaniko_cr.yaml)


Users have the option to define their own(custom) `BuildStrategies` and make them available for consumption
by `Builds`.

## Builds

### Buildpacks v3

Create the below CR for starting a buildpacks-v3 `Build`

```yml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: example-build-buildpack
spec:
  # Git Source definition
  source:
    url: https://github.com/sclorg/nodejs-ex
    credentials:
      name: github-auth-olemefer

  # Strategy defined in the buildpacks-v3 CR 
  # in the 'openshift' namespace.
  strategy: 
    name: "buildpacks-v3"
    namespace: <NAMESPACE>

  # Build to be run in this image.
  builderImage: "heroku/buildpacks:18"

  # Generated image.
  output:
    image: "quay.io/<USERNAME>/<REPO-NAME>"
    credentials:
      name: <SECRET-NAME>
```
The secret can be created like `kubectl create secret generic <SECRET-NAME> --from-file=.dockerconfigjson=<PATH/TO/.docker/config.json> --type=kubernetes.io/dockerconfigjson`

### Source-to-Image (`s2i`)

Create the below CR for starting an s2i `Build`

```yml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: s2i-nodejs-build
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy:
    name: "source-to-image"
    namespace: <NAMESPACE>
  builderImage: "docker.io/centos/nodejs-10-centos7"
  output:
    image: "image-registry.openshift-image-registry.svc:5000/sbose/nodejs-ex"
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
  dockerfile: Dockerfile
  strategy:
    name: "buildah"
    namespace: <NAMESPACE>
  output:
    image: 'image-registry.openshift-image-registry.svc:5000/sbose/taxi-app'
  source:
    url: 'https://github.com/sbose78/taxi'
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
  strategy: 
    name: "kaniko"
    namespace: <NAMESPACE>
  dockerfile: "Dockerfile" 
  pathContext: "./"
  output:
    image: "image-registry.openshift-image-registry.svc:5000/sbose/taxi-app"
```

On **Reconcile**, the `Build` CR's `Status` gets updated,

```yml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: kaniko-golang-build
spec:
  source:
    url: https://github.com/sbose78/taxi
  strategy: 
    name: "kaniko"
    namespace: <NAMESPACE>
  dockerfile: "Dockerfile" 
  pathContext: "./"
  output:
    image: "image-registry.openshift-image-registry.svc:5000/sbose/taxi-app"
status:
  status: Running
```

----


## Try it!

- Install Tekton ( You could use the OpenShift Pipelines Community Operator ).
- Execute `./hack/crd.sh install`
- Start a sample [Kaniko](samples/build/build_kaniko_cr.yaml) build

## Development

*  Build, test & run using [HACK.md](HACK.md).

----


## Roadmap


### Status of support for build strategies

| Build Strategy  | Alpha | Beta | GA Support
| ------------- | ------------- | ------------- | ------------- |
| [Source-to-Image](samples/buildstrategy/buildstrategy_source-to-image_cr.yaml)  | ☑️ | 
| [Buildpacks-v3](samples/buildstrategy/buildstrategy_buildpacks-v3-cr.yaml)  | ⚪️ |
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

