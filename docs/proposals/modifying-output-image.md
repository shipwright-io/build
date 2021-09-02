<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: modifying-output-image
authors:
  - "@ImJasonH"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2021-04-14
last-updated: 2021-04-14
status: provisional
---

# Modifying Output Image

**Build Enhancement Proposals have been moved into the Shipwright [Community](https://github.com/shipwright-io/community) repository. This document holds an obsolete Enhancement Proposal, please refer to the up-to-date [SHIP](https://github.com/shipwright-io/community/blob/main/ships/0013-modifying-output-image.md) for more information.**

## Release Signoff Checklist

- [ ] Enhancement is `implementable`
- [X] Design details are appropriately documented from clear requirements
- [X] Test plan is defined
- [X] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Open Questions

1. Should this feature also allow users to _unset_ existing labels/annotations inherited from base images?

## Summary

Allow Build authors to describe modifications they'd like to make to output images.
For this proposal, we'll scope this down to staticly-defined image [labels](https://docs.docker.com/engine/reference/builder/#label) and [OCI annotations](https://github.com/opencontainers/image-spec/blob/master/annotations.md).

Future proposals might allow Build authors to configure image tags, and/or to allow authors to configure dynamic values based on input values.
These are non-goals of this proposal.

## Motivation

### Goals

- Begin to enable simple modifications of the output image by Build authors.

### Non-Goals

- Enable dynamic values or placeholders/templating (e.g., git commit SHA)
- Automatically inject labels or annotations without user input
- Enable modification of image layer contents (see [runtime image](https://github.com/shipwright-io/build/blob/master/docs/proposals/runtime-image.md))

## Proposal

Add new fields to describe output image labels and annotations:

```yaml=
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: my-build-name
spec:
  output:
    image: quay.io/my/image
    credentials:
      name: my-push-creds
    # New fields
    labels:
      maintainer: team@my-company.com
      description: "This is my cool image"
    annotations:
      org.opencontainers.image.url: https://my-company.com/images
      org.opencontainers.image.source: https://github.com/org/repo
```

This will be enabled by embedding the existing `Image` struct in a new struct that 

```go=
type BuildSpec struct {
    ...
    Output OutputImage `json:"output"`
}

type OutputImage struct {
    Image // embedding existing type
    
    // New fields
    Labels      map[string]string `json:"labels,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty"`
}
```

### Implementation Details/Notes/Constraints

When a BuildRun executes for a Build that specifies `labels` or `annotations`, the BuildRun controller will append a new step to the end of the Tekton TaskRun that augments the image with the specified labels and annotations, and pushes to the same tag.

This could be accomplished using a containerized CLI tool like [`crane`](https://github.com/google/go-containerregistry/blob/main/cmd/crane), or a purpose-built binary that Shipwright provides and packages in our releases.

In either case, the TaskRun produced by the BuildRun controller would include a step along the lines of:

```yaml
- name: modify-output-image
  image: gcr.io/go-containerregistry/crane  # TODO: pin this to a digest, configurable by Shipwright operators
  script: |
    crane mutate $(params.output-image) \
      --add-label maintaner="team@my-company.com" \
      --add-label description="This is my cool image" \
      --add-annotation org.opencontainers.image.url=https://my-company.com/images \
      --add-annotation org.opencontainers.image.source=https://github.com/org/repo \
```

_(This is intended as an illustration)_

### Risks and Mitigations

**Risk:** Automation that watches for image changes by tag will see two pushes as a result of the implementation above; one for the original push done by the specified BuildStrategy, and another immediately after when the next step modifies the image to set labels/annotations.

**Mitigation:** We can document this behavior as expected.

**Future Improvements:**
As a future improvement, we can make the Build's specified labels and annotations available as parameters to BuildStrategy authors, so they can pass labels/annotations to their build process directly.
In this case the appended step will perform a no-op push, which will not trigger a subsequent image change notification.

Another possible future improvement would be to support and document a local filesystem path that build strategies can write image data to instead of pushing it to the remote registry.
If the BuildRun sees image data at that location, it could perform some modifications before pushing the image itself.
This would require support from build tooling to support writing to a local file path instead of pushing, and for build strategy authors to take advantage of this support.

## Design Details

### Test Plan

And e2e test case will check that a Build that specifies labels and annotations, when run, produces an image with those values set.

## Drawbacks

It's not ideal that the implementation produces two image pushes.

## Alternatives

We could jump directly to the first **Future Improvement** listed above, and pass a Build's specified labels and annotations to the BuildStrategy as parameters.
In this case we could also just expose image labels and annotations as optional BuildStrategy parameters only for those BuildStrategies that support them.
This depends on mature support for BuildStrategy parameterization (https://github.com/shipwright-io/build/pull/697).

## Infrastructure Needed

None.
