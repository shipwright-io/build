# Git Clone Wrapper

**TL;DR:** As part of the build, the sources need to be retrieved. One option is to use `git` to clone the source to the container filesystem. This was used to be done by a Tekton Git Resource. This package contains Shipwright Build owned Git retrieval code, which is wrapping around the `git` CLI in a minimal container setup.

## Features

- SSH private key based access to Git repositories
- Basic Auth username/password access to Git repositories
- Git Large File Storage (LFS) based Git repositories
- Recursive sub-module update
- Cloning using default remote branch
- Cloning using specific branch name
- Cloning using specific tag
- Cloning using specific commit SHA
- Does not interfere with local SSH config

## Development

### Base image

The Git Clone Wrapper wraps around command line tools to serve as convenience layer. Therefore, it requires the respective binaries to be in the path. If you want to build your own base image for the wrapper CLI, make sure the following dependencies are met:

- **SSH** - version `OpenSSH_8.0p1, OpenSSL 1.1.1g FIPS  21 Apr 2020` is known to work, older versions are very likely to work as well
- **Git** - version `2.27.0` is known to work, older versions are very likely to work as well
- **Git Large File Storage (LFS)** - version `2.11.0` is known to be working

### Run the CLI code

- Run it locally:

  ```sh
  go run cmd/git/main.go \
  --url https://github.com/shipwright-io/sample-go \
  --revision 0e0583421a5e4bf562ffe33f3651e16ba0c78591 \
  --target /tmp/workspace/source
  ```

- Run it using `ko` (base image defined in `.ko.yaml`)

  ```sh
  docker run \
    --rm \
    --volume /tmp/workspace:/workspace \
    $(KO_DOCKER_REPO=ko.local ko publish --bare ./cmd/git) \
      --url https://github.com/shipwright-io/sample-lfs \
      --target /workspace/source
  ```
