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
