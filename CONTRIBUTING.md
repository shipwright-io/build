<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Contributing Guidelines

Welcome to Shipwright Build. We are excited about the prospect of you contributing to our project. Your support is more than welcome!

## Getting Started

We have initial documentation on how to start contributing here:

- Learn how to [change shipwright and try out your changes on a local cluster](/docs/development/local_development.md)
- Our main [documentation](/docs/)
- Our [Code of Conduct](/code-of-conduct.md)

## Creating new Issues

We recommend to open an issue for the following scenarios:

- Asking for help or questions. (_Use the **discussion** or **help_wanted** label_)
- Reporting a Bug. (_Use the **bug** label_)
- Requesting a new Feature. (_Use the **enhancement** label_)

The Shipwright maintainers can also be reached in our [Kubernetes Slack channel](https://kubernetes.slack.com/archives/C019ZRGUEJC).

## Writing Pull Requests

Contributions can be submitted by creating a pull request on Github. We recommend you do the following to ensure the maintainers can collaborate on your contribution:

- Fork the project into your personal Github account
- Create a new feature branch for your contribution
- Make your changes
- If you make code changes, ensure unit tests are passing by running `make test-unit`
- Open a PR with a nice description and a link to the Github issue where the changes were previously discussed.

## Code review process

There is an integration on our Github repository that automatically do things for us. Once a PR is open the tool will assign two members of the project for the code review.

The code review should cover:

- Ensure all related tests(unit, integration and e2e) are passing.
- Ensure the code style is compliant with the [coding conventions](https://github.com/kubernetes/community/blob/master/contributors/guide/coding-conventions.md)
- Ensure the code is properly documented, e.g. enough comments where needed.
- Ensure the code is adding the necessary test cases(unit, integration or e2e) if needed.

## Community Meetings Participation

We run the community meetings every Monday at 13:00 UTC time.
For each upcoming meeting we generate a new issue where we layout the topics to discuss.
See our [previous meetings](https://github.com/shipwright-io/build/issues?q=is%3Aissue+label%3Acommunity+is%3Aclosed) outcomes.
To join, please request an invite in our Slack [channel](https://kubernetes.slack.com/archives/C019ZRGUEJC).

## Contact Information

- [Slack channel](https://kubernetes.slack.com/archives/C019ZRGUEJC)
- End-user email list: [shipwright-users@lists.shipwright.io](https://lists.shipwright.io/admin/lists/shipwright-users.lists.shipwright.io/)
- Developer email list: [shipwright-dev@lists.shipwright.io](https://lists.shipwright.io/admin/lists/shipwright-dev.lists.shipwright.io/)
