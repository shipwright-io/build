## Build v2

[![Docker Repository on Quay](https://quay.io/repository/redhat-developer/buildv2/status "Docker Repository on Quay")](https://quay.io/repository/redhat-developer/buildv2)


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

On Reconcile,

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
Status:
  status: Running

```

## Development

Uses Golang 1.13 and operator-sdk 1.15.1

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

Eventually, the above would be setup by the operator.