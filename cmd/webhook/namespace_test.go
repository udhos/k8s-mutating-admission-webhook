package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

type namespaceTestCase struct {
	testName string
	rules    string
	name     string
	labels   string
	expected string
}

var namespaceTestTable = []namespaceTestCase{
	{
		testName: "empty rule",
		rules:    "",
		name:     "default",
		expected: "[]",
	},
	{
		testName: "empty rule with labels",
		rules:    "",
		name:     "default",
		labels:   `{"a":"b","c":"d"}`,
		expected: "[]",
	},
	{
		testName: "match any namespace",
		rules:    nsMatchAnyNamespace,
		name:     "default",
		expected: `[{"op":"add","path":"/metadata/labels","value":{"istio-injection":"enabled"}}]`,
	},
	{
		testName: "match any with labels",
		rules:    nsMatchAnyNamespace,
		name:     "default",
		labels:   `{"a":"b","c":"d"}`,
		expected: `[{"op":"add","path":"/metadata/labels","value":{"a":"b","c":"d","istio-injection":"enabled"}}]`,
	},
	{
		testName: "match none",
		rules:    nsMatchNoNamespace,
		name:     "default",
		expected: `[]`,
	},
	{
		testName: "match by name",
		rules:    nsMatchName,
		name:     "special",
		expected: `[{"op":"add","path":"/metadata/labels","value":{"istio-injection":"special"}}]`,
	},
	{
		testName: "mismatch by name",
		rules:    nsMatchName,
		name:     "none",
		expected: `[]`,
	},
	{
		testName: "match by name regexp",
		rules:    nsMatchNameRegexp,
		name:     "special",
		expected: `[{"op":"add","path":"/metadata/labels","value":{"istio-injection":"special"}}]`,
	},
}

const nsMatchAnyNamespace = `
rules:
- namespaces_add_labels:
  - name: ""
    add_labels:
      istio-injection: enabled
`

const nsMatchNoNamespace = `
rules:
- namespaces_add_labels:
  - name: _
    add_labels:
      istio-injection: enabled
`

const nsMatchName = `
rules:
- namespaces_add_labels:
  - name: default
    add_labels:
      istio-injection: enabled
  - name: special
    add_labels:
      istio-injection: special
`

const nsMatchNameRegexp = `
rules:
- namespaces_add_labels:
  - name: default
    add_labels:
      istio-injection: enabled
  - name: ^special$
    add_labels:
      istio-injection: special
`

// go test -count=1 -run TestNamespace ./...
func TestNamespace(t *testing.T) {

	for i, data := range namespaceTestTable {
		name := fmt.Sprintf("%d of %d:%s:",
			i, len(namespaceTestTable), data.testName)

		t.Run(name, func(t *testing.T) {

			ruleList, errRule := newRules([]byte(data.rules))
			if errRule != nil {
				t.Errorf("bad rule: %v", errRule)
			}

			var labels map[string]string
			if data.labels != "" {
				errLab := json.Unmarshal([]byte(data.labels), &labels)
				if errLab != nil {
					t.Errorf("bad pod labels: %v", errLab)
				}
			}

			var r rulesConfig
			if data.rules != "" {
				if len(ruleList.Rules) != 1 {
					t.Fatalf("bad number of rules (should be 1): %d",
						len(ruleList.Rules))
					return
				}
				r = ruleList.Rules[0]
			}

			list := namespaceAddLabels(data.name, labels, r.NamespacesAddLabels)

			result := fmt.Sprintf("%v", list)

			if result != data.expected {
				t.Errorf("got='%s' expected='%s' rule=%v",
					result, data.expected, r)
			}
		})

	}
}
