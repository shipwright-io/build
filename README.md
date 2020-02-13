## Build v2

[![Docker Repository on Quay](https://quay.io/repository/redhat-developer/buildv2/status "Docker Repository on Quay")](https://quay.io/repository/redhat-developer/buildv2)


*Proposal / *Work-in-progress

An API to build images on Kubernetes using popular strategies and tools like source-to-image, buildpack-v3, kaniko and buildah, in an extensible way.

## How

The `Build` examples below are powered by the following  `BuildStrategy` CRs

* The [Source-to-Image](samples/buildstrategy/buildstrategy_source-to-image_cr.yaml) `BuildStrategy` CR 
* The [Buildpacks-v3](samples/buildstrategy/buildstrategy_buildpacksv3-cr.yaml)  `BuildStrategy` CR
* The [Buildah](samples/buildstrategy/buildstrategy_buildah_cr.yaml)  `BuildStrategy` CR
* The [Kaniko](samples/buildstrategy/buildstrategy_kaniko_cr.yaml)  `BuildStrategy` CR


Users have the option to define their own `BuildStrategy`s and make them available for consumption by `Build`s

### Buildpacks v3

Create the below CR for starting a buildpacks-v3 `Build`

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

Create the below CR for starting an s2i `Build`

```
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
```

### Buildah

Create the below CR for starting a Buildah `Build`

```
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/sbose78/taxi
  strategy: "buildah"
  dockerfile: "Dockerfile" 
  outputImage: "image-registry.openshift-image-registry.svc:5000/sbose/taxi-app"
```

### Kaniko

Create the below CR for starting a Kaniko `Build`

```
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/sbose78/taxi
  strategy: "kaniko"
  dockerfile: "Dockerfile" 
  pathContext: "./"
  outputImage: "image-registry.openshift-image-registry.svc:5000/sbose/taxi-app"
  ```


On **Reconcile**, the `Build` CR's `Status` gets updated,

```
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

This project uses Golang 1.13+ and operator-sdk 1.15.1



### Running the Operator

Assuming you are logged in to an OpenShift/Kubernetes cluster, run

```
make clean && make build && make local
```

If the `pipeline` service account isn't already created, here are the steps to
create the same:

```
oc create serviceaccount pipeline
oc adm policy add-scc-to-user privileged -z pipeline
oc adm policy add-role-to-user edit -z pipeline
```

If your `Build`'s `outputImage` is to be pushed to the OpenShift internal registry, ensure the `pipeline` service account has the required role:

```
oc policy add-role-to-user registry-editor pipeline
```

Or

```
oc policy add-role-to-user  system:image-builder  pipeline
```

In the near future, the above would be setup by the operator.
