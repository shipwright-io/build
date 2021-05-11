<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: Expressing Execution Environment Variables in Shipwright API
authors:
  - "@gabemontero"

reviewers:
  - "@sbose78"
  - "@SaschaSchwarze0"
  - "@otaviof"
  - "@adambkaplan"
<!-- 
Note from Gabe: I went light (i.e. just Shoubhik since he asked for this EP) on reviewers as a first pass,
only because it implied a mandate / expectation.  I've chosen to update this list and folks actually review.
-->

approvers:
  - "@sbose78"
  - "@SaschaSchwarze0"
  - "@otaviof"
  - "@adambkaplan"

<!-- 
Note from Gabe: I went light (i.e. just Shoubhik since he asked for this EP) on approvers as a first pass,
only because it implied a mandate / expectation.  I've chosen to update this list and folks with approve permissions actually review,
where I'll certainly make sure we get multi company sign off.
-->

creation-date: 2021-04-09

last-updated: 2021-05-11

status: implementable    

<!-- status: provisional|implementable|implemented|deferred|rejected|withdrawn|replaced -->
see-also:
  - [Parameterize Build Strategies](https://github.com/shipwright-io/build/pull/697/files)
---

# Expressing Environment Variables in Shipwright API


## Release Signoff Checklist

- [ / ] Enhancement is `implementable`
- [ / ] Design details are appropriately documented from clear requirements
- [ / ] Test plan is defined
- [ / ] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Open Questions [optional]

> 1. API change

Assuming this gets delivered while Shipwright builds is still v1alpha1, we just have to announce to users
what new API is provided.

But if the implementation of this is delayed to the point that we are at say v1beta1, then we have to 
consider the round trip experience when a client is at a different version of the server as discussed in
[Kubernetes Deprecation Policy Rule #2](https://kubernetes.io/docs/reference/using-api/deprecation-policy/).
And yes, adding an API is not the same as deprecating one, but the round trip guarantee applies (as Tekton
learnt recently).  The use of an annotation that captures version differences comes into play.

However, sentiment was conveyed during the review process that we should not ship a v1beta1 API without this
enhancement proposal getting implemented.

So needing this for v1beta1 is what is currently conveyed in the "Graduation Criteria" section.  If available 
development cycles for this proposal allow that to happen, great.  But if this work is pushed out, we need to
track this item during grooming as we execute on the overall Shipwright roadmap. 

> 2. Validation

Existing `pkg/validate/validate.go` logic, and augmenting that for the validations noted below, vs. employment of 
a validating admission webhook.  Validation at the generated CRD level via OpenAPI was also noted as a means of ensuring
that only the allowed or blocked environment variable fields are set on a `BuildStrategy`.  At this time, this EP
only lists the spots for validations vs. declaring which should be done where.  Decisions on which approaches are 
employed, and for which elements of the new API they are employed, is currently deferred to implementation time. 

## Summary

One of the fields in the core Kubernetes API, Pods and their Containers, are Environment Variables.  Of course those
translate to operating system level environment variables being set when running the same commands executed
in Pod Containers from the command line of their personal computuer.

The next major dependency for Shipwright, Tekton, has a vast enough array of scenarios it addresses that it provides a 
direct line to the Kubernetes Container API in their API.  As as result, environment variable fields from `Steps`/`Containers`
are directly accessible from Tekton API.

The relationship chain is as follows:

- Reconciliation of a Shipwright `BuildRun` results in the creation of a corresponding Tekton `TaskRun`
- That `TaskRun` utilizes either a reference to a `Task` or an inlined `TaskSpec`, which along with standard k8s `TypeMeta`
and `ObjectMeta` types comprises the total sum of a `Task`
- And `TaskSpec` has an array of Tekton `Step` entries, where `Step` inlines k8s `Container`

There are also a set of features in Tekton around setting values for environment variables in a dynamic fashion.

While the list of scenarios Shipwright goes after is a subset of what Tekton goes after, as the [Motivation](#motivation)
section will explain, manipulation of environment variable settings on the resulting k8s Container objects exists
for Shipwright.

How to best allow Shipwright users to manage in a first class way from our API environment variables that are ultimately set on k8s 
Containers is what this enhancement proposal will address.


## Motivation

The various image management tools Shipwright encapsulates with 'BuildStrategies' all allow for tuning of their behavior
via environment variables.  The names and supported values of those variables vary between tools, and sometimes 
between different versions of those tools.  So a lot of variability we need to account for.

The same goes for source code management tools, tools for retrieving content via socket connections, both of which are 
used in Shipwright.

So supporting our users via our API and related features to manage environment variables for those tools makes a lot of
sense.

### Goals

- Shipwright API needs to articulate a first class way for specifying those environment variables approach for both 
  administrators and individual developers (i.e. authors/owners of the BuildStrategies vs. authors owners of the Builds 
  and BuildRuns) to specify those environment variables.
- Use of the existing features Tekton provides to enrich our users' management of environment variables needs to be 
  evaluated and exploited for code reuse.
- Maintain the currently Shipwright approach of treating Tekton as an implementation detail, and try to not leak Tekton
  API into our API.

### Non-Goals

- Any needed change to integrate Shipwright with Tekton parameters is handled by the [current 'Parameterize Build Strategies' EP](https://github.com/shipwright-io/build/pull/697)
  or follow up work that stems from that EP.  Until then, only static content of any environment variable value related 
  fields is supported.  In other words, the `$(params...)` syntax seen in Tekton today.
- The EP is *NOT* about environment variables that will be set in the final image.  It is about providing environment variables
  to the Steps/Containers and making those environment variables available to the tools called from those Steps/Containers
  as part of building the image.
- This EP is *NOT* taking on the official establishment of new Shipwright specific personas and roles around the 
  maintenance of `BuildStrategy`, `Build`, and `BuildRun`.

## Proposal

First, some role / actor terminology and detail (nothing new most likely for Shipwright community members):

- At the time of this writing, there are no specific k8s `Roles` and `ClusterRoles` defined in Shipwright that control
  *ONLY* the maintenance of both clustered and namespaced `BuildStrategy`, `Build`, or `BuildRun` objects.
- In conjunction, there is no defined aggregation of such precise Shipwright `Roles` and `ClusterRoles` to the well known
  k8s roles and personas, like "cluster-admin", "admin", "edit", and "view".
- Nor does documentation exist in [the repository's doc folder](https://github.com/shipwright-io/build/tree/master/docs)
  nor in [the repository's enhancement proposal guidelines folder](https://github.com/shipwright-io/build/tree/master/docs/proposals/guidelines)
  articulating such personas for use in enhancement proposals.
- Quite possibly though some form of the above points needs to occur in the project at some point.  
  
All that said, in the interim, there is at least agreement during the review process for this proposal that:

- users who create `BuildRun` or `Build` objects quite possibly will not have permission to create either clustered or 
  namespace scoped `BuildStrategy`
- conversely, maintainers of `BuildStrategy` objects quite possibly will also have decision-making power around cluster
  wide rules controlling use of the tools used in building images
- where often such control entails establishing permissions around the use of environment variables and their specific 
  settings when those tools are invoked.
- in other words, most likely the `BuildStrategy` maintainer will have greater overall permission in the k8s cluster  
  
Now, on to the meat of the proposal.

As the Shipwright EP 'Parameterize Build Strategies' notes, [Tekton Parameters](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#specifying-parameters)
allows for expressing key/value pairs, where the value the system ultimately substitutes for the key when it is used within
Shipwright API can come from specification elsewhere.

Those Tekton parameter key/value pairs can also be used in conjunction with the Environment Variables that can be
set on `Steps`/`Containers`.

Some highlights of the Tekton features around Parameters and updating environment variables in Steps:

- [direct parameter variable substition for fields related to Step/Container environment variable fields](https://github.com/tektoncd/pipeline/blob/main/docs/variables.md#fields-that-accept-variable-substitutions)
- [a 'stepTemplate' API that allows for providing defaults for environment variables](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#specifying-a-step-template)
- [use of k8s core objects like Secrets or ConfigMaps for setting the values of environment variables](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#using-a-secret-as-an-environment-source)

All of these features seem accessible to Shipwright, in that
- Shipwright generates [a TaskRun and TaskStep](https://github.com/tektoncd/pipeline/blob/v0.23.0/pkg/apis/pipeline/v1beta1/taskrun_types.go#L55) 
  from a [Build and BuildRun](https://github.com/shipwright-io/build/blob/v0.4.0/pkg/reconciler/buildrun/resources/taskrun.go#L175-L199)
- Tekton's TaskRun reconciler performs the parameter based variable substitution on the TaskRun/TaskSpec Steps/Containers 
  (the call stack is lengthy, so I won't post all the links here :-) )
- An implementation of Shipwright's [Add EP on Build Strategies Parametrization #697](https://github.com/shipwright-io/build/pull/697) or 
  follow on that allows specification and mapping from Shipwright API to then perform the appropriate `TaskSpec` to `Step` [manipulation](https://github.com/shipwright-io/build/blob/v0.4.0/pkg/reconciler/buildrun/resources/taskrun.go#L129-L141).

Independent of fully integrating Shipwright with Tekton Parameterization, a first class API for environment variables
can be devised and implemented.  Parameter variable substitution support can land afterward without any further API change.
We can have just static settings of environment variable values with an initial implementation of this proposal, or we 
can also claim support for use of parameterization, based on when the implementations of the different proposals land.

Next, if we look at the k8s [Container API's environment variable fields](https://github.com/kubernetes/api/blob/v0.21.0/core/v1/types.go#L2252-L2265)

```go
	// List of sources to populate environment variables in the container.
	// The keys defined within a source must be a C_IDENTIFIER. All invalid keys
	// will be reported as an event when the container is starting. When a key exists in multiple
	// sources, the value associated with the last source will take precedence.
	// Values defined by an Env with a duplicate key will take precedence.
	// Cannot be updated.
	// +optional
	EnvFrom []EnvFromSource `json:"envFrom,omitempty" protobuf:"bytes,19,rep,name=envFrom"`
	// List of environment variables to set in the container.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Env []EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,7,rep,name=env"`
```

First decision:  do we need to expose both `EnvFrom` and `EnvVar` off of `Containers`, especially given that
`EnvVar` has an optional `EnvFromSource` [ref](https://github.com/kubernetes/api/blob/v0.21.0/core/v1/types.go#L1929)

```go
// EnvVar represents an environment variable present in a Container.
type EnvVar struct {
	// Name of the environment variable. Must be a C_IDENTIFIER.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Optional: no more than one of the following may be specified.

	// Variable references $(VAR_NAME) are expanded
	// using the previous defined environment variables in the container and
	// any service environment variables. If a variable cannot be resolved,
	// the reference in the input string will be unchanged. The $(VAR_NAME)
	// syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped
	// references will never be expanded, regardless of whether the variable
	// exists or not.
	// Defaults to "".
	// +optional
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	// Source for the environment variable's value. Cannot be used if value is not empty.
	// +optional
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty" protobuf:"bytes,3,opt,name=valueFrom"`
}
```

This proposal is making the simplifying assumption that ultimately setting the `Env []EnvVar` field in the 
underlying Tekton `Step`/`Container` is sufficient for an initial implementation and scenarios we want to support.
That will help the initial implementation in that the explicit "name" index field helps with default kubernetes 
"path" API operations that tend to occur with devops/gitops flows.  Also, `ValueFrom *EnvVarSource` still allows
us to pull content from `Secrets`, like authentication token.  Many of the tools when building images (including
source code extraction and tools that have to download dependencies) consume authentication tokens via environment
variables.  Storing those token in k8s `Secrets` vs. having to specify them in scripts referenced in `Containers` has 
become a preferred method when building images in k8s native environments.

Next, the question of whether to use the k8s `EnvVar` directly exists.  This proposal suggests creating a new
Shipwright type that inlines the k8s type, in case we want to extend and add new features later that require
additional fields or new types from k8s or Tekton.  So with that, we start with

```go
type EnvironmentVariable struct {
   corev1.EnvVar
   
}
```

This field will be added to the `Build` and `BuildRun` objects.

NOTE: `valueFrom` in k8s has a range of options, from `Secret` refs, `ConfigMap` refs, field refs, etc.  The `Secret` ref
is the one we see immediate need for, but the EP does *NOT* proscribe restrictions at this time.  Of course, as we progress
toward implementation and customer feedback, if need be, restrictions can be added at a later date.

Then, let's examine the updates for `BuildStrategy` (both cluster and namespaced scope), `Build`, and `BuildRun` and where we
might update them with an array of `EnvironmentVariable`

`BuildStrategy` needs some further consideration, in that `BuildStep` already inlines `Container`, which allows specification 
of `EnvVar`.

This proposal wants to continue to let environment variables be set at the `BuildStep` / `Container` level.

However, we also want to allow the `BuildStrategy` maintainer to declare whether the value for an environment variable's
key / value pair can be changed by the `Build` or `BuildRun`.

To facilitate, the `BuildStep` type will be added to as follows:

```go
type BuildStep struct {
   corev1.Container `json:",inline"`
   EnvVarOverwritable   []string
}
```

If an environment variable key is in the `EnvVarOverwritbale` array, the value can be changed at the `Build` or `BuildRun`
level.  If the key is not present, its value cannot be changed at the `Build` or `BuildRun`.  So the default is "not overwritable".

Lastly, so there are two basic mindsets from an admin perspective:

- Allow any environment variable in general, but restrict a few
- Disallow or block any environment variable in general, but allow a few

To facilitate both mindsets, we'll add two string arrays to the `BuildStrategySpec`
- `AllowedEnvironmentVariables`
- `BlockedEnvironmentVariables`

Each string array list the `EnvVar` names/keys to consider.
 
If `AllowedEnvironmentVariables` is empty, assume all environment variable overrides are allowed, except for those mentioned in `BlockedEnvironmentVariables`.  
Otherwise, the list is the list, and any environment variables in `BuildStep`, `Build`, and `BuildRun` must be vetted against this list.  

`BlockedEnvironmentVariables` is redundant / ignored if `AllowedEnvironmentVariables` is set.

Either CRD/OpenAPI validation, an admission webhook, or a combination of the two, seems sufficient for enforcing that 
the `AllowedEnvironmentVariables` and `BlockedEnvironmentVariables` lists follow these rules:

- making sure only entries in `AllowedEnvironmentVariables` exist in any `BuildStep` on the `BuildStrategy` when it is created
- making sure only entries in `AllowedEnvironmentVariables` exist in the new environment variable fields proscribed by
  this EP for the `Build` and `BuildRun` types.
- making sure no entries in `BlockedEnvironmentVariables` exist in any `BuildStep` on the `BuildStrategy` when it is created
- making sure no entries in `BlockedEnvironmentVariables` exist in the new environment variable fields proscribed by 
this EP for the `Build` and `BuildRun` types.

There is also currently the `pkg/validate/validate.go` code which has some `Build`
validations.  The interface proscribed there, along with the existing Strategy validation, could be augmented to ensure the 
conditions noted above.  Expansion to consider `BuildRun` in that validation flow would be required.

An open question at the top of this EP has been added for reaching a consensus around which validation path(s) to employ.

Each entry of the allow and blocked arrays need to minimally support wildcards, if not regular expressions.
This EP will leave the decision on exactly which of those choices are employed to the implementation, based on how time
constraints shake out.

Updates to `BuildSpec` and `BuildRunSpec` to allow for an array of `EnvironmentVariable` is then straight forward.

With this combination of `BuildStrategySpec` (both a new `EnvironmentVariable` array and new string array called `BlockedEnvironmentVariables` ),
`BuildStep` (and the existing inlining of `Container`), and `EnvironmentVariable` arrays added to `BuildSpec` and 
`BuildRunSpec`, we get an order of precedence that allows both "cluster admin centric" control over environment variables, but 
opt in flexibility for "developer centric" overrides, on an `EnvironmentVariable` key by key basis.

The order of precedence is:

- `AllowedEnvironmentVariables` in the `BuildStrategySpec` is considered first.  If it is not empty, and any subsequent environment variable is not in that list, it is a validation error. 
- `BlockedEnvironmentVariables` in the `BuildStrategySpec` is considered first.  Any use of those keys in 
  `BuildStep`, `BuildSpec.[]EnvironmentVariable`, `BuildRunSpec.[]EnvironmentVariable` results in a validation error.
- `BuildStep` from the `BuildStrategySpec` is next.  The key/value pair from these applied
- If `BuildSpec.[]EnvironmentVariable` has entries, and the keys are in `BuildStep.EnvVarOverwritable`, those are applied, otherwise a validation error.
- Then in `BuildStrategySpec.[]EnvironmentVariable` has entries, and the keys are in `BuildStep.EnvVarOverwriteable`, those are applied, otherwise a validation error.
- Any use of Tekton Parameters are reconciled by the Tekton `TaskRun` Reconciler

Lastly, some yaml to complete the visualization.

`BuildStrategy` with a disallow entry, a spec level default, and a final, always set via a `BuildStep`:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: something-that-uses-docker
spec:
  # nobody is allowed to change DOCKER_TLS_VERIFY
  BlockedEnvironmentVariables:
    - 'DOCKER_TLS_VERIFY'
  buildSteps:
    - command:
        - /usr/local/bin/docker
        - build
      env:
        - name: DOCKER_API_VERSION
          value: '1.19'
        - name: DOCKER_CONFIG
          value: /tekton/home/.docker

```

`Build` with an override of `DOCKER_API_VERSION`:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: something-that-uses-docker-build
spec:
  source:
    url: https://github.com/myorg/myrepo
  strategy:
    name: something-that-uses-docker
    kind: ClusterBuildStrategy
  environmentVariables:
    - name: DOCKER_API_VERION
      value: '1.20'
```

`BuildRun` with an override of `DOCKER_API_VERSION`:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: BuildRun
metadata:
  name: something-that-uses-docker-buildrun
spec:
  environmentVariables:
    - name: DOCKER_API_VERSION
      value: '1.21'
  buildRef:
    name: something-that-uses-docker-build
```

Lastly, an example that combines with both grabbing environment variable values from k8s `Secrets`, where we will ultimately
map to manipulation of the `TaskSpec` within the `TaskRun`, along with the use of Tekton parameterization, whose exposure
within Shipwright is articulated in the [Parameterize Build Strategies](https://github.com/shipwright-io/build/blob/master/docs/proposals/parameterize-strategies.md).
The `TaskRun` end result works off the [Using a Secret as an environment source](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#using-a-secret-as-an-environment-source)
example described in upstream Tekton.

```yaml
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: a-cluster-strategy
spec:
  buildSteps:
    env:
      - name: GITHUB_TOKEN
        valueFrom:
          secretKeyRef:
            name: $(params.github-token-secret)
            key: bot-token
  params:
  - name: github-token-secret
    description: Name of the secret holding the github-token.
    default: github-token
```

Similar use of `params` and `environmentVariables` will exist in `Build` and `BuildRun` as well.

The resulting `TaskRun`:

```yaml
apiVersion: tekton.dev/v1beta1
kind: TaskRun
metadata:
  name: ...
spec:
  taskSpec:
    steps:
      - name: git-checkout
        image: <you image of choice that containers 'git'>
        env:
          - name: GITHUB_TOKEN
            valueFrom:
              secretKeyRef:
                name: <either a user supplied param value or the default 'github-token'>
                key: bot-token

```

### User Stories [optional]

#### Story 1

As an image developer using Shipwright, I need to be able to control environment variables passed to the underlying
tools used for image management, source code management, or data transfer, when I build image using Shipwright, assuming
agreement with the cluster administrator has been obtained on which environment variables I can control.

#### Story 2

As someone in charge of maintaining `BuildStrategies` and most likely the use of Shipwright across the cluster, 
I need to make sure that the users of my cluster do not use any unsafe environment variable enabled 
features that exist with the underlying tools used for image management, source code management, or data transfer, when
Shipwright runs in the cluster.

#### Story 3

As someone in charge of maintaining `BuildStrategies` and most likely the use of Shipwright across the cluster, 
I need to make sure that only approved values for certain environment variables, which in turn control
features that exist with the underlying tools used for image management, source code management, or data transfer, are used when
Shipwright runs in the cluster.

### Implementation Details/Notes/Constraints [optional]


### Risks and Mitigations

Coordination with the parameterization enhancement work is an added bit of complexity.

## Design Details

### Test Plan

This requires new unit and integration tests.  In conjunction, documentation needs to be updated with explanations
and examples.

### Graduation Criteria

Should be part of a milestone that ships as part of or before our v1beta1 API is declared.

### Upgrade / Downgrade Strategy

Nothing around environment variables exists yet in the API.  For changes after the initial implemenetation of 
this proposal, see the references to the kubernetes recommendations elsewhere in this proposal.

### Version Skew Strategy

If this feature does not make v1alpha1, or if we change it in later versions, then the kubernetes recommendations around the use of annotations
to deal with round trip transfer of `BuildStrategy`, `Build`, and `BuildRun` referenced in the [open questions](#open-questions-optional)
section will be needed.

## Implementation History


## Drawbacks

First, balancing between providing full function out of the gate vs. a reduced API that can be extended without disruption
provided healthy debate during the crafting of this enhancement proposal.

Second,  if Shipwright users end up preferring the use of third party policy engines for restricting, the parts of this 
API that attempt to provide controls around environment variable specification will appear redundant.  More detail on
these third party policy engines are in the [Alternatives](#alternatives) section below.

## Alternatives


### Which API to surface
Exposing Tekton API in Shipwright API is a way to surface environment variables, but to date,
Shipwright explicitly does *NOT* want to do that.  Tekton is still an "implementation detail".

### Third party admission webhook validation

With regard to allowing for changes to the order of precedence between `BuildStrategy`, `Build`, and
`BuildRun`, as well as the "unapproved" list, if we do not provide the "administrator gating" option, then administrators would have
to use third party policy engines like OPA Gatekeeper or Kyverno to prevent unwanted environment variable use,
or mutate/change values.

Use of regular expressions and wildcards for "approved" lists appears possible as well, though perhaps not quite 
as universally supported, and not quite as intuitive.  The more typical and natural examples tend to center around 
prevention of something explicit instead of allowance of a finite subset.

### Defaulting

#### Alternative 1

Also, in earlier iterations of this EP, employing possible default values for environment variables was entertained, but ultimately dismissed.

The notion revolved around coupling the default value capability available in Tekton's `StepTemplate` with reuse of the k8s
`EnvVar` type.  In other words, a default value field can be mapped to a `StepTemplate`'s env var setting.

With that, a new Shipwright type would be:

```go
type EnvironmentVariable struct {
   corev1.EnvVar
   
   // DefaultValue if set is the value applied if neither EnvVar.Value or EnvVar.ValueFrom have a value even 
   // after any Parameter variable substitution.
   // +optional
   DefaultValue  string
}
```

Based on user response, adding `DefaultValue` to our `EnvironmentVariable` wrapper of k8s `EnvVar` is plausible as a
clean, new field only, API extension

#### Alternative 2

Another iteration of this EP when into adding a `EnvironmentVariable` array for defaulting, but then using the `BuildStep` / `Container`
`envVar` array for controlling what could be overridden. This was replaced with the current approach.

### Config for order of precedence

Also, in earlier iterations of this EP, configuration for "developer centric" vs. "cluster admin / ops centric" order 
of precedence was consdered.

A proposed default would be "developer centric":
- apply `BuildStrategy` envs first
- then `Build`
- then `BuildRun`

Then, via the [current global config of controller env vars](https://github.com/shipwright-io/build/blob/v0.4.0/docs/configuration.md),
or if that current approach gets converted to a `ConfigMap`, per (https://github.com/shipwright-io/build/issues/651)[https://github.com/shipwright-io/build/issues/651], or if the community decides to create a `ConfigMap` for each
logical piece of "feature configuration", we have a setting that allows a switch to "ADMINISTRATOR_CENTRIC" order of precedence,
say:

- apply `BuildRun` envs first
- then `Build`
- then `BuildStrategy`

But at the moment, it is felt this EP can provide both on a environment variable by environment variable basis.

## Infrastructure Needed [optional]
