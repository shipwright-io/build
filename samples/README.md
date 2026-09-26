<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Samples

These manifests demonstrate Shipwright `Build`, `BuildRun`, `BuildStrategy`, and `ClusterBuildStrategy` resources. Apply them only after installing Shipwright Build and the strategy resources they reference. The resource documentation explains the fields in detail: [`Build`](../docs/build.md), [`BuildRun`](../docs/buildrun.md), and [`BuildStrategies`](../docs/buildstrategies.md).

The samples are grouped by API version. `v1beta1` and `v1alpha1` samples are provided for the corresponding API versions.

## `v1beta1`

### Build and BuildRun examples

- [Build examples](v1beta1/build)
- [BuildRun examples](v1beta1/buildrun)
- [Buildah vulnerability scanning](v1beta1/build/build_buildah_vulnerability_scanning_cr.yaml)

### Build technologies

- [Buildah](v1beta1/build/build_buildah_shipwright_managed_push_cr.yaml) and [strategy-managed push](v1beta1/build/build_buildah_strategy_managed_push_cr.yaml)
- [BuildKit](v1beta1/build/build_buildkit_cr.yaml)
- [Buildpacks v3](v1beta1/build/build_buildpacks-v3_cr.yaml) and [Heroku Buildpacks](v1beta1/build/build_buildpacks-v3-heroku_cr.yaml)
- [Kaniko](v1beta1/build/build_kaniko_cr.yaml)
- [ko](v1beta1/build/build_ko_cr.yaml)
- [Source-to-Image](v1beta1/build/build_source-to-image_cr.yaml)
- [Multi-architecture Buildah strategy](v1beta1/buildstrategy/multiarch-native-buildah/buildstrategy_multiarch_native_buildah_cr.yaml)

### Strategy scope and supporting resources

- [All v1beta1 strategy manifests](v1beta1/buildstrategy)
- [Namespaced Buildpacks v3 strategy](v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml)
- [Cluster-scoped Buildpacks v3 strategy](v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml)
- [Multi-architecture Buildah ClusterRole](v1beta1/buildstrategy/multiarch-native-buildah/clusterrole_multiarch_native_buildah_cr.yaml)
- [Multi-architecture Buildah RoleBinding](v1beta1/buildstrategy/multiarch-native-buildah/rolebinding_multiarch_native_buildah_cr.yaml)

## `v1alpha1`

### Build and BuildRun examples

- [Build examples](v1alpha1/build)
- [BuildRun examples](v1alpha1/buildrun)

### Build technologies

- [Buildah](v1alpha1/build/build_buildah_shipwright_managed_push_cr.yaml) and [strategy-managed push](v1alpha1/build/build_buildah_strategy_managed_push_cr.yaml)
- [BuildKit](v1alpha1/build/build_buildkit_cr.yaml)
- [Buildpacks v3](v1alpha1/build/build_buildpacks-v3_cr.yaml) and [Heroku Buildpacks](v1alpha1/build/build_buildpacks-v3-heroku_cr.yaml)
- [Kaniko](v1alpha1/build/build_kaniko_cr.yaml)
- [ko](v1alpha1/build/build_ko_cr.yaml)
- [Source-to-Image](v1alpha1/build/build_source-to-image_cr.yaml)

### Strategy scope

- [All v1alpha1 strategy manifests](v1alpha1/buildstrategy)
- [Namespaced Buildpacks v3 strategy](v1alpha1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml)
- [Cluster-scoped Buildpacks v3 strategy](v1alpha1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml)
