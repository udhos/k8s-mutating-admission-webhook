# Usage

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

    helm repo add k8s-mutating-admission-webhook https://udhos.github.io/k8s-mutating-admission-webhook

Update files from repo:

    helm repo update

Search k8s-mutating-admission-webhook:

    $ helm search repo k8s-mutating-admission-webhook -l --version ">=0.0.0"
    NAME                                              	CHART VERSION	APP VERSION	DESCRIPTION
    k8s-mutating-admission-webhook/k8s-mutating-adm...	0.3.0        	0.2.0      	A Helm chart installing k8s-mutating-admission-...
    k8s-mutating-admission-webhook/k8s-mutating-adm...	0.2.0        	0.1.0      	A Helm chart installing k8s-mutating-admission-...
    k8s-mutating-admission-webhook/k8s-mutating-adm...	0.1.0        	0.0.0      	A Helm chart installing k8s-mutating-admission-...

To install the charts:

    kubectl create ns webhook

    kubectl label ns webhook webhook=yes

    helm install k8s-mutating-admission-webhook k8s-mutating-admission-webhook/k8s-mutating-admission-webhook -n webhook
    #            ^                              ^                              ^
    #            |                              |                               \__ chart
    #            |                              |
    #            |                               \_________________________________ repo
    #            |
    #             \________________________________________________________________ release (chart instance installed in cluster)

To uninstall the charts:

    helm uninstall k8s-mutating-admission-webhook -n webhook

    # also delete the webhook configuration
    kubectl delete mutatingwebhookconfiguration udhos.github.io

# Source

<https://github.com/udhos/k8s-mutating-admission-webhook>
