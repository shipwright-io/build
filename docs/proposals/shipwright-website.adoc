<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: shipwright-website
authors:
  - "@adambkaplan"
reviewers:
  - "@otaviof"
  - "@SaschaSchwarze0"
approvers:
  - "@qu1queee"
  - "@sbose78"
creation-date: 2020-09-14
last-updated: 2020-09-14
status: implementable
---

# Shipwright Website

**Build Enhancement Proposals have been moved into the Shipwright [Community](https://github.com/shipwright-io/community) repository. This document holds an obsolete Enhancement Proposal, please refer to the up-to-date [SHIP](https://github.com/shipwright-io/community/blob/main/ships/0007-shipwright-website.md) for more information.**

## Release Signoff Checklist

- [x] Enhancement is `implementable`
- [x] Design details are appropriately documented from clear requirements
- [ ] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Summary

Create a website to host documentation, release notes, news, and blog posts.

## Motivation

The `build` repository is not sufficient to evangelize the project. Running a separate website with
documentation, release notes, and how-tos is a preferable format for users to consume information.

### Goals

- Lay out a framework for managing website content.
- Establish a process for individuals to contribute content.

### Non-Goals

- Create exhaustive documentation
- Author blog posts/tutorials

## Proposal

Shipwright can take advantage of [GitHub Pages](https://docs.github.com/en/github/working-with-github-pages/getting-started-with-github-pages)
and the [Hugo templating engine](https://gohugo.io/) to generate a static website. Hugo is a
popular framework to manage content, and is used by several communities (including upstream
[Kubernetes](https://github.com/kubernetes/website)). The site will consist of two GitHub
repositories:

1. A `website` repo with the Hugo assets, template, and content in Markdown format
2. The `shipwright-io.github.io` repository, our org's GitHub Pages repo.

The `website` repository will contain the theme and the GitHub Pages repositories as submodules.
A deployment script in `website` will update the site content on pull request merges.
This can be automated via Travis CI.

[Docsy](https://www.docsy.dev/) will be used as the baseline theme, which is optimized for software
projects. This is the base theme used by upstream Kubernetes.

### Implementation Details/Notes/Constraints

GitHub pages has support for [custom domains](https://docs.github.com/en/github/working-with-github-pages/configuring-a-custom-domain-for-your-github-pages-site).
With appropriate `CNAME` and DNS records, the GitHub Pages site can be the host for `shipwright.io`
content. GitHub Pages can also [enforce HTTPS](https://docs.github.com/en/github/working-with-github-pages/securing-your-github-pages-site-with-https).

Since the website is content (not software), the
[Creative Commons Attribution 4.0 International](https://creativecommons.org/licenses/by/4.0/legalcode)
license is appropriate for the `website` repo and generated content on `shipwright.io`. The latter
can identify licensing via a custom footer.

The Docsy theme supports internationalization with appropriate configuration in place. We should
use this to support content translations in the future.

### Risks and Mitigations

**Risk**: Unsanctioned content is published.

_Mitigation_: The main GitHub Pages site will enforce branch protection on the default branch. Only
admins and sanctioned robot accounts will be allowed to merge to the default branch.

## Design Details

The following sections are proposed for content:

1. Documentation (`/docs`)
2. Blog (`/blog`)
   1. News (`/news`)
   2. Release Notes (`/releases`)

## Implementation History

- 2020-09-14: Proposal

## Drawbacks

- Docs are separated from the repositories where code is written.
- Release note content may be duplicated.

## Alternatives

### Host docs and blogs alongside code

Only one repository has active development at present (`build`). Documentation can be hosted as
Markdown in the existing `docs` folder. Release notes can be hosted directly on GitHub without an
organization page.

Blogs and tutorials can be hosted as Markdown in `docs`, or can be published on sites run by
project maintainers.

## Infrastructure Needed

1. GitHub repositories (`website` and `shipwright-io.github.io`)
2. GitHub robot account to push changes from `website` onto `shipwright-io.github.io`.
3. Appropriate branch protection on `website` and `shipwright-io.github.io`

## Open Questions [optional]
