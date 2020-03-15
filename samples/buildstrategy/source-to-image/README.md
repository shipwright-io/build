# `source-to-image` Build Strategy

The [BuildStrategy](./buildstrategy_source-to-image_cr.yaml) is composed by [`source-to-image`][s2i]
and [`buildah`][buildah] in order to generate a `Dockerfile` and prepare the application to be
built later on with a builder. Tipically `s2i` requires a specially crafted image, which can be
informed as `builderImage` parameter.

You can install the `BuildStrategy` in your namespace or install the `ClusterBuildStrategy` at cluster scope so that it can be shared across namespaces.

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml
```

**NOTE:** 
You can switch to use namespaced scope `BuildStrategy` by changing the kind of strategy in above yaml file:
```yaml
  strategy:
    name: source-to-image
    kind: BuildStrategy
```

## Build Steps

1. `s2i` in order to generate a `Dockerfile` and prepare source-code for image build;
2. `buildah` to create the container image;
3. `buildah` to push container-image on `output.image` parameter;

[s2i]: https://github.com/openshift/source-to-image
[buildah]: https://github.com/containers/buildah
