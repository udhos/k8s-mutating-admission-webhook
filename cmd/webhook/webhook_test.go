package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

type tolerationTestCase struct {
	testName          string
	rules             string
	podTolerations    string
	namespace         string
	podName           string
	priorityClassName string
	podLabels         string
	expectedIndices   string
}

const emptyRule = ``
const emptyTolerations = `[]`

const tolerations1 = `[{"key":"key1","operator":"Equal","value":"value1","effect":"NoSchedule"}]`
const tolerations3 = `[
    {"key":"key1","operator":"Equal","value":"value1","effect":"NoSchedule"},
    {"key":"key2","operator":"Equal","value":"value2","effect":"NoSchedule"},
    {"key":"key3","operator":"Equal","value":"value3","effect":"NoSchedule"}
    ]`
const tolerationsExists = `[
        {"key":"key1","operator":"Equal","value":"value1","effect":"NoSchedule"},
        {"operator":"Exists"},
        {"key":"key3","operator":"Exists"},
        {"operator":"Exists","value":"value3"},
        {"operator":"Exists","effect":"NoSchedule"}
        ]`

const rulesRejectKey2 = `
rules:
- restrict_tolerations:
    - toleration:
        # match key2
        key: ^key2$
        #operator: ""  # empty string matches anything
        #value: ""     # empty string matches anything
        #effect: ""    # empty string matches anything
      allowed_pods:
        # match NO pod
        - namespace: _ # negated empty string matches nothing
          name: _      # negated empty string matches nothing
`

const rulesRejectAll = `
rules:
- restrict_tolerations:
  - toleration:
      # match any toleration
      #key: ""        # empty string matches anything
      #operator: ""   # empty string matches anything
      #value: ""      # empty string matches anything
      #effect: ""     # empty string matches anything
    allowed_pods:
      # match NO pod
      - namespace: _  # negated empty string matches nothing
        name: _       # negated empty string matches nothing
`

const rulesRejectOnlyExists = `
rules:
- restrict_tolerations:
  - toleration:
      # match only Exists
      key: ^$               # match only the empty string
      operator: ^Exists$
      value: ^$             # match only the empty string
      effect: ^$            # match only the empty string
    allowed_pods:
      # match NO pod
      - namespace: _        # negated empty string matches nothing
        name: _             # negated empty string matches nothing
`

const rulesOnlyPodDaemonSetCanHaveExactlyExists = `
rules:
- restrict_tolerations:
  - toleration:
      # match only Exists
      key: ^$               # match only the empty string
      operator: ^Exists$
      value: ^$             # match only the empty string
      effect: ^$            # match only the empty string
    allowed_pods:
      # this first rule does nothing, it serves only to test multiple pod rules
      - namespace: _        # negated empty string matches nothing
        name: _             # negated empty string matches nothing
      # match only POD prefixed as daemonset-
      - #namespace: ""      # empty string matches anything
        name: ^daemonset-   # match prefix
`

const rulesOnlyPodDaemonSetCanHaveExactlyExistsWithAnd = `
rules:
- restrict_tolerations:
  - toleration:
      # match only Exists
      key: ^$               # match only the empty string
      operator: ^Exists$
      value: ^$             # match only the empty string
      effect: ^$            # match only the empty string
    allowed_pods:
      # this first rule does nothing, it serves only to test multiple pod rules
      - namespace: _        # negated empty string matches nothing
        name: _             # negated empty string matches nothing
      - and:
        # match only POD prefixed as datadog-
        - namespace: ^datadog$    # empty string matches anything
          name: ^datadog-         # match prefix
        # AND NOT prefixed as datadog-agent-
        - namespace: ^datadog$    # empty string matches anything
          name: _^datadog-agent-  # reject prefix
`

const rulesPodLabelCanHaveKey2 = `
rules:
- restrict_tolerations:
    - toleration:
        # match key2
        key: ^key2$
        #operator: ""  # empty string matches anything
        #value: ""     # empty string matches anything
        #effect: ""    # empty string matches anything
      allowed_pods:
        # match pods with label good=pod
        - labels:
            good: pod
`

var tolerationTestTable = []tolerationTestCase{
	{
		testName:        "empty rule, empty toleration",
		rules:           emptyRule,
		podTolerations:  emptyTolerations,
		namespace:       "default",
		podName:         "pod-1",
		expectedIndices: "[]",
	},
	{
		testName:        "empty rule, one toleration",
		rules:           emptyRule,
		podTolerations:  tolerations1,
		namespace:       "default",
		podName:         "pod-1",
		expectedIndices: "[]",
	},
	{
		testName:        "rule rejects all tolerations, one toleration",
		rules:           rulesRejectAll,
		podTolerations:  tolerations1,
		namespace:       "default",
		podName:         "pod-1",
		expectedIndices: "[0]",
	},
	{
		testName:        "rule rejects all tolerations, three tolerations",
		rules:           rulesRejectAll,
		podTolerations:  tolerations3,
		namespace:       "default",
		podName:         "pod-1",
		expectedIndices: "[2 1 0]",
	},
	{
		testName:        "rule rejects key2, three toleration",
		rules:           rulesRejectKey2,
		podTolerations:  tolerations3,
		namespace:       "default",
		podName:         "pod-1",
		expectedIndices: "[1]",
	},
	{
		testName:        "rule rejects key2, three toleration",
		rules:           rulesRejectKey2,
		podTolerations:  tolerations3,
		namespace:       "default",
		podName:         "!",
		expectedIndices: "[1]",
	},
	{
		testName:        "no pod can have exactly Exists",
		rules:           rulesRejectOnlyExists,
		podTolerations:  tolerationsExists,
		namespace:       "default",
		podName:         "pod-1",
		expectedIndices: "[1]",
	},
	{
		testName:        "only daemonset- prefixed pod can have exactly Exists, no daemonset",
		rules:           rulesOnlyPodDaemonSetCanHaveExactlyExists,
		podTolerations:  tolerationsExists,
		namespace:       "default",
		podName:         "pod-1",
		expectedIndices: "[1]",
	},
	{
		testName:        "only daemonset- prefixed pod can have exactly Exists, have daemonset",
		rules:           rulesOnlyPodDaemonSetCanHaveExactlyExists,
		podTolerations:  tolerationsExists,
		namespace:       "default",
		podName:         "daemonset-1",
		expectedIndices: "[]",
	},
	{
		testName:        "accept Exists for datadog- prefixed pod",
		rules:           rulesOnlyPodDaemonSetCanHaveExactlyExistsWithAnd,
		podTolerations:  tolerationsExists,
		namespace:       "datadog",
		podName:         "datadog-",
		expectedIndices: "[]",
	},
	{
		testName:        "reject Exists for datadog-agent- prefixed pod",
		rules:           rulesOnlyPodDaemonSetCanHaveExactlyExistsWithAnd,
		podTolerations:  tolerationsExists,
		namespace:       "datadog",
		podName:         "datadog-agent-",
		expectedIndices: "[1]",
	},
	{
		testName:        "only key3 is restricted, pod label allows it",
		rules:           rulesPodLabelCanHaveKey2,
		podTolerations:  tolerations3,
		namespace:       "default",
		podName:         "pod-good-1",
		podLabels:       `{"good":"pod"}`,
		expectedIndices: "[]",
	},
	{
		testName:        "only key3 is restricted, pod has not label",
		rules:           rulesPodLabelCanHaveKey2,
		podTolerations:  tolerations3,
		namespace:       "default",
		podName:         "pod-good-1",
		expectedIndices: "[1]",
	},
	{
		testName:        "only key3 is restricted, pod label wrong",
		rules:           rulesPodLabelCanHaveKey2,
		podTolerations:  tolerations3,
		namespace:       "default",
		podName:         "pod-good-1",
		podLabels:       `{"bad":"news"}`,
		expectedIndices: "[1]",
	},
	{
		testName:        "only key3 is restricted, pod label wrong value",
		rules:           rulesPodLabelCanHaveKey2,
		podTolerations:  tolerations3,
		namespace:       "default",
		podName:         "pod-good-1",
		podLabels:       `{"good":"POD"}`,
		expectedIndices: "[1]",
	},
}

func TestRestrictTolerations(t *testing.T) {

	for i, data := range tolerationTestTable {
		testLabel := fmt.Sprintf("%d: %s:", i, data.testName)

		ruleList, errRule := newRules([]byte(data.rules))
		if errRule != nil {
			t.Errorf("%s bad rule: %v", testLabel, errRule)
		}

		var podTolerations []corev1.Toleration
		errTol := json.Unmarshal([]byte(data.podTolerations), &podTolerations)
		if errTol != nil {
			t.Errorf("%s bad pod tolerations: %v", testLabel, errTol)
		}

		var podLabels map[string]string
		if data.podLabels != "" {
			errLab := json.Unmarshal([]byte(data.podLabels), &podLabels)
			if errLab != nil {
				t.Errorf("%s bad pod labels: %v", testLabel, errLab)
			}
		}

		var r rulesConfig

		if data.rules != "" {
			if len(ruleList.Rules) != 1 {
				t.Fatalf("%s bad number of rules (should be 1): %d",
					testLabel, len(ruleList.Rules))
				continue
			}
			r = ruleList.Rules[0]
		}

		list := removeTolerationsIndices(data.namespace, data.podName,
			data.priorityClassName, podLabels, podTolerations,
			r.RestrictTolerations)

		str := fmt.Sprintf("%v", list)

		if str != data.expectedIndices {
			t.Errorf("%s bad removal indices: got=%s expected=%s",
				testLabel, str, data.expectedIndices)
		}

	}
}

func TestRemoveNodeSelector(t *testing.T) {

	nodeSelector := map[string]string{
		"a":        "1",
		"b":        "2",
		"c/x":      "3",
		"d":        "4",
		"foo/bar~": "5",
	}

	acceptNodeSelectors := []string{"b", "d"}

	expected := `{"op":"remove","path":"/spec/nodeSelector/a"},{"op":"remove","path":"/spec/nodeSelector/c~1x"},{"op":"remove","path":"/spec/nodeSelector/foo~1bar~0"}`

	list := removeNodeSelectors("namespace", "podname", nodeSelector, acceptNodeSelectors)

	slices.Sort(list)

	result := strings.Join(list, ",")

	if result != expected {
		t.Errorf("result:%s mismatched expected:%s", result, expected)
	}
}
