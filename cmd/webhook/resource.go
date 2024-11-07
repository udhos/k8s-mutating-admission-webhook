package main

import (
	"encoding/json"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	api_resource "k8s.io/apimachinery/pkg/api/resource"
)

func addResource(namespace, podName string, podLabels map[string]string,
	containers []corev1.Container, resources []setResource,
	debug bool) []string {

	const me = "addResource"

	var list []string

	//
	// scan resource rules
	//
	for _, r := range resources {
		if !r.Pod.match(namespace, podName, podLabels) {
			continue
		}
		// found pod
		for i, c := range containers {
			if !r.container.matchString(c.Name) {
				continue
			}
			// found container

			if debug {
				log.Printf("DEBUG %s: rule=%d/%d namespace=%s pod=%s container=%s resources=%v",
					me, i+1, len(resources), namespace, podName, c.Name, r)
			}

			origReqCPU := quantityValue(c.Resources.Requests.Cpu())
			origReqMem := quantityValue(c.Resources.Requests.Memory())
			origReqES := quantityValue(c.Resources.Requests.StorageEphemeral())

			origLimCPU := quantityValue(c.Resources.Limits.Cpu())
			origLimMem := quantityValue(c.Resources.Limits.Memory())
			origLimES := quantityValue(c.Resources.Limits.StorageEphemeral())

			// derive request from: config req, config limit, rule
			reqCPU := derive(origReqCPU, origLimCPU, r.CPU.Request)
			reqMem := derive(origReqMem, origLimMem, r.Memory.Request)
			reqES := derive(origReqES, origLimES, r.EphemeralStorage.Request)

			// derive limit from: config lim, config req, rule
			limCPU := derive(origLimCPU, origReqCPU, r.CPU.Limit)
			limMem := derive(origLimMem, origReqMem, r.Memory.Limit)
			limES := derive(origLimES, origReqES, r.EphemeralStorage.Limit)

			var changes []string
			recordChange(&changes, reqCPU, origReqCPU, "requests", "cpu")
			recordChange(&changes, reqMem, origReqMem, "requests", "memory")
			recordChange(&changes, reqES, origReqES, "requests", "ephemeral-storage")
			recordChange(&changes, limCPU, origLimCPU, "limits", "cpu")
			recordChange(&changes, limMem, origLimMem, "limits", "memory")
			recordChange(&changes, limES, origLimES, "limits", "ephemeral-storage")

			log.Printf("%s: %s/%s/%s(%d): changes: %q",
				me, namespace, podName, c.Name, i, changes)

			if len(changes) == 0 {
				return list
			}

			requests := map[string]string{}
			if reqCPU != "" {
				requests["cpu"] = reqCPU
			}
			if reqMem != "" {
				requests["memory"] = reqMem
			}
			if reqES != "" {
				requests["ephemeral-storage"] = reqES
			}

			limits := map[string]string{}
			if limCPU != "" {
				limits["cpu"] = limCPU
			}
			if limMem != "" {
				limits["memory"] = limMem
			}
			if limES != "" {
				limits["ephemeral-storage"] = limES
			}

			if debug {
				log.Printf("DEBUG %s: %s/%s/%s(%d): setting: requests=%#v limits=%#v",
					me, namespace, podName, c.Name, i, requests, limits)
			}

			req := generateResource(i, "requests", requests)
			if req != "" {
				list = append(list, req)
			}

			lim := generateResource(i, "limits", limits)
			if lim != "" {
				list = append(list, lim)
			}
		}
	}

	if debug {
		log.Printf("DEBUG %s: %v", me, list)
	}

	return list
}

// recordChange(&changes, reqCPU, origReqCPU, "requests", "cpu")
func recordChange(changes *[]string, value, origValue, reqLim, name string) {
	if value == origValue {
		return
	}
	*changes = append(*changes, fmt.Sprintf("%s.%s:(old='%s',new='%s')",
		reqLim, name, origValue, value))
}

func quantityValue(q *api_resource.Quantity) string {
	if q.IsZero() {
		return ""
	}
	return q.String()
}

func derive(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func generateResource(i int, reqLim string, value map[string]string) string {
	data, errJSON := json.Marshal(value)
	if errJSON != nil {
		log.Printf("ERROR: generateResource: json: %v", errJSON)
		return ""
	}

	const templ = `{"op":"replace","path":"/spec/containers/%d/resources/%s","value":%s}`

	return fmt.Sprintf(templ, i, reqLim, string(data))
}
