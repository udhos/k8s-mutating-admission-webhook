package main

import (
	"fmt"
	"log"
	"maps"
)

func namespaceAddLabels(name string, labels map[string]string, addLabels []nsAddLabels) []string {

	me := fmt.Sprintf("namespaceAddLabels: namespace=%s", name)

	//
	// scan namespace rules
	//
	for _, add := range addLabels {

		if !add.match(name) {
			log.Printf("%s: skipped (no rule found)", me)
			continue
		}

		//
		// found rule for namespace
		//

		lab := map[string]string{}
		maps.Copy(lab, labels)
		maps.Copy(lab, add.AddLabels)

		return addLabelsToNs(me, name, labels, add.AddLabels, lab)
	}

	return nil
}

func addLabelsToNs(caller, name string, existing, add, result map[string]string) []string {

	log.Printf("%s: labels: existing=%v adding=%v result=%v",
		caller, existing, add, result)

	var list []string
	ns, errAdd := addLabelsOnMetadata(result)
	if errAdd != nil {
		log.Printf("ERROR: %s: %v", caller, errAdd)
		return list
	}
	list = append(list, ns)
	return list
}

func addLabelsOnMetadata(add map[string]string) (string, error) {
	value, errJSON := labelsToJSONString(add)
	if errJSON != nil {
		return "", fmt.Errorf("addLabelsOnMetadata: %v", errJSON)
	}
	return fmt.Sprintf(`{"op":"add","path":"/metadata/labels","value":%s}`,
		value), nil
}
