<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: CLI
authors:
  - "@otaviof"

reviewers:
  - "@gabemontero"
  - "@HeavyWombat"
  - "@coreydaley"

approvers:
  - TBD
creation-date: 2020-10-07
last-updated:  2020-11-30
status: provisional
---

CLI
---

**Build Enhancement Proposals have been moved into the Shipwright [Community](https://github.com/shipwright-io/community) repository. This document holds an obsolete Enhancement Proposal, please refer to the up-to-date [SHIP](https://github.com/shipwright-io/community/blob/main/ships/0009-cli.md) for more information.**

# Summary

This enhancement proposal describes a command-line client for Shipwright Build operator, covering
from the command line structure, the usage as a kubectl plugin, and as a standalone binary, as
well as testing this new project.

# Motivation

Command-line interface is a fundamental component for developer experience and modern automation.
This enhancement proposal describes a command-line interface, i.e. "CLI", to enhance the developer
experience and provide easier means to interact and automate Shipwright's Builds.

## Goals

- Improve the user experience by providing a specialized client
- Make possible extended automation with shell scripting

## Non-Goals

- Re-implement operator logic
- Become the mandatory API client

# Proposal

Create a command-line client for Shipwright Build to improve the user experience and also improve
the ability to automate workflows.

This new project will concentrate our effects on the API client side of the Shipwright project,
and by maintaining the expected separation of `cmd` and `pkg`, developers can embed and extend the
possibilities of consuming Shipwright Operator API.

A [proof-of-concept command-line interface project](https://github.com/otaviof/shp) has been
written to scaffold the features, and evaluate the changes proposed in this document. The project
documentation (`README.md`) is focused on developers and contributors, while the features it will
provide are contained in this document.

## Usage and Installation

This command-line application will be consumed as a standalone binary, or as a plugin in `kubectl`.
This implies a few more extra settings, and making sure the application binary complies with the
[expected convention](https://krew.sigs.k8s.io/docs/developer-guide/develop/best-practices/).

### Standalone

The standalone binary will be available via the usual means, final users will be able to simply
install with `go get…`, or pick a pre-compiled binary from GitHub release pages. Consider the
[release section](#Releasing) for more.

### Plugin

As a [krew plugin](https://krew.sigs.k8s.io/docs/developer-guide/), the expected usage is:

```sh
$ kubectl shp ...
```

And the following global flags are expected, to keep consistency with `kubectl`:

- `-h/--help`
- `-n/--namespace`
- `-A/--all-namespaces`

Those three flags are mandatory to keep compatibility. However, the
[upstream kubectl package](https://pkg.go.dev/k8s.io/kubectl@v0.17.6/pkg/cmd/util) provides
[more options](https://github.com/otaviof/shp/blob/55ce2e8d58435b0264e3db0bef5cf439abfeca18/vendor/k8s.io/cli-runtime/pkg/genericclioptions/config_flags.go#L317)
out of the box.

## Command

The project will use [cobra](https://github.com/spf13/cobra)/[viper](https://github.com/spf13/viper)
for scaffolding the command-line structure, which in terms of organization and workflow, follows
`kubectl` convention. Thus, the following pattern:

```sh
$ shp <verb> <resource> <name> [options]
```

For instance:

```sh
$ shp create build nodejs-ex
$ shp run build nodejs-ex
```

Alternatively, we may have a more conventional pattern, as in:

```sh
$ shp <resource> [verb] <name> [options]
```

For example:

```sh
$ shp build create nodejs-ex
$ shp build run nodejs-ex
$ shp logs nodejs-ex
```

During the review process of this enhancement proposal, we must decide which pattern suits better,
which one we prefer to go ahead with.

## Features

For the initial release, the aim is to be able to define a `Build`, run it and display logs.

### Defining a Build

```sh
$ shp <verb> build <name> [options]
```

Will be responsible for managing Build objects, using the verbs: `create`, `update`, list and
`delete`. Later on, end users should have the ability to bootstrap a local repository clone using a
single command.

### Running a Build

```sh
$ shp run build <name> [options]
```

This sub-command will instantiate a new BuildRun resource for a Build, effectively starting the
build process. And, will print follow up commands that may be issued to see the BuildRun status
and inspect logs.

### Logs

```sh
$ shp logs <name> [options]
```

Retrieve all logs related to the informed `BuildRun` name. It will also be able to follow
(`--follow`) container log output as they are executed. The log lines displayed are organised by
the sequence of steps, and easy to read the whole build process output in a single go.

Additionally, it must provide easy syntax to reach a specific `BuildRun` instance, or the most
recent `BuildRun` for a `Build`.

## More Features

The command-line interface is planned to host more features, like for instance managing local and
remove artifacts, as well as help end-users to upload data into the cluster.

Thus, the initial design of the CLI must allow more subcommands to be added, accommodating more
features which will depend in a client.

## Testing

The strategy to test the command-line application will be based on `go test` for the unit testing,
and for end-to-end testing we should adopt [bats](https://github.com/sstephenson/bats). Bats
framework gives us a structured way to run shell script commands and what we expect them to
return, the command-line client shp is no more than a shell command.

The testing structure will be composed of:

- **Unit**: written in Golang and using [Gomega](https://onsi.github.io/gomega/) for assertion;
- **End-to-End**: written in Golang and located at the traditional `test/e2e` directory;
- **System**: written using Bats framework (Bash), or similar approach;

Bats will also be helpful for a future `shp` container-image, we are able to mount the Bats files
in the container-image produced, and run our system testing against it. Therefore, Bats covers
testing of a local command-line binary, as well as it does a container-image. Another tool may take
Bats' place, covering the same test-use cases accordingly.

## Releasing

The command-line release will initially be available on GitHub Pages process, on which we can also
index on Shipwright website. Later on we can evaluate the need for a container-image carrying on
the binary, or an RPM.
