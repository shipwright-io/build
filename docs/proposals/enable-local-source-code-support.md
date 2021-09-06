<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: Enable Local Source Code Support
authors:
  - "@qu1queee"
reviewers:
  - "@SaschaSchwarze0"
  - "@ImJasonH"
  - "@HeavyWombat"
approvers:
  - "@SaschaSchwarze0"
  - "@otaviof"
creation-date: 2021-04-05
last-updated: 2021-05-03
status: implementable
---

# Enable Local Source Code Support

**Build Enhancement Proposals have been moved into the Shipwright [Community](https://github.com/shipwright-io/community) repository. This document holds an obsolete Enhancement Proposal, please refer to the up-to-date [SHIP](https://github.com/shipwright-io/community/blob/main/ships/0016-enable-local-source-code-support.md) for more information.**


## Release Signoff Checklist

- [x] Enhancement is `implementable`
- [x] Design details are appropriately documented from clear requirements
- [x] Test plan is defined
- [x] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Summary

Enable Shipwright users to build images from local source code, without the need of involving a Git repository service ( _e.g Github, Gitlab_ ).

Enabling local source code requires a coordination of work between [Shipwright/CLI](https://github.com/shipwright-io/cli) and [Shipwright/Build](https://github.com/shipwright-io/build), where the first one implements a bundle mechanism to push the source code into a self-extracted image and the second one exposes changes in the Build API and embeds the self-extracted image on runtime ( _prepended step_ ) inside the strategies.
## Motivation

There are several reasons for the need of local source code:

- **Development Mode Support**: Provide developers the ability to build container images while working on development mode, without forcing them to _commit/push_ changes to their related version control system ( _git_ ). This should also enable any workflow using Shipwright to provide a full end-to-end flow, where developers can build and deploy from their local workstation.

- **Closing The Gaps**: Local source code support inlines with the continuous effort on improving the developer experience. There are different vendors or projects that indirectly support this feature. A popular example is [Cloud Foundry](https://docs.cloudfoundry.org/devguide/deploy-apps/deploy-app.html) with the `cf push` experience, enabling developers to build and deploy any application from local source code. Also [Kf](https://cloud.google.com/migrate/kf/docs/2.3/quickstart) provides the same developer experience, but it runs on Kubernetes.

### Goals

- Fulfill a missing feature in Tekton/Pipeline repository, in order to provide support for local source code when executing a series of Task steps.

- Enhance the Build resource API, by introducing new fields that signalize the usage of local source code.

- Conclude on the best way to expose the usage of local source code via the Build API.

- Ensure local source code is transparent to users, in other words, we should not need to duplicate Shipwright strategies, this should be all done via a Build definition.

- Make the Build `spec.source.url` not mandatory.

- Layout a `bundle` mechanism on how to exploit the usage of a container registry, to push/pull local source code in combination with the Shipwright Strategies. The implementation of this mechanism should be done in Shipwright/CLI repo. _Note_: Bundle refers to packaging source code into a container image.

### Non-Goals

- Layout all different mechanisms to support local source code in Shipwright/CLI. For example, something that bypass a bundle approach. This proposal should not restrict other alternatives for supporting this feature in the future.

- Modify Shipwright strategies to support local source code.

- Define how to surface in the BuildRun status fields metadata about the local source code.

- Layout all non-functional requirements on the bundles feature, like _performance_ or _scalability_.

## Proposal

This EP is well aware of all potential changes for the Build API, such as:

- [Remote Artifacts](https://github.com/shipwright-io/build/blob/master/docs/proposals/remote-artifacts.md)
- [Build Inputs Overhaul](https://github.com/shipwright-io/build/pull/652)

Due to the state of the Remote Artifacts ( _exists in the API but is not yet fully functional_ ) feature and the uncertainty of the Build Inputs Overhaul proposal, this EP proposes to do the following changes in the current state of the Build API, meaning under `spec.source`.

### Local Source via Bundles

The Bundle mechanism has it´s origins from this [Bundles](https://github.com/mattmoor/mink/tree/master/pkg/bundles) package and it´s based on the premise that if a user is pushing a container image to a container registry, then workflows could use the same approach to push the local source code, and pull it at a later stage during the building image mechanism.

The Bundle mechanism executes the following:

- [1] Takes the source code from a local directory.
- [2] Build a container image that can self-extracted on runtime.
- [3] Push the container image with the source code into a container registry, making it available for further usages.
- [4] The Build controller on signalization of local source code feature, will prepend a step with the bundle image generated in [3] on the Strategy steps, as a replacement for the current image that pulls the source code from git.

_Note_: To bundle the source code, an entity ( _Shipwright CLI_ ) will require to run code that will take the local source code from the user and put it inside a container image.

_Note_: On runtime, when a Shipwright BuildRun is executing, that container image will be pulled and should self-extract its bundle, making the source code available ( _placed on a shared volume_ ) for other containers ( _like BuildKit, Kaniko, et al._ ) to use it.

_Note_: Any workflow using Shipwright that intends to provide support for local source code will require to re-use the Shipwright/CLI implementation or implement a standalone one.

### API Changes

The changes in the API will provide users the following functionalities:

- [1] Specify from where ( _local directory_ ) to build a container image that will bundle the source code and that will self-extract its contents at runtime. Also known as `bundle` image.
- [2] Specify an existing container image to use, for pulling the source code.

This EP is merelely concentrated on [1], but will make [2] easily to implement at a later stage.

The API changes should consider the following features:

- **Ability to upload/download source code**: Users should be able to specify how to build a bundle image. Users should be able to specify which bundle image to use, if it already exists.

- **Ability to provide means of authentication to a registry**: Users should be able to authenticate when pulling/pushing to their container registry of choice.

- **Ability to clean-up images if desired**: Users should have means to delete bundle images after the application container image is build.

- **Ability to specify what to bundle**: Users should be able to specify an absolute path to their local source code directory.

- **Ability to ensure reproducible Builds for bundle images**: Users should be able to verify if source code corresponds to a particular bundle image. To achieve this we can apply the pattern of reproducible builds, by pinning images timestamps to a particular time in the past. An example of the concept on reproducible builds is explained in [here](https://reproducible-builds.org/).

- **Ability to specify particular directories that should not be bundle**: Users should be able to signalize which directories to not bundle, e.g. _/vendor_ . Reasonable defaults like _.github_ should be in place. Similar to the definitions under a `.gitignore` or `.dockerignore` file.

### Proposal 01: API modifications

> Extend `spec.source`

 Introduce `spec.source.container`, which should host all the necessary metadata to signalize the usage of local source code, this should be a go `struct` which indicates what to bundle.

 See an API [example](https://github.com/qu1queee/build/blob/qu1queee/local_source/pkg/apis/build/v1alpha1/source.go#L46-L59) . The [Container](https://github.com/qu1queee/build/blob/qu1queee/local_source/pkg/apis/build/v1alpha1/source.go#L46) struct should **at least** allow users to define the following:

 1. A `spec.source.container.image` mandatory [field](https://github.com/qu1queee/build/blob/qu1queee/local_source/pkg/apis/build/v1alpha1/source.go#L49) for specifying the container image endpoint, registry and repository to build. We should support only image references by digest. If we would like to support image references by tag, then we should consider surfacing the digest of the image under the BuildRun Status fields.
 We expect Shipwright/CLI to push this image and Shipwright/Build will pull it and run its entrypoint ( _and never push it_ ).

 On top of that, to fulfill the features defined above, we could extend the API to include:

 2. A `spec.source.container.pruneBundle` optional field. To signalize the deletion of the bundle image at the very end of the execution of a strategy. Should be disable by _default_.

 _Note_: To be decided later on how to allow authentication for container images as a source of code via `spec.source.container`. At the moment some of the options are:

   - Reuse the `spec.source.credentials`. However this is intended only for authentication to git, for pulling source code.
   - Reuse the `spec.output.credentials`. However the scenario where the authentication applies to both pulling ( _image with code_ ) and pushing ( _final container image_ ), might be uncommon.
   - Add a new `spec.source.container.credentials`. Might not be needed in all cases, as one could pull a bundle that is publicly available.

> Make `spec.source.url` a none mandatory field

 1. This was done thinking on assets only hosted in `git`, which no longer holds true. See an [example](https://github.com/qu1queee/build/blob/qu1queee/local_source/pkg/apis/build/v1alpha1/source.go#L19-L20).

### Proposal 02: Runtime Modifications

> Ensure that the bundle image is prepended as the first step in the Build strategies.

  1. When `spec.source.container.image` is defined, we should  not longer create the Tekton Input PipelineResource. We do this today, to tell Tekton that we want to pull source from a git repository, which ends as a container that pulls it. See an [example](https://github.com/qu1queee/build/blob/qu1queee/local_source/pkg/reconciler/buildrun/resources/taskrun.go#L184-L185) on future changes. It might happen that at the time of implementing this EP we do not longer use the git PipelineResource but rather our in-house custom [image](https://github.com/shipwright-io/build/pull/751). For both scenarios, we will need to ensure that the `git` image is not present any longer on the steps.
  
  2. We need to **prepend** a new step in our Task step definition, which will pull our local source code from the bundle image. See an [example](https://github.com/qu1queee/build/blob/qu1queee/local_source/pkg/reconciler/buildrun/resources/taskrun.go#L171-L186). Important to notice, the image to pull will be a self-extracted image, therefore the `workingDir` container definition should be under `/workspace/source`, which is a well-known path in the Shipwright strategies, where source code is expected to be. It might happen that at the time of implementing this EP we do not longer use `/workspace/source` as we are continuously deprecating in Shipwright some of the custom Tekton behaviours. We will need to ensure the bundle image extracts the code on the path where Shipwright custom images expect it to be.

  3. If authentication was specified for the bundle image, we need to ensure we have that mounted in the pod either in the form of a secret or a service-account. _Note_: We eventually might stop using Tekton service-account support in TaskRun resources, this will force us to rely on something like `spec.imagePullSecrets` at the pod level.

### Proposal 03: Self-Extractable Base Image

> Build an image that will serve as the base layer for a bundle image

  1. Build an image that can self-extract code, by walking the existing tarball and copying the contents into a particular directory.

  2. Requires a new repository in our quay.io account.

  3. Should avoid usage of root users or any privilege mode.

_Note_: To be define where to host the image source code, but preferably in the CLI side.

### Proposal 04: Bundle Mechanism ( _CLI_ )

> Introduce a bundle `pkg`in the CLI side

 See an [example](https://github.com/qu1queee/cli/tree/qu1queee/crud_cmd/pkg/shp/bundle). The current examples provided in this EP, re-use the existing [Bundles](https://github.com/mattmoor/mink/tree/master/pkg/bundles) package, but for Shipwright/CLI we require to have our own custom implementation.

 1. Develop a Bundle pkg that can walk a given directory and produce a consumable tarball for later usage.

 2. The above bundle pkg should append the tarball layer with the base image on Proposal 03 and produce the final bundle image. _Note_: We should ensure that the final bundle image is compatible with the Kubernetes cluster architecture.

 3. Ensure that the Bundle pkg can authenticate when pushing the bundle image to the registry. This will require local authentication to a registry. For example a `docker login` in the user workstation.

 4. Ensure that the Bundle pkg have reasonable defaults on directories that should not be included in the generated tarball.

 5. Ensure that on image creation, we can follow the reproducible build pattern. By pinning image timestamps to a fixed date.

 6. Modify the existing CLI subcommand to surface the usage of local source code. If this is the case, it should bundle and create an image. See an [example](https://github.com/qu1queee/cli/blob/qu1queee/crud_cmd/pkg/shp/cmd/build/create.go#L115-L134). Ideally the subcommand to trigger the usage of the local source code, should support the following features:

   - A flag to specified an absolute `path` where the local source code is located in the local workstation.
   - We should consider having a configuration file, where reasonable defaults for the `shp` subcommands and flags can be specified.
   - A flag to provide a list of directories that we shouldn´t bundle. For example a `vendor` folder in a go project.


### User Stories [optional]

Build users need to define the required parameter values if they want to opt-in for the usage of local source code feature.

#### As a Shipwright/Build contributor I want to have a well defined API to support local source code

The Build resource API needs to provide means of signalizing the desire of local source code. In terms of API changes this should be minimal.

Therefore, users should be able to signalize this feature via:

```yaml
spec:
  source:
    container:
      image: docker.io/<registry-repository>/abundle@sha256:3235326357dfb65f17 # Note: full digest incomplete
```

#### As a Shipwright User I want to build images in Development mode from my local Dockerfile or plain source code

Users should only need to specify the mandatory fields under `spec.source.container` without modifying their existing strategies of choice. Local source code feature should work out of the box with different tools, like Kaniko, BuildKit, Buildpacks, et al.

This will be done via Shipwright/CLI in the form of:

```sh
shp build create a-build --local-path "/absolute-path/src/github.com/shipwright-io/sample-go/docker-build" --output-image "docker.io/<registry-repository>/img:latest" --strategy "kaniko"
```

_Note_: The above is just an example on how the CLI might look like.

#### As a Shipwright User I want to ensure reproducible bundle images

We want to provide trust to users when building bundle images with their source code, so that on multiple builds with the same local source code, the bundle image always express the same _SHA_. Subsequent Builds should generate the same image if the source code is unchanged.

#### As a Shipwright User I want to avoid excessive registry storage from Bundle images

We want to allow users to signalize the desire of pruning a bundle image that was used for building their application container image.

Adding `pruneBundle` should ensure a mechanism for deleting `docker.io/<registry-repository>/bundle:latest`.

```yaml
---
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: a-kaniko-build
spec:
  source:
    container:
      image: docker.io/<registry-repository>/bundle:latest
      pruneBundle: true # Disable by default
  strategy:
    name: kaniko
    kind: ClusterBuildStrategy
  output: ...
```

The above Build definition should ensure that a step in our [TaskSpec](https://github.com/qu1queee/build/blob/qu1queee/local_source/pkg/reconciler/buildrun/resources/taskrun.go#L59) definition is appended, which will ensure a pruning of the image in the related container registry.

_Note_: Supporting multiple registry providers might be too complicated and different alternatives should be considering for pruning.

#### As a Shipwright/CLI contributor I want to provide a mechanism to support local source code

This is related to the CLI workflow, that should support the bundle approach and maintain the bundle base image. The implementation should be generic enough for other workflows, like OpenShift Build, IBM Cloud Code Engine, to consume it.

### Implementation Details/Notes/Constraints [optional]

The following provides an example of core concepts and how do they relate:

1. Users require to create a Build via the SHP CLI, see example:

   ```sh
   shp build create "local-build" \
   --output-image "docker.io/<your-registry-repository>/local-go-kaniko:latest" \
   --strategy "kaniko" \
   --local-path "<your-absolute-local-path>/github.com/shipwright-io/sample-go/docker-build" \
   --bundle-image "docker.io/<your-registry-repository>/latest-bundle:latest"
   ```

   Users are required to define two flags on the `create` subcommand:

   - `--bundle-image`: Where to put the bundle. Note that this could have a reasonable default, based on the value of `--output-image`.
   - `--local-path`: From where in the local workstation to retrieve the source code.

   The SHP CLI will then:

   - a. Build the bundle image based on `--bundle-image` if specified, if not a reasonable default will be used.
   - b. Create a Build and specify under `spec.source.container.image` a reference to the image create in a), that includes a `digest`.

2. Users should trigger the created Build via the SHP CLI, see example:

   ```sh
   shp build run "local-build"
   ```

_Note_: There is a prototype for educational purposes that can be tested if desire, see the related [documentation](https://github.com/qu1queee/cli/blob/qu1queee/crud_cmd/README.md). This uses the following two branches:

- Changes in Shipwright/Build, see [branch](https://github.com/qu1queee/build/tree/qu1queee/local_source)
- Changes in Shipwright/CLI, see [branch](https://github.com/qu1queee/cli/tree/qu1queee/crud_cmd)



### Risks and Mitigations

- This requires coordination between Build and CLI. A well defined list of issues for both backlogs should be created.

- Using a container registry to push/pull the source code might bring concerns around performance. This EP only covers functional requirements. Non-functional requirements like performance or scalability should be discuss during the proposed implementation, and via further development cycles.

- The approach of bundles should not lock-in the CLI with a single approach. It should be flexible enough, to extend to new approaches in the future.

## Design Details

### Test Plan

- Requires development of the self-extractable base image in Shipwright/CLI.
- Requires development of the bundles `pkg` in Shipwright/CLI.
- Requires development of the API extension and the runtime changes in Shipwright/Build.
- This requires new unit, integration and a new e2e tests with local source code in both CLI and Build.

### Graduation Criteria

Should be part of any release before our v1.0.0

### Upgrade / Downgrade Strategy

Not apply, this should not break anything, is rather an extension of the API.

### Version Skew Strategy

N/A

## Implementation History

N/A

## Drawbacks

This approach relies on a container registry being present, which should be secure in a way so that is safe for users to push source code to. Not all Kubernetes Clusters have a container registry inside it, not all developers have direct access to a container registry.

# Alternatives

## Uploading Data to the Cluster

As mentioned before in this enhancement proposal document, currently a container build executes a `git clone` before the actual container-image build starts, this information is stored in a Kubernetes volume mounted at `/workspace/source`.

Previously, in [issue #97](https://github.com/shipwright-io/build/issues/97), we've explored a different mechanism to receive users' data upload, instead of executing the regular `git clone`. This mechanism is comparable to `kubectl cp` and `oc rsync`.

1. Make the build POD wait for users upload, by adding a init-container which will receive the data
2. The init-container saves the payload on a locally mounted volume, analogous to `/workspace/source`
3. The `BuildStrategy` steps takes place
4. Kubernetes will handle the volume left over by the build POD, when a `emptyDir: {}` type of volume is employed, the data is discarded right after the build is complete

As a general solution this alternative approach can leverage different type of Kubernetes volumes, and allows the cluster-administrator to grant privileges to end-users. In other words, it offers fine grain control over allowing specific users to upload data, and once the data is uploaded the operator can decide what will happen next.

This approach requires more exploration of the building blocks employed on `kubectl cp` and `oc rysnc`, Kubernetes port-forwarding and more.
