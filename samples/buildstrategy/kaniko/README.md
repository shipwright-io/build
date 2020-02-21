# `kaniko` Build Strategy

The `kaniko` strategy is composed by Kaniko's `executor`[kaniko], with the objective of building a
container-image, out of informed `Dockerfile` and context directory.

To install this strategy, use:

```sh
kubectl apply -f samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml
```

[kaniko]: https://github.com/GoogleContainerTools/kaniko