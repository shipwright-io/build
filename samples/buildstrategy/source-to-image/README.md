# `source-to-image` Build Strategy

The [strategy](./buildstrategy_source-to-image_cr.yaml) is composed by [`source-to-image`][s2i]
and [`buildah`][buildah] in order to generate a `Dockerfile` and prepare the application to be
built later on with a builder. Tipically `s2i` requires a specially crafted image, which can be
informed as `builderImage` parameter.

To install this strategy, use:

```sh
kubectl apply -f samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml
```

## Build Steps

1. `s2i` in order to generate a `Dockerfile` and prepare source-code for image build;
2. `buildah` to create the container image;
3. `buildah` to push container-image on `output.image` parameter;

[s2i]: https://github.com/openshift/source-to-image
[buildah]: https://github.com/containers/buildah