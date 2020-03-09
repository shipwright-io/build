# `buildah` Build Strategy

The `buildah` strategy consists of using `buildah`[buildah] to build and push a container image, out
of a `Dockerfile`, so it does expect to find a `Dockerfile` on the source-code, or alternatively
informed on `Build` object. As a common `spec.builderImage` use `quay.io/buildah/stable`.

To install this strategy, use:

```sh
kubectl apply -f samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml
```

[buildah]: https://github.com/containers/buildah