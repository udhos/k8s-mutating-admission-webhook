package main

import (
	"fmt"
	"log"
)

func daemonsetNodeSelector(namespace string, dsName string,
	dsLabels map[string]string,
	disableDaemonsets []selectDaemonset) []string {

	const me = "daemonsetNodeSelector"

	//
	// scan daemonset rules
	//
	for _, ds := range disableDaemonsets {

		if !ds.match(namespace, dsName, dsLabels) {
			log.Printf("%s: %s/%s labels=%v: skipped", me, namespace, dsName, dsLabels)
			continue
		}

		//
		// found rule for daemonset
		//

		if len(ds.NodeSelector) > 0 {
			//
			// add configured node selector
			//
			return disable(me, "custom", namespace, dsName, dsLabels, ds.NodeSelector)
		}

		//
		// add default node selector
		//
		return disable(me, "default", namespace, dsName, dsLabels, map[string]string{"non-existing": "true"})
	}

	return nil
}

func disable(caller, label, namespace, dsName string, dsLabels, nodeSelector map[string]string) []string {
	log.Printf("%s: %s/%s labels=%v: disabling with %s nodeSelector=%v",
		caller, namespace, dsName, dsLabels, label, nodeSelector)

	var list []string
	ns, errNs := addNodeSelectorOnTemplate(nodeSelector)
	if errNs != nil {
		log.Printf("ERROR: %s %s: %v", caller, label, errNs)
		return list
	}
	list = append(list, ns)
	return list
}

func addNodeSelectorOnTemplate(nodeSelector map[string]string) (string, error) {
	value, errJSON := labelsToJSONString(nodeSelector)
	if errJSON != nil {
		return "", fmt.Errorf("addNodeSelectorOnTemplate: %v", errJSON)
	}
	return fmt.Sprintf(`{"op":"add","path":"/spec/template/spec/nodeSelector","value":%s}`,
		value), nil
}
