<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Running on development mode

The following document highlights how to deploy a Build controller locally for running on development mode.

**Before generating an instance of the Build controller, ensure the following:**

- Target your Kubernetes cluster. We recommend the usage of KinD for development, which you can launch via our [install-kind.sh](/hack/install-kind.sh) script.
- On the cluster, ensure the Tekton controllers are running. You can use our Tekton installation script in [install-tekton.sh](/hack/install-tekton.sh)

---

Once the code have been modified, you can generate an instance of the Build controller running locally to validate your changes. For running the Build controller locally via the `local` target:

```sh
pushd $GOPATH/src/github.com/shipwright-io/build
  make local
popd
```

_Note_: The above target will uninstall/install all related CRDs and start an instance of the controller via the `operator-sdk` binary. All existing CRDs instances will be deleted.
