package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

type daemonsetTestCase struct {
	name      string
	rules     string
	namespace string
	dsName    string
	dsLabels  string
	expected  string
}

var daemonsetTestTable = []daemonsetTestCase{
	{
		name:      "empty rule",
		rules:     "",
		namespace: "default",
		dsName:    "ds1",
		dsLabels:  ``,
		expected:  "[]",
	},
	{
		name:      "match any daemonset",
		rules:     matchAnyDaemonset,
		namespace: "default",
		dsName:    "ds1",
		dsLabels:  ``,
		expected:  `[{"op":"add","path":"/spec/template/spec/nodeSelector","value":{"node":"alpha"}}]`,
	},
	{
		name:      "match none",
		rules:     matchNoDaemonset,
		namespace: "default",
		dsName:    "ds1",
		dsLabels:  ``,
		expected:  `[]`,
	},
	{
		name:      "put default node selector",
		rules:     matchAnyDaemonsetPutDefaultNS,
		namespace: "default",
		dsName:    "ds1",
		dsLabels:  ``,
		expected:  `[{"op":"add","path":"/spec/template/spec/nodeSelector","value":{"non-existing":"true"}}]`,
	},
	{
		name:      "match by name",
		rules:     matchName,
		namespace: "default",
		dsName:    "ds2",
		dsLabels:  ``,
		expected:  `[{"op":"add","path":"/spec/template/spec/nodeSelector","value":{"non-existing":"true"}}]`,
	},
	{
		name:      "mismatch by name",
		rules:     matchName,
		namespace: "default",
		dsName:    "ds1",
		dsLabels:  ``,
		expected:  `[]`,
	},
	{
		name:      "match by name regexp",
		rules:     matchNameRegexp,
		namespace: "default",
		dsName:    "ds2",
		dsLabels:  ``,
		expected:  `[{"op":"add","path":"/spec/template/spec/nodeSelector","value":{"non-existing":"true"}}]`,
	},
}

const matchAnyDaemonset = `
rules:
- disable_daemonsets:
  - namespace: ""
    name: ""
    #labels:
    #  a: b
    node_selector:
      node: alpha
`

const matchNoDaemonset = `
rules:
- disable_daemonsets:
  - namespace: _
    name: ""
    #labels:
    #  a: b
    node_selector:
      node: alpha
`

const matchAnyDaemonsetPutDefaultNS = `
rules:
- disable_daemonsets:
  - namespace: ""
    name: ""
    #labels:
    #  a: b
`

const matchName = `
rules:
- disable_daemonsets:
  - namespace: ""
    name: ds2
    #labels:
    #  a: b
`

const matchNameRegexp = `
rules:
- disable_daemonsets:
  - namespace: ""
    name: ^ds2$
    #labels:
    #  a: b
`

func TestDaemonset(t *testing.T) {

	for i, data := range daemonsetTestTable {
		name := fmt.Sprintf("%d of %d:%s:",
			i, len(daemonsetTestTable), data.name)

		t.Run(name, func(t *testing.T) {

			ruleList, errRule := newRules([]byte(data.rules))
			if errRule != nil {
				t.Errorf("bad rule: %v", errRule)
			}

			var dsLabels map[string]string
			if data.dsLabels != "" {
				errLab := json.Unmarshal([]byte(data.dsLabels), &dsLabels)
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

			list := daemonsetNodeSelector(data.namespace, data.dsName, dsLabels, r.DisableDaemonsets)

			result := fmt.Sprintf("%v", list)

			if result != data.expected {
				t.Errorf("got='%s' expected='%s'", result, data.expected)
			}
		})

	}
}
