package main

import (
	"encoding/json"
	"fmt"
)

func addPlacement(namespace string, podName string, placePods []placementConfig) []string {
	var list []string

	//
	// scan pod add rules
	//
	for _, pc := range placePods {
		if pc.Pod.match(namespace, podName) {
			//
			// found add rule for pod
			//
			list = append(list, addOne(pc.Add)...)
		}
	}

	return list
}

func addOne(add addConfig) []string {

	var list []string

	for _, tol := range add.Tolerations {
		list = append(list, addToleration(tol))
	}

	list = append(list, addNodeSelector(add.NodeSelector))

	return list
}

func addToleration(tol tolerationConfig) string {
	return fmt.Sprintf(`{"op":"add","path":"/spec/tolerations/-","value":{"key":"%s","operator":"%s","effect":"%s","value":"%s"}`,
		tol.Key, tol.Operator, tol.Effect, tol.Value)
}

func addNodeSelector(nodeSelector map[string]string) string {
	return fmt.Sprintf(`{"op":"add","path":"/spec/nodeSelector","value":%s}`,
		labelsToJSONString(nodeSelector))
}

func labelsToJSONString(v map[string]string) string {
	data, _ := json.Marshal(v)
	return string(data)
}
