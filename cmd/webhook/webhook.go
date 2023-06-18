package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func handlerRoute(app *config, w http.ResponseWriter, r *http.Request) {
	const me = "handlerRoute"
	log.Printf("%s: %s %s %s - ok",
		me, r.RemoteAddr, r.Method, r.RequestURI)

	deserializer := app.codecs.UniversalDeserializer()

	// Parse the AdmissionReview from the http request.
	admissionReviewRequest, errAr := admissionReviewFromRequest(r, deserializer)
	if errAr != nil {
		msg := fmt.Sprintf("%s: error getting admission review from request: %v",
			me, errAr)
		log.Printf(msg)
		http.Error(w, msg, 400)
		return
	}

	log.Printf("admissionReview: %v", admissionReviewRequest)
	log.Printf("request: %v", admissionReviewRequest.Request)

	if admissionReviewRequest.Request == nil {
		msg := fmt.Sprintf("%s: missing request in admission review", me)
		log.Printf(msg)
		http.Error(w, msg, 400)
		return
	}

	log.Printf("request resource: %v", admissionReviewRequest.Request.Resource)

	// Do server-side validation that we are only dealing with a pod resource. This
	// should also be part of the MutatingWebhookConfiguration in the cluster, but
	// we should verify here before continuing.
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if admissionReviewRequest.Request.Resource != podResource {
		msg := fmt.Sprintf("%s: did not receive pod, got %s",
			me, admissionReviewRequest.Request.Resource.Resource)
		log.Printf(msg)
		http.Error(w, msg, 400)
		return
	}

	// Decode the pod from the AdmissionReview.
	rawRequest := admissionReviewRequest.Request.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := deserializer.Decode(rawRequest, nil, &pod); err != nil {
		msg := fmt.Sprintf("%s: error decoding raw pod: %v",
			me, err)
		log.Printf(msg)
		http.Error(w, msg, 500)
		return
	}

	// Create a response that will add a label to the pod if it does
	// not already have a label with the key of "hello". In this case
	// it does not matter what the value is, as long as the key exists.
	admissionResponse := &admissionv1.AdmissionResponse{}
	var patch string
	patchType := admissionv1.PatchTypeJSONPatch
	if _, ok := pod.Labels["hello"]; !ok {
		patch = `[{"op":"add","path":"/metadata/labels","value":{"hello":"world"}}]`
	}

	admissionResponse.Allowed = true
	if patch != "" {
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
		log.Printf(msg)
		http.Error(w, msg, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func admissionReviewFromRequest(r *http.Request, deserializer runtime.Decoder) (*admissionv1.AdmissionReview, error) {
	// Validate that the incoming content type is correct.
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("expected application/json content-type")
	}

	// Get the body data, which will be the AdmissionReview
	// content for the request.
	var body []byte
	if r.Body != nil {
		requestData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = requestData
	}

	// Decode the request body into
	admissionReviewRequest := &admissionv1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, admissionReviewRequest); err != nil {
		return nil, err
	}

	return admissionReviewRequest, nil
}
