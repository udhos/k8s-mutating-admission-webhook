package main

import (
	"strings"
	"testing"
)

func TestRemoveNodeSelector(t *testing.T) {

	nodeSelector := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
		"d": "4",
	}

	acceptNodeSelectors := []string{"b", "d"}

	expected := `{"op":"remove","path":"/spec/nodeSelector/a"},{"op":"remove","path":"/spec/nodeSelector/c"}`

	list := removeNodeSelectors("namespace", "podname", nodeSelector, acceptNodeSelectors)

	result := strings.Join(list, ",")

	if result != expected {
		t.Errorf("reulst:%s mismatched expected:%s", result, expected)
	}
}
