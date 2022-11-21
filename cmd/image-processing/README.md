<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->
# Image processing

As part of the build, the output image needs to be mutated (annotated and labeled), and pushed. This package contains Shipwright Build owned image processing code.

## Features

- Mutate the image with [annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md)
- Mutate the image with labels
- Push the image

## Development

### Run the CLI code

- Run it locally:

  ```sh
  go run cmd/image-processing/main.go \
  --image $IMAGE \
  --annotation "org.opencontainers.image.url=https://my-company.com/images" \
  --label "maintainer=team@my-company.com" \
  [--insecure] \
  [--push some-local-dir-or-tarball]
  ```

  If we are trying to mutate the image in a private registry, authentication to the registry should be done before running the command.

- Run it using `ko` (base image defined in `.ko.yaml`)

  ```sh
    docker run \
      --rm \
      --volume $HOME/.docker/config.json:/.docker/config.json \
      -e DOCKER_CONFIG=.docker \
      $(KO_DOCKER_REPO=ko.local ko publish --bare ./cmd/image-processing) \
      --image $IMAGE \
      --annotation "org.opencontainers.image.url=https://my-company.com/images" \
      --label "maintaner=team@my-company.com"
  ```
