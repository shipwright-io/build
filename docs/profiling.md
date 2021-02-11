<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Build Controller Profiling

The build operator supports a `pprof` profiling mode, which is omitted from the binary by default. To use the profiling, use the operator image that was built with `pprof` enabled.

## Enable `pprof` in the build operator

In the Kubernetes cluster, edit the `build-operator` deployment to use the container tag with the `debug` suffix.

```sh
kubectl --namespace <namespace> set image \
  deployment/build-operator \
  build-operator="$(kubectl --namespace <namespace> get deployment build-operator --output jsonpath='{.spec.template.spec.containers[].image}')-debug"
```

## Connect `go pprof` to build operator

Depending on the respective setup, there could be multiple build operator pods for high availability reasons. In this case, you have to look-up the current leader first. The following command can be used to verify the currently active leader:

```sh
kubectl --namespace <namespace> get configmap build-operator-lock --output json \
  | jq --raw-output '.metadata.annotations["control-plane.alpha.kubernetes.io/leader"]' \
  | jq --raw-output .holderIdentity
```

The `pprof` endpoint is not exposed in the cluster and can only be used from inside the container. Therefore, set-up port-forwarding to make the `pprof` port available locally.

```sh
kubectl --namespace <namespace> port-forward <build-operator-pod-name> 8383:8383
```

Now, you can setup a local webserver to browse through the profiling data.

```sh
go tool pprof -http localhost:8080 http://localhost:8383/debug/pprof/heap
```

_Please note:_ For it to work, you have to have `graphviz` installed on your system, for example using `brew install graphviz`, `apt-get install graphviz`, `yum install graphviz`, or similar.
