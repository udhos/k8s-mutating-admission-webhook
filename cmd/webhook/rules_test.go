package main

import (
	"encoding/json"
	"testing"
)

type labelTestCase struct {
	existing string
	required string
	expected bool
}

var labelTestTable = []labelTestCase{
	{"{}", "{}", true},
	{`{}`, `{"a":"b"}`, false},
	{`{"a":"b"}`, "{}", true},
	{`{"a":"b"}`, `{"a":"b"}`, true},
	{`{"a":"b"}`, `{"a":"b","c":"d"}`, false},
	{`{"a":"b"}`, `{"a":"c"}`, false},
	{`{"a":"b"}`, `{"c":"b"}`, false},
	{`{"a":"b"}`, `{"x":"x"}`, false},
	{`{"a":"b","c":"d"}`, `{}`, true},
	{`{"a":"b","c":"d"}`, `{"a":"b"}`, true},
	{`{"a":"b","c":"d"}`, `{"c":"d"}`, true},
	{`{"a":"b","c":"d"}`, `{"e":"f"}`, false},
	{`{"a":"b","c":"d"}`, `{"a":"b","c":"d"}`, true},
	{`{"a":"b","c":"d"}`, `{"e":"f","g":"h"}`, false},
	{`{"a":"b","c":"d"}`, `{"a":"b","c":"d","e":"f"}`, false},
	{`{"a":"b","c":"d"}`, `{"e":"f","g":"h","i":"j"}`, false},
}

func TestLabels(t *testing.T) {
	for i, data := range labelTestTable {
		var labExisting map[string]string
		if errU := json.Unmarshal([]byte(data.existing), &labExisting); errU != nil {
			t.Errorf("%d: bad existing: '%s': %v", i, data.existing, errU)
		}

		var labRequired map[string]string
		if errU := json.Unmarshal([]byte(data.required), &labRequired); errU != nil {
			t.Errorf("%d: bad required: '%s': %v", i, data.required, errU)
		}

		if result := hasLabels(labExisting, labRequired); result != data.expected {
			t.Errorf("%d: got=%t expected: %t", i, result, data.expected)
		}
	}
}

func TestRulesPlacePods(t *testing.T) {

	const input = `
rules:
- place_pods:
  - pods:
      - namespace: ""
    add:
      node_selector:
        node: alpha
`

	list, errInput := newRules([]byte(input))
	if errInput != nil {
		t.Errorf("input: %v", errInput)
	}

	if len(list.Rules) != 1 {
		t.Fatalf("bad number of rules (should be 1): %d",
			len(list.Rules))
	}

	r := list.Rules[0]

	if len(r.PlacePods) != 1 {
		t.Errorf("bad number of place pods rules (should be 1): %d",
			len(r.PlacePods))
	}

	if len(r.PlacePods[0].Pods) != 1 {
		t.Errorf("bad number of place pods match rules (should be 1): %d",
			len(r.PlacePods[0].Pods))
	}

	if len(r.PlacePods[0].Add.NodeSelector) != 1 {
		t.Errorf("bad number of node selector labels (should be 1): %d",
			len(r.PlacePods[0].Add.NodeSelector))
	}
}
