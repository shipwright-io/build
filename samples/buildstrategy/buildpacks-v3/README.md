# `buildpacks-v3` Build Strategy

Supporting [buildpacks-v3][buildpacks] `BuildStrategy` via a Cloud Native Builder ([CNB][cnb])
container image, able to implement [lifecycle commands][lifecycle]. The following CNB images are the
most common options:

* [`heroku/buildpacks:18`][hubheroku];
* [`cloudfoundry/cnb:bionic`][hubcloudfoundry];

To install this strategy, use:

```sh
kubectl apply -f samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml
```

## Lifecycle Steps

* **detector**: inpect for the type of project to be build;
* **analyzer**: inpect for container images and previous project build cache;
* **restorer**: restore previous state before building;
* **builder**: execute the actual container image build;

[buildpacks]: https://buildpacks.io/
[cnb]: https://buildpacks.io/docs/concepts/components/builder/
[lifecycle]: https://buildpacks.io/docs/concepts/components/lifecycle/
[hubheroku]: https://hub.docker.com/r/heroku/buildpacks/
[hubcloudfoundry]: https://hub.docker.com/r/cloudfoundry/cnb