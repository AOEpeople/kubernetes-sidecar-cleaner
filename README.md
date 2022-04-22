## Usage

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

helm repo add sidecar-cleaner https://sidecar-cleaner.github.io/helm-charts

If you had already added this repo earlier, run `helm repo update` to retrieve
the latest versions of the packages.  You can then run `helm search repo
sidecar-cleaner` to see the charts.

To install the sidecar-cleaner chart:

    helm install my-sidecar-cleaner sidecar-cleaner/sidecar-cleaner

To uninstall the chart:

    helm delete my-sidecar-cleaner