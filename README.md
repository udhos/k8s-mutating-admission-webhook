# k8s-mutating-admission-webhook
k8s-mutating-admission-webhook

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
$ kubectl api-versions | grep admissionregistration
admissionregistration.k8s.io/v1
$
```

# Build

```
./build.sh
```

Create a certificate for testing.

```
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout key.pem -out cert.pem
```

Run.

```
export ADDR=:8443
webhook
```

Test.

```
curl -k -d '{"a":"b"}' https://localhost:8443/mutate
```

# References

* [Article - Writing a very basic kubernetes mutating admission webhook](https://medium.com/ovni/writing-a-very-basic-kubernetes-mutating-admission-webhook-398dbbcb63ec)
* [github - A playground to build a very crude k8s mutating webhook in Go](https://github.com/alex-leonhardt/k8s-mutate-webhook)

* [Article - Create a Basic Kubernetes Mutating Webhook](https://trstringer.com/kubernetes-mutating-webhook/)
* [github - Kubernetes Mutating Webhook example](https://github.com/trstringer/kubernetes-mutating-webhook)

* [github - Kubernetes Mutating webhook sample example](https://github.com/cloud-ark/sample-mutatingwebhook)

* [A Kubernetes mutating webhook server that implements sidecar injection](https://github.com/morvencao/kube-sidecar-injector)

* [Dynamic Admission Control Certificate Management with cert-manager](https://trstringer.com/admission-control-cert-manager/)
