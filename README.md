# kubernetes-sidecar-cleaner


> [!CAUTION]
> With the adoption of native SidecarContainers this project is obsolete and is not needed for Istio running on Kubernetes 1.29 and later. Fur further details please consult the [Istio documentation](https://istio.io/latest/blog/2023/native-sidecars/).


A simple app to watch and clean up istio-proxy sidecars in kubernetes Jobs with /quitquitquit.

## Installation
```shell
helm repo add sidecar-cleaner https://opensource.aoe.com/kubernetes-sidecar-cleaner/
helm install my-sidecar-cleaner sidecar-cleaner/sidecar-cleaner
```
Also see: [opensource.aoe.com/kubernetes-sidecar-cleaner/](https://opensource.aoe.com/kubernetes-sidecar-cleaner/)

## Configuration
Please check the [values.yaml](charts/sidecar-cleaner/values.yaml) for possible configurations.

## Contribution

### Release a new app version
- Create a tag/release following  semantic versioning in the format of vx.y.z (eg. v1.17.5). The [github worklow](.github/workflows/docker.yml) takes care of pushing a new container to the registry.
- Update the `appVersion` in [Chart.yaml](charts/sidecar-cleaner/Chart.yaml) and push the changes.
- [Release a new helm chart version](#release-a-new-helm-chart-version)

### Release a new helm chart version
- Increase the `version` in [Chart.yaml](charts/sidecar-cleaner/Chart.yaml) and push the changes. The [github worklow](.github/workflows/helm.yml) takes care of releasing a new chart version.
