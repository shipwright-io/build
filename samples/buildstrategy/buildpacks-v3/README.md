# `buildpacks-v3` Build Strategy

Supporting [buildpacks-v3][buildpacks] BuildStrategy via a Cloud Native Builder ([CNB][cnb])
container image, able to implement [lifecycle commands][lifecycle]. The following CNB images are the
most common options:

* [`heroku/buildpacks:18`][hubheroku];
* [`cloudfoundry/cnb:bionic`][hubcloudfoundry];

You can install the `BuildStrategy` in your namespace or install the `ClusterBuildStrategy` at cluster scope so that it can be shared across namespaces.

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml
```

To install the namespaced scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml
```


## Try it!

The buildpacks-v3 strategy needs you specify
* A Quay.io or a DockerHub image repository where the built image would be pushed to
* The credentials needed to push to the repository - a Docker configuration to access your image host.

### Fetch Quay Docker Config

Visit the settings page for your Quay.io account `"https://quay.io/user/<USERNAME>?tab=settings"`

You'll be prompted to authenticate, and then you'll get a screen that allows you download credential, pick the "Docker Configuraiton" on the left hand of the screen.

On this screen, there is a link below "Step 1", to download your secret "Download <USERNAME>-auth.json", download this file.

### Create a Kubernetes Secret

```
oc create secret generic regcred --from-file=.dockerconfigjson="$HOME/Downloads/${QUAYIO_USERNAME}-auth.json" --type=kubernetes.io/dockerconfigjson
```

### Create a Build
The Build is a build definition which defines the related configuration for the build.
Such as Github source, Output Image, BuildStrategy, etc...
```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildpack-nodejs-build
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy:
    name: buildpacks-v3
    kind: ClusterBuildStrategy
  builderImage: heroku/buildpacks:18
  output:
    image: quay.io/yourorg/yourrepo
    credentials: regcred
```

### Start a BuildRun
The BuildRun is a build execution instance which runs from an existing build definition
```yaml
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: buildpack-nodejs-buildrun
spec:
  buildRef:
    name: buildpack-nodejs-build
```

**NOTE:** 
You can switch to use namespaced scope `BuildStrategy` by changing the kind of strategy in above yaml file:
```yaml
  strategy:
    name: buildpacks-v3
    kind: BuildStrategy
```


## Lifecycle Steps

* **detector**: inspect for the type of project to be build;
* **restorer**: restore previous state before building;
* **builder**: execute the actual container image build;
* **exporter**: upload container image to registry;

[buildpacks]: https://buildpacks.io/
[cnb]: https://buildpacks.io/docs/concepts/components/builder/
[lifecycle]: https://buildpacks.io/docs/concepts/components/lifecycle/
[hubheroku]: https://hub.docker.com/r/heroku/buildpacks/
[hubcloudfoundry]: https://hub.docker.com/r/cloudfoundry/cnb
