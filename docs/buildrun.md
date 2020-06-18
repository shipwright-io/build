# BuildRun

- [Overview](#overview)
- [BuildRun Controller](#buildrun-controller)
- [Configuring a BuildRun](#configuring-a-buildrun)
  - [Defining the BuildRef](#defining-the-buildref)
  - [Defining Resources](#defining-resources)
  - [Defining the ServiceAccount](#defining-the-serviceaccount)
- [BuildRun Status](#buildrun-status)
- [Relationship with Tekton Tasks](#relationship-with-tekton-tasks)

## Overview

The resource `BuildRun` (`buildruns.dev/v1alpha1`) is the build process of a `Build` resource definition which is executed in Kubernetes.

A `BuildRun` resource allows the user to define:

- The `BuildRun` name, through which the user can monitor the status of the image construction.
- A referenced `Build` instance to use during the build construction.
- Compute resources.
- A service account for hosting all related secrets in order to build the image.

A `BuildRun` is available within a namespace.

## BuildRun Controller

The controller watches for:

- Updates on a `Build` resource (_CRD instance_)
- Updates on a `TaskRun` resource (_CRD instance_)

When the controller reconciles it:

- Looks for any existing owned `TaskRuns` and update its parent `BuildRun` status.
- Retrieves the specified `SA` and sets this with the specify output secret on the `Build` resource.
- Generates a new tekton `TaskRun` if it does not exist, and set a reference to this resource(_as a child of the controller_).
- On any subsequent updates on the `TaskRun`, the parent `BuildRun` resource instance will be updated.

## Configuring a BuildRun

The `BuildRun` definition supports the following fields:

- Required:
  - [`apiVersion`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the API version, for example `build.dev/v1alpha1`.
  - [`kind`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the Kind type, for example `BuildRun`.
  - [`metadata`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Metadata that identify the CRD instance, for example the name of the `BuildRun`.
  - `spec.buildRef` - Specifies an existing `Build` resource instance to use.

- Optional:
  - `spec.resources` - Refers to the compute [resources](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/) used on the container where the image is built.
  - `spec.serviceAccount` - Refers to the SA to use when building the image. (_defaults to the `default` SA_)
  - `spec.timeout` - Defines a custom timeout. The value needs to be parsable by [ParseDuration](https://golang.org/pkg/time/#ParseDuration), for example `5m`. The value overwrites the value that is defined in the `Build`.
  - `spec.output.image` - Refers to a custom location where the generated image would be pushed. The value will overwrite the `output.image` value which is defined in `Build`. ( Note: other properties of the output, for example, the credentials cannot be specified in the buildRun spec. )
### Defining the BuildRef

A `BuildRun` resource can reference a `Build` resource, that indicates what image to build. For example:

```yaml
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: buildpack-nodejs-buildrun-namespaced
spec:
  buildRef:
    name: buildpack-nodejs-build-namespaced
```

### Defining Resources

A `BuildRun` resource can define resources, like **limits** to use in the pod where the build execution will take place, for example:

```yaml
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: kaniko-golang-buildrun
spec:
  buildRef:
    name: kaniko-golang-build
  resources:
    limits:
      cpu: "1"
```

### Defining the ServiceAccount

A `BuildRun` resource can define a serviceaccount to use. Usually this SA will host all related secrets referenced on the `Build` resource, for example:

```yaml
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: buildpack-nodejs-buildrun-namespaced
spec:
  buildRef:
    name: buildpack-nodejs-build-namespaced
  serviceAccount:
    name: pipeline
```

You can also use set the `spec.serviceAccount.generate` path to `true`. This will generate the service account during runtime for you.

_**Note**_: When the SA is not defined, the `BuildRun` will default to the `default` SA in the namespace.

## BuildRun Status

The `BuildRun` resource is updated as soon as the current image building status changes:

```sh
$ kubectl get buildruns.build.dev buildpack-nodejs-buildrun
NAME                          SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
buildpack-nodejs-buildrun     Unknown     Running     70s
```

And finally:

```sh
$ kubectl get buildruns.build.dev buildpack-nodejs-buildrun
NAME                          SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
buildpack-nodejs-buildrun     True        Succeeded   2m10s       74s
```

### Build Snapshot

For every BuildRun controller reconciliation, the `buildSpec` in the Status of the `BuildRun` is updated if an existing owned `TaskRun` is present. During this update, a `Build` resource snapshot is generated and embedded into the `status.buildSpec` path of the `BuildRun`. A `buildSpec` is just a copy of the original `Build` spec, from where the `BuildRun` executed a particular image build. The snapshot approach allows developers to see the original `Build` configuration.

## Relationship with Tekton Tasks

The `BuildRun` resource abstracts the image construction by delegating this work to the Tekton Pipeline [TaskRun](https://github.com/tektoncd/pipeline/blob/master/docs/taskruns.md). Compared to a Tekton Pipeline [Task](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md), a `TaskRun` runs all `steps` until completion of the `Task` or until a failure occurs in the `Task`.

The `BuildRun` controller during the Reconcile will generate a new `TaskRun`. During the execution, the controller will embed in the `TaskRun` `Task` definition the requires `steps` to execute. These `steps` are define in the strategy defined in the `Build` resource, either a `ClusterBuildStrategy` or a `BuildStrategy`.
