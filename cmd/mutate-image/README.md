<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->
# Mutate Image Wrapper

As part of the build, the output image needs to be mutated (annotated and labeled). This package contains Shipwright Build owned image mutate code, which is using the [crane](https://github.com/google/go-containerregistry/tree/main/cmd/crane) in the background.

## Features

- Mutate the image with [annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md)
- Mutate the image with labels

## Development

### Run the CLI code

- Run it locally:

  ```sh
  go run cmd/mutate-image/main.go \
  --image $IMAGE \
  --annotation "org.opencontainers.image.url=https://my-company.com/images" \
  --label "maintainer=team@my-company.com"
  ```

  If we are trying to mutate the image in a private registry, authentication to the registry should be done before running the command.

- Run it using `ko` (base image defined in `.ko.yaml`)

  ```sh
    docker run \
      --rm \
      --volume $HOME/.docker/config.json:/.docker/config.json \
      -e DOCKER_CONFIG=.docker \
      $(KO_DOCKER_REPO=ko.local ko publish --bare ./cmd/mutate-image) \
      --image $IMAGE \
      --annotation "org.opencontainers.image.url=https://my-company.com/images" \
      --label "maintaner=team@my-company.com"
  ```
