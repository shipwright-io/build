## Build v2

*Proposal / *Work-in-progress

An API to build images on Kubernetes using popular strategies and tools like source-to-image,
buildpack-v3, kaniko and buildah, in an extensible way.

## How

The following CRs are examples of `BuildStrategy` supported by this operator:

* [Source-to-Image](samples/buildstrategy/buildstrategy_source-to-image_cr.yaml);
* [Buildpacks-v3](samples/buildstrategy/buildstrategy_buildpacksv3-cr.yaml);
* [Buildah](samples/buildstrategy/buildstrategy_buildah_cr.yaml);
* [Kaniko](samples/buildstrategy/buildstrategy_kaniko_cr.yaml);


Users have the option to define their own `BuildStrategy`s and make them available for consumption
by `Build`s.

## `BuildStrategy`

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
  strategy: "buildah"
  dockerfile: "Dockerfile"
  outputImage: "image-registry.openshift-image-registry.svc:5000/example/taxi-app"
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
  strategy: "source-to-image"
  builderImage: "docker.io/centos/nodejs-10-centos7"
  outputImage: "image-registry.openshift-image-registry.svc:5000/sbose/nodejs-ex"
Status:
  status: Running
```

## Development

* This project uses Golang 1.13+ and operator-sdk 1.15.1.
* The controllers create/watch Tekton objects.

### Running the Operator

Assuming you are logged in to an OpenShift/Kubernetes cluster, run

```sh
make clean && make build && make local
```

If the `pipeline` service account isn't already created, here are the steps to create the same:

```sh
oc create serviceaccount pipeline
oc adm policy add-scc-to-user privileged -z pipeline
oc adm policy add-role-to-user edit -z pipeline
```

If your `Build`'s `outputImage` is to be pushed to the OpenShift internal registry, ensure the
`pipeline` service account has the required role:

```sh
oc policy add-role-to-user registry-editor pipeline
```

Or

```sh
oc policy add-role-to-user  system:image-builder  pipeline
```

In the near future, the above would be setup by the operator.
