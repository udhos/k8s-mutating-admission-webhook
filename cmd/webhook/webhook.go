package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func httpError(w http.ResponseWriter, msg string, code int) {
	log.Print(msg)
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

	// Do server-side validation that we are only dealing with a pod resource. This
	// should also be part of the MutatingWebhookConfiguration in the cluster, but
	// we should verify here before continuing.
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if admissionReviewRequest.Request.Resource != podResource {
		msg := fmt.Sprintf("%s: did not receive pod, got %s",
			me, admissionReviewRequest.Request.Resource.Resource)
		httpError(w, msg, 400)
		return
	}

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
		tolerationRemovalList := removeTolerations(namespace, podName, pod.Spec.Tolerations, app.rules.RestrictTolerations)
		nodeSelectorRemovalList := removeNodeSelectors(namespace, podName, pod.Spec.NodeSelector, app.conf.acceptNodeSelectors)
		patchList := append(tolerationRemovalList, nodeSelectorRemovalList...)
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

func tolerationToString(podToleration corev1.Toleration) string {
	return fmt.Sprintf("key(%s) op(%s) value(%s) effect(%s)",
		podToleration.Key,
		podToleration.Operator,
		podToleration.Value,
		podToleration.Effect)
}

func removeTolerations(namespace, podName string, podTolerations []corev1.Toleration,
	restrictToleration []restrictTolerationConfig) []string {

	toRemove := removeTolerationsIndices(namespace, podName, podTolerations,
		restrictToleration)

	// build patch list removing all tolerations by index
	list := make([]string, 0, len(toRemove))
	for _, i := range toRemove {
		list = append(list, fmt.Sprintf(`{"op":"remove","path":"/spec/tolerations/%d"}`, i))
	}
	return list
}

func removeTolerationsIndices(namespace, podName string, podTolerations []corev1.Toleration,
	restrictToleration []restrictTolerationConfig) []int {

	var toRemove []int // list of tolerations index to remove

	size := len(podTolerations)
	removed := make([]bool, size) // report only: tolerations removed
	track := make([]string, size)

	// scan pod tolerations starting from last index down to the first one
	for i := size - 1; i >= 0; i-- {
		pt := podTolerations[i]

		track[i] = "[no toleration rule matched]"

		// scan restricted tolerations
		for j, rt := range restrictToleration {

			if isRestricted := rt.Toleration.match(pt); !isRestricted {
				continue // this rule does not restrict the pod toleration
			}

			//
			// pt is restricted toleration, can the pod have it?
			//
			var isAllowed bool
			podRule := -1
			for k, allowedPod := range rt.AllowedPods {
				if allowedPod.match(namespace, podName) {
					isAllowed = true
					podRule = k
					break
				}
			}
			if !isAllowed {
				//
				// pod is not allowed to have the toleration pt
				//
				toRemove = append(toRemove, i) // add to remove list
				removed[i] = true
				track[i] = fmt.Sprintf("[tolerationRule=%d/%d]",
					j, len(restrictToleration)) // explain removal

				// stop checking pt against restricted tolerations
				break
			}

			track[i] = fmt.Sprintf("[tolerationRule=%d/%d podRule=%d/%d]",
				j, len(restrictToleration), podRule, len(rt.AllowedPods)) // explain acceptance
		}
	}

	// report tolerations removed
	for i, rem := range removed {
		tol := tolerationToString(podTolerations[i])
		trk := track[i]
		log.Printf("pod: %s/%s: toleration=%s: removed=%t %s",
			namespace, podName, tol, rem, trk)
	}

	return toRemove
}

func removeNodeSelectors(namespace, podName string, nodeSelector map[string]string, acceptSelectors []string) []string {
	var toRemove []string

	for removeKey := range nodeSelector {
		var accepted bool
		for _, acceptKey := range acceptSelectors {
			if removeKey == acceptKey {
				accepted = true
				break
			}
		}
		if !accepted {
			key := escapeJSONPointer(removeKey)
			toRemove = append(toRemove, fmt.Sprintf(`{"op":"remove","path":"/spec/nodeSelector/%s"}`, key))
		}
		log.Printf("pod: %s/%s: nodeSelector=%s: accepted=%t",
			namespace, podName, removeKey, accepted)
	}

	return toRemove
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
