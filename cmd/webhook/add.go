package main

import (
	"encoding/json"
	"fmt"
	"log"
)

func addPlacement(namespace string, podName string, podLabels map[string]string, placePods []placementConfig) []string {
	//
	// scan pod add rules
	//
	for _, pc := range placePods {
		if pc.match(namespace, podName, podLabels) {
			//
			// found add rule for pod
			//
			return addOne(pc.Add)
		}
	}

	return nil
}

func addOne(add addConfig) []string {

	var list []string

	for _, tol := range add.Tolerations {
		list = append(list, addToleration(tol))
	}

	ns, errNs := addNodeSelector(add.NodeSelector)
	if errNs != nil {
		log.Printf("ERROR: addOne: %v", errNs)
		return list
	}

	list = append(list, ns)

	return list
}

func addToleration(tol tolerationConfig) string {
	return fmt.Sprintf(`{"op":"add","path":"/spec/tolerations/-","value":{"key":"%s","operator":"%s","effect":"%s","value":"%s"}}`,
		tol.Key, tol.Operator, tol.Effect, tol.Value)
}

func addNodeSelector(nodeSelector map[string]string) (string, error) {
	value, errJSON := labelsToJSONString(nodeSelector)
	if errJSON != nil {
		return "", fmt.Errorf("addNodeSelector: %v", errJSON)
	}
	return fmt.Sprintf(`{"op":"add","path":"/spec/nodeSelector","value":%s}`,
		value), nil
}

func labelsToJSONString(v map[string]string) (string, error) {
	data, errLabels := json.Marshal(v)
	if errLabels != nil {
		return "", fmt.Errorf("labelsToJSONString: input:%v error:%v", v, errLabels)
	}
	return string(data), nil
}
