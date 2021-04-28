<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: removal-tekton-resources
authors:
  - "@SaschaSchwarze0"
reviewers:
  - "@HeavyWombat"
  - "@ImJasonH"
approvers:
  - "@qu1queee"
creation-date: 2021-04-10
last-updated: 2021-04-20
status: provisional
---

# Removing Tekton resource usages

## Release Signoff Checklist

- [ ] Enhancement is `implementable`
- [ ] Design details are appropriately documented from clear requirements
- [ ] Test plan is defined
- [X] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Open Questions [optional]

There are some open items insight that EP that can be discussed:

- Convention for system parameters
- Single vs multiple containers to load the sources
- Non-root support

## Motivation

The Tekton community is not moving forward with resources. See these two references:

- [PipelineResources](https://github.com/tektoncd/pipeline/blob/main/docs/resources.md), see the note
- [Replacing PipelineResources with Tasks](https://github.com/tektoncd/pipeline/blob/main/docs/migrating-v1alpha1-to-v1beta1.md#replacing-pipelineresources-with-tasks)

Shipwright Build relies on PipelineResources as it uses two of them [when the TaskRun gets created](https://github.com/shipwright-io/build/blob/33cf526607a47ccefa186ad170ad20afa17fec27/pkg/reconciler/buildrun/resources/taskrun.go#L218-L254):

- a Git resource, specifying this causes Tekton to add a step to clone the git repository and to put the commit sha into the TaskRun status. The commit sha is not further used by Shipwright Build, for example it is not copied into the status of the BuildRun.
- an Image resource, specifying this causes Tekton to add the `create-dir-image` step which creates the `/workspace/output/image` directory, and the image digest exporter step that reads an index.json file from that directory and stores the image digest into the TaskRun status. Shipwright Build does not further use the image digest, for example it is not copied into the status of the BuildRun. The image resource also gives us Tekton's variable `$(outputs.resources.image.url)` to be present which we use to [inject the image URL into the build-strategy steps](https://github.com/shipwright-io/build/blob/v0.4.0/pkg/reconciler/buildrun/resources/taskrun.go#L38).

To ensure we are not getting locked into an older Tekton version, we need to remove our usages of them.

The replacement that Tekton suggests in their migration guide is to use Task (steps) and results.

## Summary

Nothing really to add here. I would repeat the above motivation and the below goals if I would write something here.

### Goals

- Tekton resources are not used anymore.
- A concept to download the sources that are defined in a Build without the need for the build strategy author to do something for this.
- A concept to store metadata about sources and the image in the TaskRun status (to pick it up eventually in a BuildRun).
- Improving the documentation for the build strategy author.
- A build user can continue to use his Builds without being required to make any changes.
- At runtime, the container(s) that load sources can run as a non-root user.

### Non-Goals

- A pluggable mechanism to support additional kinds of sources without changing the Shipwright build controller code-base. (That's a very interesting scenario but is to be covered in another proposal. Once the implementation for our existing source types, Git and HTTP, is common, this extension should be simple.)
- A build strategy author can define custom results in the build strategy to be passed to the BuildRun's status, for example to report which buildpacks have been used in a Paketo build. (Also a very interesting scenario but is to be covered in another proposal.)
- A build user can see metadata about the source (like the commit sha) or output image (like the image digest) in the BuildRun status. (This is very valuable scenario, but this EP will only maintain the status-quo = store those information in the TaskRun status. Another proposal can make use of it.)

## Proposal

### User Stories

- As build strategy author, I do not want to specify how sources are loaded so that I can focus on transforming sources to a container. The download of sources from potentially different systems should continue to be covered by the Shipwright Build system.
- As build strategy author, I want to use well-documented parameters to access the path to the source code, or the name of the output image, so that I do not need to know the internal directory structure (for example `/workspace/source` should not be present in a build strategy).
- As Shipwright Build developer, I want to continue to store metadata about the source (git commit sha) and the output (image digest) in the TaskRun status so that we can eventually expose this information for the Build user in the status of a BuildRun.

### Implementation Details/Notes/Constraints [optional]

We are going to use several concepts to replace Tekton's resource capability:

#### 1. Tekton workspace

By using resources, Tekton automatically creates sub-directories under `/workspace`, specifically, the input resource name becomes the directory name. That's why there is a `/workspace/source` directory. Outputs go into `/workspace/output`. Therefore, our image resource leads to the directory `/workspace/output/image`.

With our own logic taking over responsibility on downloading sources, we also need to make sure there is a directory for this. We will create a [Tekton workspace](https://github.com/tektoncd/pipeline/blob/v0.21.0/docs/workspaces.md#using-workspaces-in-tasks) named `source` backed by an `emptyDir` volume which will then also lead to the `/workspace/source` directory.

For the output image, no specific directory will be required anymore.

#### 2. Tekton Parameters

To pass information from the user's Build and BuildRun into the steps defined by the build strategy author, we will expand our usage of [Tekton parameters](https://github.com/tektoncd/pipeline/blob/v0.21.0/docs/tasks.md#specifying-parameters). A set of system parameters will be made available. As naming convention we will prefix them with `shp-` (`shp` is also the command using by our [CLI](https://github.com/shipwright-io/cli)) to prevent conflicts with those parameters defined by the build strategy author (see [Add EP on Build Strategies Parametrization #697](https://github.com/shipwright-io/build/pull/697)). After the prefix, the name will consist of only lowercase letters and dashes.

Here is the list of parameters that we will make available:

| Name | Purpose |
| ---- | ------- |
| `shp-output-image` | The reference of the image to be built. |
| `shp-source-root` | The path to the directory where we download the sources to. The value is hard-coded and will be `/workspace/source` |
| `shp-source-context` | The path to the context directory inside the sources. The value depends on the presence of the `contextDir` in the user's Build. If it is not present, then the value will be `/workspace/source`, otherwise it will be `path.Join("/workspace/source", build.spec.contextDir)`. The purpose is that a build strategy author is not required to manually concatenate these values. |

The build strategy author uses those parameters using the `$(params.<PARAMETER_NAME>)` syntax.

We stop supporting the `$(build.output.image)` replacement token that is today used to access the image reference from the build strategy.

The build strategy documentation will list above parameters which makes them part of our API.

By using Tekton's replacement logic for parameters, we enable build strategy authors to use them at more places than the few places for which we support our current string transformations (`command`, `args`, `image`).

#### 3. Tekton Results

To return data from logic running inside the container as a Tekton Task step, the corresponding Tekton concept is the [result](https://github.com/tektoncd/pipeline/blob/v0.21.0/docs/tasks.md#emitting-results). We will use results to get data from the build steps into the TaskRun status. From there, we can eventually make it available in the BuildRun result.

The naming pattern of our system-defined results will be the same as for parameters: they will start with the `shp-` prefix and then contain only lowercase letters and dashes, except for those parts defined by the user (see below, the name of the user-defined entry in `spec.sources`).

As of today, we have both `spec.source` for a single Git resource as well as `spec.sources` for additional HTTP resources. Each source can emit one or multiple results. The naming pattern is the following:

- For `spec.source`, it can emit `shp-source-default-${resultName}`. As the source is always of type Git, we will have one result there: `shp-source-default-commit-sha` to store the sha of the commit. Eventually, we can extend the results that a Git source emits, for example with results for the author and message of the commit.
- For `spec.sources`, arbitrary resources can emit `shp-source-${spec.sources[i].name}-${resultName}`. For example, assuming the user's Build contains a HTTP source named `license-text` and the HTTP source would be emitting a result named `size`, then the name of Tekton result will be `shp-source-license-text-size`. Note: this proposal only means to define the naming pattern for the results from `spec.sources`. The decision on which results we introduce for the HTTP sources and for future source types should be covered elsewhere.

The above logic accepts the risk of conflicts if a user's Build contains an item in `spec.sources` that uses the name `default`. The risk is accepted based on the assumption that `spec.source` will eventually be removed and that we currently do not add any result for the HTTP sources.

Tekton results work by expecting the container to write the value of the result to a file. Tekton then puts the content of those files into the termination message of the container and from there, it puts them into the TaskRun's status. Termination messages have a maximum length of 4 kB. This also limits how many results one can fill and how long those values can be. This must be considered in the future when we extend our results.

The results for the sources will be filled by our own code as described later. A build strategy author must not do anything to fill them.

For the output image, we will also provide two more system-defined results: `shp-image-digest` and `shp-image-size`. As the image building process is part of the build strategy, only the build strategy author can have the knowledge about where to get those values from and subsequently to write them to files. The build strategy author will access the path for the result files using Tekton's replacement variables for results, in this case: `$(results.shp-image-digest.path)` and `$(results.shp-image-size.path)`. This will be documented in our build strategy documentation. A build strategy author is not required to provide a value for those results. In case values are not provided, they will not be present in the TaskRun status.

In our sample build strategies, we should support these two results as much as possible:

- `ko` supports to write an OCI image manifest file using the `--oci-layout-path` argument. The strategy needs to write that manifest to a temporary location, use a tool to extract the digest and size from the index.json file, and write this data to the result files.
- Kaniko supports to write an OCI image manifest file using the `--oci-layout-path` argument. Kaniko is based on scratch, as such, we cannot extend the existing build-and-push step to use some tool and run some extraction logic. We need to add an extra step. For this, we need to add volumeMounts to write the temporary file in the `build-and-push` step, and consume it in the following one to extract and store the results.
- The strategy that uses s2i in combination with Kaniko needs to adopt the solution for Kaniko.
- The Buildpacks strategies can use the `--report` argument for the [create](https://buildpacks.io/docs/concepts/components/lifecycle/create/) command to write a toml file. From there, the value can be extracted using shell logic like in the [Tekton catalog](https://github.com/tektoncd/catalog/blob/main/task/buildpacks/0.3/buildpacks.yaml#L153). Though, I prefer to use the same step to save resources. As far as I remember, only the digest is available but no size.
- Buildah TBD `buildah inspect` ?
- BuildKit writes the images digest to stdout only atm. There is a long-standing issue, [Add some way to get the sha256 digest of the image #1158](https://github.com/moby/buildkit/issues/1158), but no resolution yet. As alternative, one can use the [`crane digest`](https://github.com/google/go-containerregistry/blob/main/cmd/crane/doc/crane_digest.md) command to retrieve the digest after the image was pushed. This can run in a separate step using the `gcr.io/go-containerregistry/crane` image.

For the runtime image support, we need to adopt the solution that is described above for Kaniko.

#### 4. Own containers that perform step logic

With the git-clone not anymore happening in a Tekton-owned container, we need to provide our own step=container as part of the Task spec that performs this operation. Injecting our own steps is not new. We do this already today for the runtime image support, and also for the HTTP sources for the remote artifacts support. So far, we are just using existing container images and use shell commands like `wget` to download sources. For a more robust behavior to load sources, I propose to develop our own binaries that we package in our own container images and use them for the steps the we inject into a TaskRun.

For a Git resource, the step that we create in the TaskSpec then looks like the following. Every occurrence of `default` comes from the name of the item in `spec.sources`, or `default` for `spec.source`:

```yaml
[...]
taskSpec:
  steps:
    - name: source-default
      image: some-registry/shipwright-io/git-source@sha:08154711
      args:
        - --url=https://github.com/shipwright-io/sample-java                     # from the user's Build
        - --revision=develop                                                     # from the user's Build, omitted if not specified which will load the default branch
        - --target=$(params.shp-source-root)                                     # resolves to /workspace/source
        - --result-file-commit-sha=$(results.shp-source-default-commit-sha.path)
      resources: [...]
    [...]
```

For a private repository, we will explicitly mount the secret and not rely on Tekton's credentials initialization (= we will also stop adding this secret to the service account; and will not anymore require a tekton annotation in this secret):

```yaml
[...]
taskSpec:
  steps:
    - name: source-default
      image: some-registry/shipwright-io/git-source@sha:19237192
      args:
        - --url=git@github.com:shipwright-io/sample-nodejs-private.git           # from the user's Build
        - --revision=develop                                                     # from the user's Build, omitted if not specified which will load the default branch
        - --target=$(params.shp-source-root)                                     # resolves to /workspace/source
        - --result-file-commit-sha=$(results.shp-source-default-commit-sha.path)
        - --secret-path=/workspace/source-secrets/default
      resources: [...]
      volumeMounts:
        - mountPath: /workspace/source-secrets/default
          name: ssh-secret                                                       # from the user's build
          readOnly: true
  volumes:
    - name: ssh-secret                                                           # name of the secret
      secret:
        secretName: ssh-secret                                                   # from the user's build
        defaultMode: 256                                                         # this is 0400 in octal
    [...]
```

The image URL is a new configuration option of the build controller deployment (similar to how Tekton allows to specify its supporting images). In our release process, we need to make sure we point to the right image in our yaml.

The resources of the step are also configurable at the build controller deployment level with reasonable defaults.

The way we run the download logic is something we can implement in a staged approach and start with the simple approach: Just a container with the necessary command-line tools (like `git`), and a simple executable implemented by us that parses the arguments and calls the `git` binary similar to [today's logic in Tekton](https://github.com/tektoncd/pipeline/blob/v0.21.0/pkg/git/git.go).

Eventually, I propose we evolve our code to use libraries like [go-git](https://github.com/go-git/go-git) to perform the logic. This enables us to improve our error handling (and retry certain network errors) and reporting as this usually can be done better when using libraries compared to when calling other executables. On the other hand, we need to check if go-git has maybe limitations (for example missing lfs support) that forces us to stay on the approach to call the `git` binary.

For the executables that we build, I propose we use go as programming language. Similar as Tekton does it, I propose to put its code in the shipwright-io/build repository. This simplifies the build process as the digest of these supporting images need to become the values of configuration options of our build controller deployment. We will need to see how well `ko` helps us here as for those images there is no podspec with an image property that `ko` needs to build. Sharing one git project allows to re-use common code without the need to import packages across code repositories, for example for exit code constants that the container uses to report certain error situations and that our controller will use to translate them to machine-readable reasons and user-readable messages.

The container images that we build must be multi-platform images.

### Risks and Mitigations

The complexity and the relationship with other enhancement proposals like [Add EP on Build Strategies Parametrization #697](https://github.com/shipwright-io/build/pull/697) where we add build strategy author defined parameters that - name-wise - can conflict with system parameters; with ongoing pull requests like [Remote Artifacts](https://github.com/shipwright-io/build/pull/616) that introduces the first source kind (HTTP) that we download in our own step rather than using a Tekton resource; and with future items like [Remove support for the pipeline service account #680](https://github.com/shipwright-io/build/issues/680) that I intentionally partially covered for the source secret already as I do not think it makes much sense to implement our own logic relying on something that we know will go away.

To address the complexity, I suggest that the implementation is done in a staged approach with smaller but consistent changes (the numbering sometimes implies dependencies, but some things are also independent and can be worked on in parallel or in a different order):

1. Introduce system parameter for the output image (`shp-output-image`), remove the image resource, use the new system parameter in the sample build strategies to access the designated location of the output image. Comment out those arguments in the Kaniko and ko strategies that write the image digest file.
2. Implement a container image that accepts the arguments for the Git operation and runs the `git clone`, either already using [go-git](https://github.com/go-git/go-git) if that is easy to achieve, or using Tekton's approach to call the `git` binary. Tests for the binary must be implemented as well for public and private repositories.
3. Introduce system parameters that point to the source directory (`shp-source-root` and `shp-source-context`) and use them in our sample build strategies as a replacement of the hardcoded `/workspace/source` paths.
4. Introduce the `shipwright` workspace. Replace the Git resource with our own custom step that consumes the container image from (2). Stop adding the Git secret to the service account and mount it to this step directly.
5. Add a result for the commit sha and fill it from our Git executable.
6. Add a result for the image digest and extend the sample build strategies to write to this result.
7. Improve the implementation for HTTP sources to make it consistent with this design.
8. Define and implement a well-defined set of exit codes per resource kind and use them in the BuildRun controller for an enhanced error reporting.

The goal to run as non-root is not covered in great detail in this proposal as some items are not 100 % clear to me yet, for example:

- Will we need to define a `podTemplate` in the Task spec to set a global `securityContext` to run also Tekton init containers as a non-root user?
- Is for example the `/tekton/results` directory already writable for every user?
- Can we decide what the id of the non-root user is or will different build strategies need to use different users?
- Do we need to give the build strategy author flexibility as which user the init containers and our source downloads run and subsequently as which user the source files are stored on disk?

Trying to mitigate it by listing it here to gather feedback/ideas/suggestions on this area.

## Design Details

### Test Plan

- Unit tests for the new code that runs the git clone.
- Integration tests for the executable that performs the git clone.
- Build strategies in the catalog are changed similarly as the sample build strategies.
- Different error situations (like non-existing code repository, wrong SSH key for a private repository, etc.) are included in the integration tests once we have improved error handling.

### Documentation

Documentation changes from the implementation for this EP will be mainly in the documentation of build strategies. The [current documentation for build strategies](../buildstrategies.md) mainly focuses on our samples, and a few aspects like resources and annotations. For other aspects like the access of source code we rely on the author of a build strategy to find out that `/workspace/source` needs to be used. This will be addressed by documenting the system parameters introduced by this EP and our samples using them. We should be better than the Tekton Catalog in that area where I sometimes stumble over samples that use internal paths (like `/tekton/results/<RESULT_NAME>` instead of `$(results.<RESULT_NAME>.path)`).

### Graduation Criteria

I am not planning that there will be an option to choose between the old Tekton resources approach, and the new implementation. As such, there will not be dev previews or similar.

### Upgrade / Downgrade Strategy

This proposal does not include any changes to our CRDs, but it will contain breaking changes that build strategy authors need to take care of:

- Sources will not anymore be in `/workspace/source` but elsewhere. The build strategy author must use the parameters to access this location.
- The output image will not anymore be a resource. The build strategy author must use parameters to access the location.
- With the output image resource be gone, the directory `/workspace/output/image` will not exist anymore. Build strategies authors must adopt a different approach to report the image digest.

### Version Skew Strategy

N/A

## Implementation History

## Drawbacks

## Alternatives

Given Tekton resources will not graduate to v1, they will eventually disappear. Not doing anything is not an alternative. Implementation alternatives are atm inside the [proposal](#proposal) section for the one or other aspect. Once we make decisions, we can clean that up and move things here.

---

To run our own logic to download the sources, instead of having one step per source, an alternative is to have a single step that handles all sources with advantages for both options:

| Single container | Multiple containers |
| -- | -- |
| (+) Clear separation and easier to implement, for example to make the result deterministic when some source overwrites files from another one<br>(+) Error reporting using exit code is easy to map back to the specific source that failed to load<br>(+) Different containers can use different languages to perform the operation (theoretically, practically, everything will likely be go) | (+) Performance is likely better as downloads can happen in parallel<br>(+) smaller pod effective resources<br>(+) less container images that need to be maintained |

[We decided](https://github.com/shipwright-io/build/pull/727#discussion_r615828224) to go with one step per source. Rational is simplicity and also the assumption that the number of sources will in most cases be just one. Sometimes two or few, but we do not envision Builds to have 20 sources where the number of containers (that all need to fit on the same node as they are all in one pod) would become a problem. We may want to document this behavior in the Build documentation = encouraging the Build author to limit to only a few sources.

---

Instead of a single parameter for the output image (`shp-output-image`), one could have split this into different parameters to separate the tag into an own parameter (`shp-output-image-tag`). This would have helped strategies that need this value separately. [We decided](https://github.com/shipwright-io/build/pull/727#discussion_r615930419) against this for simplicity and because ko is the only tool today that needs this (and parses the image URL to split it). It is possible to revisit this in the future and to provide additional parameters if needed.

## Infrastructure Needed [optional]

Nothing
