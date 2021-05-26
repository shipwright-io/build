<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: Parameterize Build Strategies
authors:
  - "@qu1queee"
reviewers:
  - "@SaschaSchwarze0"
  - "@ImJasonH"
  - "@sbose78"
approvers:
  - "@SaschaSchwarze0"
  - "@ImJasonH"
creation-date: 2021-03-25
last-updated: 2021-04-20
status: implemented
---

# Parameterize Build Strategies

## Release Signoff Checklist

- [x] Enhancement is `implementable`
- [ ] Design details are appropriately documented from clear requirements
- [x] Test plan is defined
- [x] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Open Questions [optional]

> 1. This will break the Build API.

When should we do this? Do we need backwards compatibility?

> 2. This will force strategy administrators to document required Parameters.

How to do this?

## Summary

Today we use a feature of Tekton call [Parameters](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#specifying-parameters) when defining a `Task` spec. This feature allow us to populate a Parameter value with the users specifications from the `Build` API. In this way, we can manipulate on each strategy step arguments (_**arg** key_).

## Motivation

There are several reasons for the need of a more well define parameterization mechanism:

- **Extensibility**: The current approach locks all strategies into three parameters, we need to allow N parameters if needed, depending on the strategy administrator requirement.

- **API enhancement**: There are some fields in the Build API, like `spec.builder.image` that could belong to a parameterize block. Reserving a parent field in the Build API for a use-case that is not present across strategies is not ideal, it just generate complexity on the API level.

- **Flexibility**: Providing a new API block on the `Build` or `BuildRun` for defining N parameters, will provide more flexibility for strategy administrators when defining new strategies. This will also simplify any potential documentation around "How to build a strategy?".

- **Transparency**: At the moment is not clear which are the default parameters across strategies, for both strategy admins and Build users. Using Tekton Params nomenclature in the strategies will make it clearly.

### Goals

- Introduce a `spec.params` field across Build and BuildRun API´s, as a way for users to provide N parameters according to their preferred strategy of choice. Defining `spec.params` in a BuildRun overrides any definition of the same param in Build´s.

- Introduce a `spec.parameters` on Strategies, both cluster or namespaced scope. This allow strategy administrators to layout the definition of N parameters, as required for their strategies.

- Revisit what we consider today as default Parameters that are defined on runtime when creating `TaskRuns`. These are `DOCKERFILE`, `CONTEXT_DIR` and `BUILDER_IMAGE`. Decide if we need to reduce or extend the default definitions and properly document their usage.

- Expose the usage of Tekton Params directly on the strategies, in favor of simplicity and readability. This boils down to `params.DOCKERFILE`, instead of `build.dockerfile`.

### Non-Goals

- Build from scratch a paremeterize mechanism, by replacing the one available in Task/TaskRuns.

- Provide support for Tekton Params of the type `array`. This does not mean we block support for this feature in the future.

## Proposal

### Part 1: Introduce spec.params

Introduce the `spec.params` field across Build and BuildRun, and the `spec.parameters` for BuildStrategies resources. This will define a list  of parameters definition, of the type `string`. This new `spec.params` does not provide support for Tekton params of the type `array`. This field can only be use, if the parameter is supported in the strategy of choice.

For example:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: a-cluster-strategy
spec:
  buildSteps: #Content omitted for this example
  parameters:
  - name: a-param
    description: A description of this parameter definition.
    default: "The default parameter value"
```

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: a-build
spec:
  source: #Content omitted for this example
  strategy:
    name: a-strategy-with-params
  output: #Content omitted for this example
  params:
  - name: a-param
    value: A parameter value.
```

```yaml
apiVersion: shipwright.io/v1alpha1
kind: BuildRun
metadata:
  name: a-buildrun
spec:
  buildRef:
    name: a-build
  params:
  - name: a-param
    value: Another parameter value because my build is not up-to-date.
```

_Note_: If a **Buildrun** specifies `a-param` via its `spec.params`, this will override the value defined in the `a-build`. In other words, BuildRuns have a higher priority when defining params.

### Part 2: Runtime Parameters

As mentioned in the Goals section, we currently define three parameters that can be constructed on runtime, also know as _runtime parameters_ or _system parameters_. System parameters are define during runtime, on the creation of a Taskrun, where the parameter definition and the parameter value mapping takes place. These runtime parameters are:

- DOCKERFILE
- CONTEXT_DIR
- BUILDER_IMAGE (_optional: It only take place if the spec.builder.image is defined_)

This EP propose to stop using `BUILDER_IMAGE` as a runtime Parameter but rather to delegate its definition to user of N strategy. This means `BUILDER_IMAGE` should be defined under `spec.params` in the future.

The list of runtime parameters will then look as follows:

- DOCKERFILE
- CONTEXT_DIR

_Note_: This should not mean we lock-in our runtime parameter, as we move on with the adoption of more tools, we might need to increase the amount of runtime parameters in the future.

Naming conventions around runtime parameters should also be considered, in a way that we can achieve unique names that will allow strategy administrators to understand that they are reserved and cannot be defined on a strategy. This should evolve into:

- `<prefix>-dockerfile`
- `<prefix>-context-dir`

where all runtime parameters should follow particular conventions, as follows:

- Runtime parameter´s name includes a prefix
- Runtime parameter´s name is written with dashes on multiple words
- Runtime parameter´s name are lower-case

### Part 3: Nomenclature changes

As referenced in [issue 694](https://github.com/shipwright-io/build/issues/694), we should remove the internal plumbing we do at runtime to map Build API definition´s in the strategies, in favor of the Tekton nomenclature. For example:

_In the Kaniko strategy `build-and-push` step args:_

Instead of doing:

```yaml
--context=/workspace/source/$(build.source.contextDir)
```

we should do:

```yaml
--context=/workspace/source/$(params.CONTEXT_DIR)
```

### Part 4: Sanity Checks

We will require a sanity check mechanism, in order to validate the quality of the defined user params. This belongs to implementation details, but generic examples are:

- Decide how to handle parameters that have none default and that are not specified at the Build/BuildRun level.
- Validate that the defined parameter in the strategy is not a reserved runtime parameter.
- Validate if the specified user params was defined in the strategy.

### Part 5: Documentation Enhancement

Currently we do not have proper documentation on:

- Tutorials on how to build Strategies.
- Which are the runtime parameters and how they are used.

This ensures that as part of this EP implementation, we can provide a set of documents to fulfill the above missing points.

### User Stories [optional]

Strategy authors will be able to define and document the usage of N parameters in their strategies. Build users will then need to define the required parameter values if they want to opt-in for the usage of certain strategies.

#### As a Build Strategy Administrator I want to parameterize a Dockerfile name, if users name their Dockerfile differently

For [`Buildkit`](https://github.com/moby/buildkit), which is another tool for doing in-cluster builds, the usage of the runtime parameters(DOCKERFILE,CONTEXT_DIR) would not be enough. While for this tool, a user can specify the name of the `Dockerfile` if this is different to "Dockerfile". For an strategy administrator, the usage of the Parameters feature will help to provide support for this specific tooling behaviour.

An strategy admin will require to define the following:

```yaml
---
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: buildkit
spec:
  parameters:
  - name: DOCKERFILE_NAME
    description: Name of your Dockerfile inside the DOCKERFILE context
    default: "Dockerfile"
  buildSteps:
  - name: build-and-push
    image: ...
    command:
      - buildctl-daemonless.sh
    args:
      - --debug
      - build
      - --progress=plain
      - --frontend=dockerfile.v0
      - --opt
      - filename=$(params.DOCKERFILE_NAME)
      - --local
      - context=/workspace/source/$(params.CONTEXT_DIR)
      - --local
      - dockerfile=/workspace/source/$(params.DOCKERFILE)
      - --output
      - type=image,name=$(build.output.image),push=true
      - --export-cache
      - type=inline
      - --import-cache
      - type=registry,ref=$(build.output.image)
```

while a user of the above strategy, will require to provide a value for `DOCKERFILE_NAME` if the default is not enough.

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildkit-build
spec:
  source: #Content omitted for this example
  strategy:
    name: buildkit
  output: #Content omitted for this example
  params:
  - name: DOCKERFILE_NAME
    value: "FoobarDockerfile"
```

#### As a Build Strategy Administrator I want to provide a human-readable description for a parameter, so that users of generic clients can get guidance on how to fill parameters; for example the Shipwright CLI can make use of this information

This ensures that we support a `description` field per parameter definition on strategies, so that users have a well-defined guidance on how to populate a parameter.

An strategy admin will require to define the following:

```yaml
---
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: buildkit
spec:
  parameters:
  - name: DOCKERFILE_NAME
    description: Name of your Dockerfile inside the DOCKERFILE context
    default: "Dockerfile"
  buildSteps:
  - name: build-and-push
    image: #Content omitted for this example
    command: #Content omitted for this example
    args: #Content omitted for this example
```

_Note_: See the usage of description under the `DOCKERFILE_NAME` parameter definition.

#### As a Shipwright Build user I want to have the flexibility to override parameters definition on my referenced Build

This allow Build users to do parameters values definition override on the BuildRun level. For example:

```yaml
---
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: buildkit
spec:
  parameters:
  - name: INSECURE_REGISTRY
    description: Defines if an image should be pushed to an insecure registry
    default: false
  buildSteps:
  - name: build-and-push
    image: ...
    command:
      - buildctl-daemonless.sh
    args:
      - ...
      - --output
      - type=image,name=$(build.output.image),push=true,registry.insecure=$(params.INSECURE_REGISTRY)
      - ...
```

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: a-build
spec:
  source: #Content omitted for this example
  strategy:
    name: buildkit
  output: #Content omitted for this example
  params:
  - name: INSECURE_REGISTRY
    value: false
```

```yaml
apiVersion: shipwright.io/v1alpha1
kind: BuildRun
metadata:
  name: a-buildrun
spec:
  buildRef:
    name: a-build
  params:
  - name: INSECURE_REGISTRY
    value: true
```

The above allows a user to opt-in for pushing to an insecure registry, although the referenced Build disables this behaviour.

### Implementation Details/Notes/Constraints [optional]

@ImJasonH already have an implementable prototype via [link](https://github.com/shipwright-io/build/compare/master...ImJasonH:params), which fits this EP.

This implementation requires the [Build API](../../pkg/apis/build/v1alpha1/build_types.go), the [BuildRun API](../../pkg/apis/build/v1alpha1/buildrun_types.go) and the [Strategy API](../../pkg/apis/build/v1alpha1/buildstrategy.go) to support the `params` field.

This implementation also seeks to remove this [logic](https://github.com/shipwright-io/build/blob/master/pkg/reconciler/buildrun/resources/taskrun.go#L35-L49) in favor of simplicity and readability, as mentioned in the GOALS section.

### Risks and Mitigations

- API breaking change when removing the `spec.builder.image`. _Note_: this feature might not be widely adopted so far.

- Strategy administrators require a way to communicate to users the parameters needed for their strategies.

- Not doing this, will lock Shipwright users and Strategy administrators to the current three runtime parameters we define today.

## Design Details

### Test Plan

- This requires new unit and integration tests.

### Graduation Criteria

Should be part of any release before our v1.0.0

### Upgrade / Downgrade Strategy

For a running Shipwright deployment, no change is required. For a running `Build` with a reference to `spec.builder.image` and X strategy, we recommend the `Build` to be recreated after the strategy X is compliant with the builder image parameter, based on the parameter name in order to specify a builder image.

### Version Skew Strategy

N/A

## Implementation History

N/A

## Drawbacks

None

## Alternatives

None at the moment.
