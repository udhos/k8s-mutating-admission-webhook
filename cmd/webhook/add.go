package main

import (
	"encoding/json"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
)

// addPlacement adds tolerations, nodeSelector, priorityClass, container env vars.
func addPlacement(namespace, podName, priorityClassName string,
	priority *int32,
	podLabels map[string]string,
	containers []corev1.Container,
	placePods []placementConfig) []string {

	//
	// scan pod add rules
	//
	for _, pc := range placePods {

		if pc.match(namespace, podName, priorityClassName, podLabels) {
			//
			// found add rule for pod
			//
			return addOne(namespace, podName, priorityClassName, priority,
				containers, pc.Add)
		}
	}

	return nil
}

func addOne(namespace, podName, priorityClassName string, priority *int32,
	containers []corev1.Container, add addConfig) []string {

	var list []string

	for _, tol := range add.Tolerations {
		list = append(list, addToleration(namespace, podName, tol))
	}

	if len(add.NodeSelector) > 0 {
		ns, errNs := addNodeSelector(namespace, podName, add.NodeSelector)
		if errNs != nil {
			log.Printf("ERROR: addOne: %v", errNs)
			return list
		}
		list = append(list, ns)
	}

	if len(add.Containers) > 0 {
		list = append(list, addContainerEnv(namespace, podName, containers,
			add.Containers)...)
	}

	if add.PriorityClassName != "" {
		list = append(list, setPriorityClass(namespace, podName,
			add.PriorityClassName, priorityClassName, priority)...)
	}

	return list
}

func setPriorityClass(namespace, podName, newClass, oldClass string, priority *int32) []string {
	var list []string

	var priorityStr string
	if priority != nil {
		priorityStr = fmt.Sprintf("%d", *priority)
	} else {
		priorityStr = "<undefined>"
	}

	log.Printf("setPriorityClass: ns=%s pod=%s oldClass='%s' newClass='%s' oldPriority=%s",
		namespace, podName, oldClass, newClass, priorityStr)

	// add or replace priorityClassName
	str := fmt.Sprintf(`{"op":"add","path":"/spec/priorityClassName","value":"%s"}`,
		newClass)
	list = append(list, str)

	// remove priority if set
	if priority != nil {
		str := `{"op":"remove","path":"/spec/priority"}`
		list = append(list, str)
	}

	return list
}

func addContainerEnv(namespace, podName string, containers []corev1.Container,
	addContainers map[string]containerConfig) []string {

	containerIndex := map[string]int{}
	containerInitialized := map[int]bool{}

	for i, c := range containers {
		containerIndex[c.Name] = i
	}

	var list []string

	for name, c := range addContainers {

		for _, env := range c.Env {
			i, found := containerIndex[name]
			if !found {
				log.Printf("ERROR: addContainerEnv: ns=%s pod=%s container not found: '%s'", namespace, podName, name)
				continue
			}
			envKey := env["name"]
			if envKey == nil {
				log.Printf("ERROR: addContainerEnv: ns=%s pod=%s container='%s' missing env name", namespace, podName, name)
				continue
			}
			envKeyStr, isStr := envKey.(string)
			if !isStr {
				log.Printf("ERROR: addContainerEnv: ns=%s pod=%s container='%s' bad env name type: name='%v' type=%T", namespace, podName, name, envKey, envKey)
				continue
			}
			value, errJSON := json.Marshal(env)
			if errJSON != nil {
				log.Printf("ERROR: addContainerEnv: ns=%s pod=%s container='%s' bad env json: name='%s' error=%v value=%v", namespace, podName, name, envKeyStr, errJSON, env)
				continue
			}
			valueStr := string(value)
			str := fmt.Sprintf(`{"op":"add","path":"/spec/containers/%d/env/-","value":%s}`, i, valueStr)

			log.Printf("addContainerEnv: %s/%s/%s(%d) adding env var name=%s entry=%v", namespace, podName, name, i, envKeyStr, valueStr)

			if len(containers[i].Env) == 0 {
				// need to create env array first
				if !containerInitialized[i] {
					list = append(list, fmt.Sprintf(`{"op":"add","path":"/spec/containers/%d/env","value":[]}`, i))
					containerInitialized[i] = true
				}
			}

			list = append(list, str)
		}
	}

	return list
}

func addToleration(namespace, podName string, tol tolerationConfig) string {
	log.Printf("addToleration: ns=%s pod=%s: %s", namespace, podName,
		tolerationFieldsToString(tol.Key, tol.Operator, tol.Value, tol.Effect))

	return fmt.Sprintf(`{"op":"add","path":"/spec/tolerations/-","value":{"key":"%s","operator":"%s","effect":"%s","value":"%s"}}`,
		tol.Key, tol.Operator, tol.Effect, tol.Value)
}

func addNodeSelector(namespace, podName string, nodeSelector map[string]string) (string, error) {

	value, errJSON := labelsToJSONString(nodeSelector)
	if errJSON != nil {
		return "", fmt.Errorf("addNodeSelector: %v", errJSON)
	}

	log.Printf("addNodeSelector: ns=%s pod=%s: %s", namespace, podName, value)

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
