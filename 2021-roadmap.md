<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->
# **Shipwright 2021 Roadmap**

**Mission Statement:** Shipwright aims to do one thing well: build images from source. Automatically, securely, reliably.

Features that further that goal should be welcomed, and prioritized to optimize that goal. Features that don't should be improved and removed if necessary.

Shipwright should be simple to use and flexible to extend. Once set up, it should run smoothly for users and operators alike, and when failures happen (they always do) they should be easy to diagnose and fix.


1. **Build ← you are here!**
    *   Pluggable BuildStrategies, with pre-defined support for Buildpacks, Kaniko, S2i, Buildah; users can modify, install their own
    *   Continue to improve built-in BuildStrategies for security best practices ([#169](https://github.com/shipwright-io/build/issues/169))
2. **API Improvements / Simplification**
    *   The API should be small, powerful and easy to use
    *   Synchronous webhook validation ([#685](https://github.com/shipwright-io/build/pull/685))
        *   Deprecate and remove BuildStatus and BuildStrategyStatus
    *   BuildRun status should include resolved git commit ([#205](https://github.com/shipwright-io/build/issues/205)) and built image digest ([#618](https://github.com/shipwright-io/build/issues/618))
    *   BuildStrategy Parameterization ([#697](https://github.com/shipwright-io/build/pull/697))
        *   Move builder + dockerfile fields to strategy-specific params
    *   Don’t rely on `creds-init` or PipelineResources ([#696](https://github.com/shipwright-io/build/issues/696))
        *   These features not likely to be extended to Tekton v1
        *   Remove auto-generating SAs at runtime ([#679](https://github.com/shipwright-io/build/issues/679))
    *   [API Review](https://github.com/shipwright-io/build/issues/516) + API Hygiene
        *   Remove deprecated BuildRun status fields ([#517](https://github.com/shipwright-io/build/issues/517))
        *   **Note:** Breaking API changes are _allowed_ in alpha, and should be encouraged where they improve the experience, but must be carried out responsibly.
    *   Remove or scope down `runtime-image` support
    *   Multiple inputs / volumes ([EP](https://github.com/shipwright-io/build/blob/master/docs/proposals/remote-artifacts.md))
    *   Some feature like [`runPolicy`](https://docs.openshift.com/container-platform/4.7/cicd/builds/advanced-build-operations.html#builds-build-run-policy_advanced-build-operations): `Serial` / `Parallel` / `SerialLatestOnly`
    *   **Note:** The above is not intended to be an exhaustive list of features/bugs we intend to close in 2021
3. **Project Hygiene + Health**
    *   The project should run well, making new contributions easy
    *   Docs / website, blogs, tutorials ([#700](https://github.com/shipwright-io/build/issues/700)), conference talks
    *   [CI / flakebusting](https://github.com/shipwright-io/build/issues/653) -- stamping out flakes improves contributor experience, accelerates project growth
4. **Community Governance**
    *   Donate project and "Shipwright" trademark to a foundation
    *   Establish vendor-neutral governance model
5. **Triggering**
    *   "Put both C’s in CI/CD" -- Builds should be able to create BuildRuns automatically
    *   Watch a repo (or image, or other things) and create new BuildRuns
6. **CLI**
    *   Using Shipwright shouldn't require `kubectl`
    *   The CLI should make logs easier, especially, to make debugging failed builds better.
    *   CLI to create/watch BuildRuns helps with integrations with other tooling (Tekton, Jenkins, etc.)
    *   CLI should support some way to build on-cluster from local source -- this might require API changes ([IBM demoed in February](https://github.com/shipwright-io/build/issues/551#issuecomment-771002913))
7. **Operator / Operations**
    *   Installing and managing a Shipwright installation should be easy
        *   Upgrade installation
        *   Upgrade installed BuildStrategies
    *   Configure Shipwright with a `ConfigMap` instead of controller flags/envs ([#651](https://github.com/shipwright-io/build/issues/651))
    *   Metrics / Observability
    *   Scaling / Load Testing -- Shipwright should not be a bottleneck
8. **Rebase** ([#656](https://github.com/shipwright-io/build/issues/656))
    *   Quickly and efficiently picking up security fixes can be as important as producing images from new source
    *   w/ Triggering, enables watching a base image, producing rebased images quickly → CVE-to-deployed-fix time should be as short as possible
9. **Integrations**
    *   Shipwright exists among a community of projects; we should work well together with others
    *   Document + demo running a Build inside a Tekton Pipeline using Custom Task
    *   Document + demo triggering rollouts after Shipwright produces an image
    *   Document + blog about niche build strategies (e.g., Bazel?)
