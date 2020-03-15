# `kaniko` Build Strategy

The `kaniko` BuildStrategy is composed by Kaniko's `executor`[kaniko], with the objective of building a
container-image, out of informed `Dockerfile` and context directory.

You can install the `BuildStrategy` in your namespace or install the `ClusterBuildStrategy` at cluster scope so that it can be shared across namespaces.

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml
```

**NOTE:** 
You can switch to use namespaced scope `BuildStrategy` by changing the kind of strategy in above yaml file:
```yaml
  strategy:
    name: kaniko
    kind: BuildStrategy
```

[kaniko]: https://github.com/GoogleContainerTools/kaniko
