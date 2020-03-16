# `buildah` Build Strategy

The `buildah` BuildStrategy consists of using `buildah`[buildah] to build and push a container image, out
of a `Dockerfile`, so it does expect to find a `Dockerfile` on the source-code, or alternatively
informed on `Build` object. As a common `spec.builderImage` use `quay.io/buildah/stable`.

You can install the `BuildStrategy` in your namespace or install the `ClusterBuildStrategy` at cluster scope so that it can be shared across namespaces.

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml
```

**NOTE:** 
You can switch to use namespaced scope `BuildStrategy` by changing the kind of strategy in above yaml file:
```yaml
  strategy:
    name: buildah
    kind: BuildStrategy
```

[buildah]: https://github.com/containers/buildah
