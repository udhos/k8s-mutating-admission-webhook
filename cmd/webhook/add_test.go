package main

import (
	"encoding/json"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

type placePodsTestCase struct {
	testName          string
	rules             string
	namespace         string
	podName           string
	priorityClassName string
	priority          *int32
	podLabels         string
	containers        []corev1.Container
	expected          string
}

const placeRulesMissingMatch = `
place_pods:
  - add:
      node_selector:
        node: alpha
`

const placeRulesMatchAll1 = `
place_pods:
  - pods:
      - namespace: ""
    add:
      node_selector:
        node: alpha
`

const placeRulesMatchAll2 = `
place_pods:
  - pods:
      - namespace: ""
    add:
      tolerations:
        - key: key1
          operator: Equal
          value: value1
          effect: NoSchedule
`

const placeRulesMatchAll3 = `
place_pods:
  - pods:
      - namespace: ""
    add:
      tolerations:
        - key: key1
          operator: Equal
          value: value1
          effect: NoSchedule
      node_selector:
        node: alpha
`

const placeRulesColorLabels = `
place_pods:
  - pods:
      - labels:
          color: red
      - labels:
          color: blue
    add:
      node_selector:
        node: red-or-blue
  - pods:
      - labels:
          color: white
      - labels:
          color: black
    add:
      node_selector:
        node: white-or-black
`

const placeRulesHasJobLabel = `
place_pods:
  - pods:
      - labels:
          batch.kubernetes.io/job-name: "regexp="
    add:
      node_selector:
        nodepool: job
      tolerations:
        - key: nodepool
          operator: Equal
          value: job
          effect: NoSchedule
`

const placeRulesHasJobLabelValue = `
place_pods:
  - pods:
      - labels:
          batch.kubernetes.io/job-name: "regexp=^test$"
    add:
      node_selector:
        nodepool: job
      tolerations:
        - key: nodepool
          operator: Equal
          value: job
          effect: NoSchedule
`

const placeRulesEnv = `
place_pods:
  - pods:
      - namespace: ""
    add:
      containers:
        test-container:
            env:
            - name: ENV1
              value: VALUE1
            - name: MY_NODE_NAME
              valueFrom:
                fieldRef:
                    fieldPath: spec.nodeName
            - name: MY_CPU_REQUEST
              valueFrom:
                resourceFieldRef:
                    containerName: test-container
`

const placePriorityClass = `
place_pods:
  - pods:
      - has_priority_class_name: ^$ # match empty priority class name
        namespace: ""               # match any namespace
    add:
      priority_class_name: medium
  - pods:
      - has_priority_class_name: _reservation # exclude reservation class
        namespace: ""                         # match any namespace
    add:
      priority_class_name: low
`

var placePodsTestTable = []placePodsTestCase{
	{
		testName:  "empty rule",
		rules:     "",
		namespace: "default",
		podName:   "pod-1",
		podLabels: ``,
		expected:  "[]",
	},
	{
		testName:  "missing match rule",
		rules:     placeRulesMissingMatch,
		namespace: "default",
		podName:   "pod-1",
		podLabels: ``,
		expected:  "[]",
	},
	{
		testName:  "match all 1",
		rules:     placeRulesMatchAll1,
		namespace: "default",
		podName:   "pod-1",
		podLabels: ``,
		expected:  `[{"op":"add","path":"/spec/nodeSelector","value":{"node":"alpha"}}]`,
	},
	{
		testName:  "match all 2",
		rules:     placeRulesMatchAll2,
		namespace: "default",
		podName:   "pod-1",
		podLabels: ``,
		expected:  `[{"op":"add","path":"/spec/tolerations/-","value":{"key":"key1","operator":"Equal","effect":"NoSchedule","value":"value1"}}]`,
	},
	{
		testName:  "match all 3",
		rules:     placeRulesMatchAll3,
		namespace: "default",
		podName:   "pod-1",
		podLabels: ``,
		expected:  `[{"op":"add","path":"/spec/tolerations/-","value":{"key":"key1","operator":"Equal","effect":"NoSchedule","value":"value1"}} {"op":"add","path":"/spec/nodeSelector","value":{"node":"alpha"}}]`,
	},
	{
		testName:  "match color label 1",
		rules:     placeRulesColorLabels,
		namespace: "default",
		podName:   "pod-1",
		podLabels: `{"color":"red"}`,
		expected:  `[{"op":"add","path":"/spec/nodeSelector","value":{"node":"red-or-blue"}}]`,
	},
	{
		testName:  "match color label 2",
		rules:     placeRulesColorLabels,
		namespace: "default",
		podName:   "pod-1",
		podLabels: `{"color":"blue"}`,
		expected:  `[{"op":"add","path":"/spec/nodeSelector","value":{"node":"red-or-blue"}}]`,
	},
	{
		testName:  "match color label 3",
		rules:     placeRulesColorLabels,
		namespace: "default",
		podName:   "pod-1",
		podLabels: `{"color":"white"}`,
		expected:  `[{"op":"add","path":"/spec/nodeSelector","value":{"node":"white-or-black"}}]`,
	},
	{
		testName:  "match color label 4",
		rules:     placeRulesColorLabels,
		namespace: "default",
		podName:   "pod-1",
		podLabels: `{"color":"black"}`,
		expected:  `[{"op":"add","path":"/spec/nodeSelector","value":{"node":"white-or-black"}}]`,
	},
	{
		testName:  "match color label 5",
		rules:     placeRulesColorLabels,
		namespace: "default",
		podName:   "pod-1",
		podLabels: `{"color":"green"}`,
		expected:  `[]`,
	},
	{
		testName:  "it is job",
		rules:     placeRulesHasJobLabel,
		namespace: "default",
		podName:   "pod-1",
		podLabels: `{"batch.kubernetes.io/job-name":"anything"}`,
		expected:  `[{"op":"add","path":"/spec/tolerations/-","value":{"key":"nodepool","operator":"Equal","effect":"NoSchedule","value":"job"}} {"op":"add","path":"/spec/nodeSelector","value":{"nodepool":"job"}}]`,
	},
	{
		testName:  "not a job",
		rules:     placeRulesHasJobLabel,
		namespace: "default",
		podName:   "pod-1",
		podLabels: `{"not-job":"anything"}`,
		expected:  `[]`,
	},
	{
		testName:  "it is job with right label value",
		rules:     placeRulesHasJobLabelValue,
		namespace: "default",
		podName:   "pod-1",
		podLabels: `{"batch.kubernetes.io/job-name":"test"}`,
		expected:  `[{"op":"add","path":"/spec/tolerations/-","value":{"key":"nodepool","operator":"Equal","effect":"NoSchedule","value":"job"}} {"op":"add","path":"/spec/nodeSelector","value":{"nodepool":"job"}}]`,
	},
	{
		testName:  "it is job with wrong label value",
		rules:     placeRulesHasJobLabelValue,
		namespace: "default",
		podName:   "pod-1",
		podLabels: `{"batch.kubernetes.io/job-name":"test1"}`,
		expected:  `[]`,
	},
	{
		testName:  "add env to container with empty env",
		rules:     placeRulesEnv,
		namespace: "default",
		podName:   "pod-env-1",
		containers: []corev1.Container{
			{Name: "test-container"},
		},
		expected: `[{"op":"add","path":"/spec/containers/0/env","value":[]} {"op":"add","path":"/spec/containers/0/env/-","value":{"name":"ENV1","value":"VALUE1"}} {"op":"add","path":"/spec/containers/0/env/-","value":{"name":"MY_NODE_NAME","valueFrom":{"fieldRef":{"fieldPath":"spec.nodeName"}}}} {"op":"add","path":"/spec/containers/0/env/-","value":{"name":"MY_CPU_REQUEST","valueFrom":{"resourceFieldRef":{"containerName":"test-container"}}}}]`,
	},
	{
		testName:  "add env to container",
		rules:     placeRulesEnv,
		namespace: "default",
		podName:   "pod-env-1",
		containers: []corev1.Container{
			{
				Name: "test-container",
				Env:  []corev1.EnvVar{{Name: "KEY1", Value: "VAL1"}},
			},
		},
		expected: `[{"op":"add","path":"/spec/containers/0/env/-","value":{"name":"ENV1","value":"VALUE1"}} {"op":"add","path":"/spec/containers/0/env/-","value":{"name":"MY_NODE_NAME","valueFrom":{"fieldRef":{"fieldPath":"spec.nodeName"}}}} {"op":"add","path":"/spec/containers/0/env/-","value":{"name":"MY_CPU_REQUEST","valueFrom":{"resourceFieldRef":{"containerName":"test-container"}}}}]`,
	},
	{
		testName:  "add env to second container",
		rules:     placeRulesEnv,
		namespace: "default",
		podName:   "pod-env-2",
		containers: []corev1.Container{
			{Name: "first"},
			{
				Name: "test-container",
				Env:  []corev1.EnvVar{{Name: "KEY1", Value: "VAL1"}},
			},
		},
		expected: `[{"op":"add","path":"/spec/containers/1/env/-","value":{"name":"ENV1","value":"VALUE1"}} {"op":"add","path":"/spec/containers/1/env/-","value":{"name":"MY_NODE_NAME","valueFrom":{"fieldRef":{"fieldPath":"spec.nodeName"}}}} {"op":"add","path":"/spec/containers/1/env/-","value":{"name":"MY_CPU_REQUEST","valueFrom":{"resourceFieldRef":{"containerName":"test-container"}}}}]`,
	},
	{
		testName:  "add env to non-existing container",
		rules:     placeRulesEnv,
		namespace: "default",
		podName:   "pod-env-3",
		containers: []corev1.Container{
			{Name: "first"},
			{Name: "second "},
		},
		expected: `[]`,
	},
	{
		testName:          "empty priority class name",
		rules:             placePriorityClass,
		namespace:         "default",
		podName:           "pod-1",
		priorityClassName: "",
		expected:          `[{"op":"add","path":"/spec/priorityClassName","value":"medium"}]`,
	},
	{
		testName:          "reservation priority class name",
		rules:             placePriorityClass,
		namespace:         "default",
		podName:           "pod-1",
		priorityClassName: "reservation",
		expected:          "[]",
	},
	{
		testName:          "other priority class name",
		rules:             placePriorityClass,
		namespace:         "default",
		podName:           "pod-1",
		priorityClassName: "other",
		expected:          `[{"op":"add","path":"/spec/priorityClassName","value":"low"}]`,
	},
	{
		testName:          "existing priority should be removed",
		rules:             placePriorityClass,
		namespace:         "default",
		podName:           "pod-1",
		priorityClassName: "other",
		priority:          &priority,
		expected:          `[{"op":"add","path":"/spec/priorityClassName","value":"low"} {"op":"remove","path":"/spec/priority"}]`,
	},
}

var priority int32 = 500

// go test -count 1 -run '^TestPlacePods$' ./cmd/webhook
func TestPlacePods(t *testing.T) {

	for i, data := range placePodsTestTable {
		testLabel := fmt.Sprintf("%d: %s:", i, data.testName)

		r, errRule := newRules([]byte(data.rules))
		if errRule != nil {
			t.Errorf("%s bad rule: %v", testLabel, errRule)
		}

		var podLabels map[string]string
		if data.podLabels != "" {
			errLab := json.Unmarshal([]byte(data.podLabels), &podLabels)
			if errLab != nil {
				t.Errorf("%s bad pod labels: %v", testLabel, errLab)
			}
		}

		list := addPlacement(data.namespace, data.podName,
			data.priorityClassName, data.priority, podLabels, data.containers,
			r.PlacePods)

		result := fmt.Sprintf("%v", list)

		if result != data.expected {
			t.Errorf("%s\n==      got:'%s'\n== expected:'%s'",
				testLabel, result, data.expected)
		}
	}
}
