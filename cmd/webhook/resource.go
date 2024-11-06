package main

import (
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	api_resource "k8s.io/apimachinery/pkg/api/resource"
)

func addResource(namespace, podName string, podLabels map[string]string,
	containers []corev1.Container, resources []setResource) []string {

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

			log.Printf("DEBUG %s: rule=%d/%d namespace=%s pod=%s container=%s resources=%v",
				me, i+1, len(resources), namespace, podName, c.Name, r)

			list = generateResource(
				c.Resources.Requests.Cpu(), i, "requests", "cpu",
				r.CPU.Request, list)

			list = generateResource(
				c.Resources.Limits.Cpu(), i, "limits", "cpu",
				r.CPU.Limit, list)

			list = generateResource(
				c.Resources.Requests.Memory(), i, "requests", "memory",
				r.Memory.Request, list)

			list = generateResource(
				c.Resources.Limits.Memory(), i, "limits", "memory",
				r.Memory.Limit, list)

			list = generateResource(
				c.Resources.Requests.StorageEphemeral(), i,
				"requests", "ephemeral-storage",
				r.EphemeralStorage.Request, list)

			list = generateResource(
				c.Resources.Limits.StorageEphemeral(), i,
				"limits", "ephemeral-storage",
				r.EphemeralStorage.Limit, list)
		}
	}

	log.Printf("DEBUG %s: %v", me, list)

	return list
}

func generateResource(q *api_resource.Quantity, i int, reqLim, name, value string, list []string) []string {
	if !q.IsZero() {
		return list // already defined, do not change it
	}
	if value == "" {
		return list // rule did not define it, skip it
	}

	const templ = `{"op":"add","path":"/spec/containers/%d/resources/%s","value":{"%s":"%s"}}`
	list = append(list, fmt.Sprintf(templ, i, reqLim, name, value))

	return list
}
