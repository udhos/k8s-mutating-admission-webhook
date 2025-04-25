package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func httpError(w http.ResponseWriter, msg string, code int) {
	log.Printf("%d %s", code, msg)
	http.Error(w, msg, code)
}

func handlerWebhook(app *application, w http.ResponseWriter, r *http.Request) {
	const me = "handlerWebhook"

	if app.conf.debug {
		log.Printf("DEBUG %s: %s %s %s",
			me, r.RemoteAddr, r.Method, r.RequestURI)
	}

	deserializer := app.codecs.UniversalDeserializer()

	// Parse the AdmissionReview from the http request.
	admissionReviewRequest, errAr := admissionReviewFromRequest(r, deserializer, app.conf.debug)
	if errAr != nil {
		msg := fmt.Sprintf("%s: error getting admission review from request: %v",
			me, errAr)
		httpError(w, msg, 400)
		return
	}

	if admissionReviewRequest.Request == nil {
		msg := fmt.Sprintf("%s: missing request in admission review", me)
		httpError(w, msg, 400)
		return
	}

	// Do server-side validation that we are only dealing with correct resource. This
	// should also be part of the MutatingWebhookConfiguration in the cluster, but
	// we should verify here before continuing.
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1",
		Resource: "pods"}
	if admissionReviewRequest.Request.Resource == podResource {
		handlePod(app, w, admissionReviewRequest, deserializer)
		return
	}

	daemonsetResource := metav1.GroupVersionResource{Group: "apps",
		Version: "v1", Resource: "daemonsets"}
	if admissionReviewRequest.Request.Resource == daemonsetResource {
		handleDaemonset(app, w, admissionReviewRequest, deserializer)
		return
	}

	namespaceResource := metav1.GroupVersionResource{Group: "",
		Version: "v1", Resource: "namespaces"}
	if admissionReviewRequest.Request.Resource == namespaceResource {
		handleNamespace(app, w, admissionReviewRequest, deserializer)
		return
	}

	msg := fmt.Sprintf("%s: did not receive pod/daemontset/namespace, got: %s",
		me, admissionReviewRequest.Request.Resource.Resource)
	httpError(w, msg, 400)
}

func handlePod(app *application, w http.ResponseWriter,
	admissionReviewRequest *admissionv1.AdmissionReview, deserializer runtime.Decoder) {

	const me = "handlePod"

	// Decode the pod from the AdmissionReview.
	rawRequest := admissionReviewRequest.Request.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := deserializer.Decode(rawRequest, nil, &pod); err != nil {
		msg := fmt.Sprintf("%s: error decoding raw pod: %v",
			me, err)
		httpError(w, msg, 500)
		return
	}

	namespace := admissionReviewRequest.Request.Namespace
	podName := pod.GetObjectMeta().GetName()
	if podName == "" {
		podName = pod.GetObjectMeta().GetGenerateName()
	}

	// Create a response.
	admissionResponse := &admissionv1.AdmissionResponse{}
	var patch string

	var ignore bool
	for _, ns := range app.conf.ignoreNamespaces {
		if namespace == ns {
			ignore = true
			break
		}
	}

	if ignore {
		log.Printf("pod: %s/%s: ignored", namespace, podName)
	} else {

		// remove tolerations and nodeSelector
		tolerationRemovalList := removeTolerations(namespace, podName, pod.ObjectMeta.Labels, pod.Spec.Tolerations, app.rules.RestrictTolerations)
		nodeSelectorRemovalList := removeNodeSelectors(namespace, podName, pod.Spec.NodeSelector, app.conf.acceptNodeSelectors)

		// add tolerations and nodeSelector
		placementList := addPlacement(namespace, podName, pod.ObjectMeta.Labels, app.rules.PlacePods)

		resourceList := addResource(namespace, podName, pod.ObjectMeta.Labels,
			pod.Spec.Containers, app.rules.Resources, app.conf.debug)

		patchList := append(tolerationRemovalList, nodeSelectorRemovalList...)
		patchList = append(patchList, placementList...)
		patchList = append(patchList, resourceList...)

		if len(patchList) > 0 {
			patch = "[" + strings.Join(patchList, ",") + "]"
		}
	}

	if app.conf.debug {
		log.Printf("DEBUG %s: patch: '%s'",
			me, patch)
	}

	admissionResponse.Allowed = true
	if patch != "" {
		patchType := admissionv1.PatchTypeJSONPatch
		admissionResponse.PatchType = &patchType
		admissionResponse.Patch = []byte(patch)
	}

	// Construct the response, which is just another AdmissionReview.
	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, errMarshal := json.Marshal(admissionReviewResponse)
	if errMarshal != nil {
		msg := fmt.Sprintf("%s: error marshalling response json: %v",
			me, errMarshal)
		httpError(w, msg, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleDaemonset(app *application, w http.ResponseWriter,
	admissionReviewRequest *admissionv1.AdmissionReview, deserializer runtime.Decoder) {

	const me = "handleDaemonset"

	// Decode the daemonset from the AdmissionReview.
	rawRequest := admissionReviewRequest.Request.Object.Raw
	ds := appsv1.DaemonSet{}
	if _, _, err := deserializer.Decode(rawRequest, nil, &ds); err != nil {
		msg := fmt.Sprintf("%s: error decoding raw daemonset: %v",
			me, err)
		httpError(w, msg, 500)
		return
	}

	namespace := admissionReviewRequest.Request.Namespace
	dsName := ds.GetObjectMeta().GetName()

	// Create a response.
	admissionResponse := &admissionv1.AdmissionResponse{}
	var patch string

	var ignore bool
	for _, ns := range app.conf.ignoreNamespaces {
		if namespace == ns {
			ignore = true
			break
		}
	}

	if ignore {
		log.Printf("daemonset: %s/%s: ignored", namespace, dsName)
	} else {
		patchList := daemonsetNodeSelector(namespace, dsName,
			ds.ObjectMeta.Labels, app.rules.DisableDaemonsets)
		if len(patchList) > 0 {
			patch = "[" + strings.Join(patchList, ",") + "]"
		}
	}

	if app.conf.debug {
		log.Printf("DEBUG %s: patch: '%s'",
			me, patch)
	}

	admissionResponse.Allowed = true
	if patch != "" {
		patchType := admissionv1.PatchTypeJSONPatch
		admissionResponse.PatchType = &patchType
		admissionResponse.Patch = []byte(patch)
	}

	// Construct the response, which is just another AdmissionReview.
	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, errMarshal := json.Marshal(admissionReviewResponse)
	if errMarshal != nil {
		msg := fmt.Sprintf("%s: error marshalling response json: %v",
			me, errMarshal)
		httpError(w, msg, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleNamespace(app *application, w http.ResponseWriter,
	admissionReviewRequest *admissionv1.AdmissionReview, deserializer runtime.Decoder) {

	const me = "handleNamespace"

	// Decode the namespace from the AdmissionReview.
	rawRequest := admissionReviewRequest.Request.Object.Raw
	ns := corev1.Namespace{}
	if _, _, err := deserializer.Decode(rawRequest, nil, &ns); err != nil {
		msg := fmt.Sprintf("%s: error decoding raw namespace: %v",
			me, err)
		httpError(w, msg, 500)
		return
	}

	// Create a response.
	admissionResponse := &admissionv1.AdmissionResponse{}
	var patch string

	name := ns.GetObjectMeta().GetName()

	patchList := namespaceAddLabels(name, ns.ObjectMeta.Labels, app.rules.NamespacesAddLabels)
	if len(patchList) > 0 {
		patch = "[" + strings.Join(patchList, ",") + "]"
	}

	if app.conf.debug {
		log.Printf("DEBUG %s: patch: '%s'",
			me, patch)
	}

	admissionResponse.Allowed = true
	if patch != "" {
		patchType := admissionv1.PatchTypeJSONPatch
		admissionResponse.PatchType = &patchType
		admissionResponse.Patch = []byte(patch)
	}

	// Construct the response, which is just another AdmissionReview.
	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, errMarshal := json.Marshal(admissionReviewResponse)
	if errMarshal != nil {
		msg := fmt.Sprintf("%s: error marshalling response json: %v",
			me, errMarshal)
		httpError(w, msg, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func tolerationToString(podToleration corev1.Toleration) string {
	return fmt.Sprintf("key(%s) op(%s) value(%s) effect(%s)",
		podToleration.Key,
		podToleration.Operator,
		podToleration.Value,
		podToleration.Effect)
}

// https://jsonpatch.com/#json-pointer
func escapeJSONPointer(s string) string {
	s1 := strings.ReplaceAll(s, "~", "~0")
	return strings.ReplaceAll(s1, "/", "~1")
}

func admissionReviewFromRequest(r *http.Request, deserializer runtime.Decoder, debug bool) (*admissionv1.AdmissionReview, error) {
	const me = "admissionReviewFromRequest"

	// Validate that the incoming content type is correct.
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("expected application/json content-type")
	}

	// Get the body data, which will be the AdmissionReview
	// content for the request.
	var body []byte
	if r.Body != nil {
		requestData, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = requestData
	}

	if debug {
		log.Printf("DEBUG %s: body: %v", me, string(body))
	}

	// Decode the request body into
	admissionReviewRequest := &admissionv1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, admissionReviewRequest); err != nil {
		return nil, err
	}

	return admissionReviewRequest, nil
}
