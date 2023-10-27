package main

import (
	"slices"
	"strings"
	"testing"
)

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
