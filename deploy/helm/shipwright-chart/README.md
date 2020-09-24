# shipwright-operator

## Introduction

This helm chart deploys the build-operator from shipwright resources.

## Installing the Chart

To install helm chart for the deployment of build-operator.

Before deploy to k8s, you may run `dry-run` command to check if the value of deployment is injected as expectation.

```bash
helm install --dry-run build-operator ./shipwright-chart/
  --set operator.name={name of operator}
  --set operator.image={your.operator.image}
  --set namespace.name={namespace to deploy operator}
```

If all values are expected to be injected, then install it.
```bash
helm install build-operator ./shipwright-chart/
  --set operator.name={name of operator}
  --set operator.image={your.operator.image}
  --set namespace.name={namespace to deploy operator}
```
Helm will deploy the manifest defined in `templates\` directory, which will deploy an operator named `build-operator` by default and deploy the operator of shipwright resources in targeted namespace.
## Uninstalling the Chart

To delete the helm chart:

```bash
helm delete build-operator 
```

## Configuration



| Parameter                                         | Description                                                                                       | Default                                        |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `operator.image`| the image that contains shipwright operator binary| REPLACE_IMAGE 
| `operator.name`| the name of deployed operaotr| build-operator 
| `namespace.name`| namespace to deply operator| build-operator 


