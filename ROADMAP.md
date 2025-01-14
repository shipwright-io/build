<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Project Roadmap

Shipwright's detailed roadmap can be found on the
[Shipwright Overview GitHub Project](https://github.com/orgs/shipwright-io/projects/6).

## Focus Areas

- Security
  - Incorporate Supply Chain Security Best Practices and deliver them to users through our Build API.
  - Continuously address and communicate vulnerability findings across our projects.

- Community
  - Outreach through diverse channels to ensure continuous engagement, such as talks, blog posts, social media.
  - Ensure our Community [standards](https://github.com/shipwright-io/.github/blob/main/CODE_OF_CONDUCT.md#our-standards) are consistently applied.
  - Cultivate a strong relationship with the adopted Foundation.
  - Encourage and recognize contributions.

- Build Tools
  - Consistently integrate the latest features of Build Tools into our Build Strategies.
  - Expand our support for Build Tools in response to user demand or market trends.

- Third-Party Integrations
  - Periodically evaluate and update our existing integrations.
  - Ensure Integrations with CI/CD tools that can leverage Shipwright or existing CI/CD pipelines.
  - Encourage contributors to create integrations.

## 2025 Release Schedule

The build sub-project releases minor version updates quarterly with new features and
updates to its dependencies. We use the latest Kubernetes minor version and Tekton LTS version for
development and testing. Older versions of Kubernetes and Tekton may be supported by the community
on a best-effort basis.

Below is the tentative release schedule for 2025, with the anticipated versions of Kubernetes and
Tekton used for development. Actual release dates may vary based on community availability and
release stability.

- v0.15.0: week of 2025-02-14
  - Kubernetes version: 1.32
  - Tekton Pipelines version: 0.68-LTS (expected)
- v0.16.0: week of 2025-05-16
  - Kubernetes version: 1.33 (expected)
  - Tekton Pipelines version: 0.71-LTS (expected)
- v0.17.0: week of 2025-08-15
  - Kubernetes version: 1.33 or 1.34 (expected)
  - Tekton Pipelines version: 0.74-LTS
- v0.18.0: week of 2025-11-14
  - Kubernetes version: 1.34 (expected)
  - Tekton Pipelines version: 0.77-LTS (expected)

The build sub-project leads the overall
[Shipwright release schedule](https://github.com/shipwright-io/community/blob/main/ROADMAP.md) by
two weeks. This facilitates updates to dependent Shipwright sub-projects, such as the
[operator](https://github.com/shipwright-io/operator) and
[shp CLI](https://github.com/shipwright-io/cli).
