# Git Clone Wrapper

**TL;DR:** As part of the build, the sources need to be retrieved. One option is to use `git` to clone the source to the container filesystem. This was used to be done by a Tekton Git Resource. This package contains Shipwright Build owned Git retrieval code, which is wrapping around the `git` CLI in a minimal container setup.

## Features

- SSH private key based access to Git repositories
- Basic Auth username/password access to Git repositories
- Git Large File Storage (LFS) based Git repositories
- Recursive Submodule update
- Cloning using default remote branch
- Cloning using specific branch name
- Cloning using specific tag
- Cloning using specific commit SHA
- Does not interfere with local SSH config

## Development

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
