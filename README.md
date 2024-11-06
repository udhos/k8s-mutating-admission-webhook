[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/k8s-mutating-admission-webhook/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/k8s-mutating-admission-webhook)](https://goreportcard.com/report/github.com/udhos/k8s-mutating-admission-webhook)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/k8s-mutating-admission-webhook.svg)](https://pkg.go.dev/github.com/udhos/k8s-mutating-admission-webhook)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/k8s-mutating-admission-webhook)](https://artifacthub.io/packages/search?repo=k8s-mutating-admission-webhook)
[![Docker Pulls](https://img.shields.io/docker/pulls/udhos/k8s-mutating-admission-webhook)](https://hub.docker.com/r/udhos/k8s-mutating-admission-webhook)

# k8s-mutating-admission-webhook

k8s-mutating-admission-webhook

* [Concepts](#concepts)
* [Create kind cluster](#create-kind-cluster)
* [Build](#build)
* [Test](#test)
* [Docker](#docker)
* [Helm chart](#helm-chart)
  * [Using the helm repository](#using-the-helm-repository)
  * [Using local chart](#using-local-chart)
* [References](#references)
  * [Patch](#patch)
  * [Webhook](#webhook)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc.go)

# Concepts

The MutatingWebhookConfiguration is where we tell k8s which resource requests should be sent to our webhook.

The K8S api-server will send a AdmissionReview “request” and expects a AdmissionReview “response”

The AdmissionReview consists of AdmissionRequest and AdmissionResponse objects.

The webhook needs to “unmarshal” theAdmissionReview from JSON format into some kind of object so it can read the AdmissionRequest and modify the AdmissionResponse object within it.

The webhook creates its own AdmissionResponse object, copies the UID from the AdmissionRequest object and replaces the AdmissionResponse object within the AdmissionReview with its own (overwrites it).

AdmissionReview request:

```
{
  "kind": "AdmissionReview",
  "apiVersion": "admission.k8s.io/v1beta1",
  "request": {...}
}
```

AdmissionReview response:

```
{
  "kind": "AdmissionReview",
  "apiVersion": "admission.k8s.io/v1beta1",
  "request":  { <<ORIGINAL REQUEST>> },
  "response": { <<OUR RESPONSE>>     }
}
```

# Create kind cluster

Check versions.

```
kubectl version --short
Flag --short has been deprecated, and will be removed in the future. The --short output will become the default.
Client Version: v1.27.3
Kustomize Version: v5.0.1
Server Version: v1.27.3
```

```
$ kind version
kind v0.20.0 go1.20.4 linux/amd64
```

Create kind cluster.

```
$ kind create cluster --name lab
```

Make sure the MutatingAdminssionController is enabled in the k8s api-server.

```
$ kubectl api-resources | grep -i mutating
mutatingwebhookconfigurations                  admissionregistration.k8s.io/v1        false        MutatingWebhookConfiguration

$ kubectl api-versions | grep -i admission
admissionregistration.k8s.io/v1

$ kubectl get apiservices | grep -i admission
v1.admissionregistration.k8s.io        Local     True        3m47s
```

Find existing MutatingWebhookConfigurations.

```
$ kubectl get mutatingwebhookconfigurations
No resources found
```

# Build

```
./build.sh
```

Run.

```
export DEBUG=true
webhook
```

Test.

```
curl -k -d '{"a":"b"}' -H 'content-type: application/json' https://localhost:8443/mutate
```

# Test

```
./docker/build.sh

docker push udhos/k8s-mutating-admission-webhook:0.0.0; docker push udhos/k8s-mutating-admission-webhook:latest

kind create cluster --name lab

kubectl apply -f deploy

kubectl -n webhook get po
NAME                                             READY   STATUS    RESTARTS   AGE
k8s-mutating-admission-webhook-65f8bb6b4-h6gd4   1/1     Running   0          4m13s

kubectl -n webhook logs k8s-mutating-admission-webhook-65f8bb6b4-h6gd4

kubectl run nginx --image=nginx:latest

kubectl logs nginx

kind delete cluster --name lab
```

# Docker

Docker hub:

https://hub.docker.com/r/udhos/k8s-mutating-admission-webhook


# Helm chart

## Using the helm repository

See https://udhos.github.io/k8s-mutating-admission-webhook/.

## Using local chart

```
kubectl create ns webhook

kubectl label ns webhook webhook=yes

# render chart to stdout
helm template k8s-mutating-admission-webhook ./charts/k8s-mutating-admission-webhook -n webhook

# install chart
helm upgrade k8s-mutating-admission-webhook ./charts/k8s-mutating-admission-webhook -n webhook --install

# logs
kubectl -n webhook logs deploy/k8s-mutating-admission-webhook -f
```

# References

## Patch

```json
[
    {"op":"add","path":"/metadata/labels","value":{"hello":"world"}}
]
```

```json
PATCH /my/data HTTP/1.1
Host: example.org
Content-Length: 326
Content-Type: application/json-patch+json
If-Match: "abc123"

[
    { "op": "test", "path": "/a/b/c", "value": "foo" },
    { "op": "remove", "path": "/a/b/c" },
    { "op": "add", "path": "/a/b/c", "value": [ "foo", "bar" ] },
    { "op": "replace", "path": "/a/b/c", "value": 42 },
    { "op": "move", "from": "/a/b/c", "path": "/a/b/d" },
    { "op": "copy", "from": "/a/b/d", "path": "/a/b/e" }
]
```

* [Is there a way in "kubectl patch" to delete a specific object in an array without specifying the index?](https://stackoverflow.com/questions/64355902/is-there-a-way-in-kubectl-patch-to-delete-a-specific-object-in-an-array-withou)

* [JSON Patch](https://www.rfc-editor.org/rfc/rfc6902)

* [JSON Pointer](https://www.rfc-editor.org/rfc/rfc6901)

## Webhook

* [Article - Writing a very basic kubernetes mutating admission webhook](https://medium.com/ovni/writing-a-very-basic-kubernetes-mutating-admission-webhook-398dbbcb63ec)

* [github - A playground to build a very crude k8s mutating webhook in Go](https://github.com/alex-leonhardt/k8s-mutate-webhook)

* [Article - Create a Basic Kubernetes Mutating Webhook](https://trstringer.com/kubernetes-mutating-webhook/)

* [github - Kubernetes Mutating Webhook example](https://github.com/trstringer/kubernetes-mutating-webhook)

* [github - Kubernetes Mutating webhook sample example](https://github.com/cloud-ark/sample-mutatingwebhook)

* [Article - Diving into Kubernetes MutatingAdmissionWebhook](https://medium.com/ibm-cloud/diving-into-kubernetes-mutatingadmissionwebhook-6ef3c5695f74)

* [github - A Kubernetes mutating webhook server that implements sidecar injection](https://github.com/morvencao/kube-sidecar-injector)

* [Article - Dynamic Admission Control Certificate Management with cert-manager](https://trstringer.com/admission-control-cert-manager/)

* [Article - In-depth introduction to Kubernetes admission webhooks](https://banzaicloud.com/blog/k8s-admission-webhooks/)

