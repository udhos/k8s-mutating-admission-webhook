package main

import (
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
)

func removeTolerations(namespace, podName string, podLabels map[string]string, podTolerations []corev1.Toleration,
	restrictToleration []restrictTolerationConfig) []string {

	toRemove := removeTolerationsIndices(namespace, podName, podLabels, podTolerations,
		restrictToleration)

	// build patch list removing all tolerations by index
	list := make([]string, 0, len(toRemove))
	for _, i := range toRemove {
		list = append(list, fmt.Sprintf(`{"op":"remove","path":"/spec/tolerations/%d"}`, i))
	}
	return list
}

func removeTolerationsIndices(namespace, podName string, podLabels map[string]string, podTolerations []corev1.Toleration,
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
				if allowedPod.match(namespace, podName, podLabels) {
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
