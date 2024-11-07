package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	api_resource "k8s.io/apimachinery/pkg/api/resource"
)

type resourceTestCase struct {
	name string

	namespace  string
	podName    string
	podLabels  map[string]string
	containers []v1.Container

	rules string

	expectRequests map[string]string
	expectLimits   map[string]string
}

var resourceTestTable = []resourceTestCase{
	{
		name:      "empty rule does not affect resources",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []v1.Container{
			{
				Name: "container1",
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						"cpu":               api_resource.MustParse("55m"),
						"memory":            api_resource.MustParse("11M"),
						"ephemeral-storage": api_resource.MustParse("222M"),
					},
					Limits: v1.ResourceList{
						"cpu":               api_resource.MustParse("111m"),
						"memory":            api_resource.MustParse("22M"),
						"ephemeral-storage": api_resource.MustParse("333M"),
					},
				},
			},
		},
		rules:          "",
		expectRequests: nil,
		expectLimits:   nil,
	},
	{
		name:      "match-NOTHING rule does not affect resources",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []v1.Container{
			{
				Name: "container1",
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						"cpu":               api_resource.MustParse("55m"),
						"memory":            api_resource.MustParse("11M"),
						"ephemeral-storage": api_resource.MustParse("222M"),
					},
					Limits: v1.ResourceList{
						"cpu":               api_resource.MustParse("111m"),
						"memory":            api_resource.MustParse("22M"),
						"ephemeral-storage": api_resource.MustParse("333M"),
					},
				},
			},
		},
		rules:          ruleMatchNothing,
		expectRequests: nil,
		expectLimits:   nil,
	},
	{
		name:      "match-all rule preserves pod resources",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []v1.Container{
			{
				Name: "container1",
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						"cpu":               api_resource.MustParse("55m"),
						"memory":            api_resource.MustParse("11M"),
						"ephemeral-storage": api_resource.MustParse("222M"),
					},
					Limits: v1.ResourceList{
						"cpu":               api_resource.MustParse("111m"),
						"memory":            api_resource.MustParse("22M"),
						"ephemeral-storage": api_resource.MustParse("333M"),
					},
				},
			},
		},
		rules:          ruleMatchAllButDontChange,
		expectRequests: nil,
		expectLimits:   nil,
	},
	{
		name:      "set all resources on resourceless pod",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []v1.Container{
			{
				Name: "container1",
			},
		},
		rules:          ruleSetAllResources,
		expectRequests: map[string]string{"cpu": "55m", "memory": "11M", "ephemeral-storage": "222M"},
		expectLimits:   map[string]string{"cpu": "111m", "memory": "22M", "ephemeral-storage": "333M"},
	},
	{
		name:      "set all cannot change existing resources",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []v1.Container{
			{
				Name: "container1",
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						"cpu":               api_resource.MustParse("855m"),
						"memory":            api_resource.MustParse("811M"),
						"ephemeral-storage": api_resource.MustParse("8222M"),
					},
					Limits: v1.ResourceList{
						"cpu":               api_resource.MustParse("8111m"),
						"memory":            api_resource.MustParse("822M"),
						"ephemeral-storage": api_resource.MustParse("8333M"),
					},
				},
			},
		},
		rules:          ruleSetAllResources,
		expectRequests: nil,
		expectLimits:   nil,
	},
	{
		name:      "inherit limit from request",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []v1.Container{
			{
				Name: "container1",
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						"cpu":               api_resource.MustParse("40m"),
						"memory":            api_resource.MustParse("20M"),
						"ephemeral-storage": api_resource.MustParse("200M"),
					},
				},
			},
		},
		rules:          ruleSetAllResources,
		expectRequests: map[string]string{"cpu": "40m", "memory": "20M", "ephemeral-storage": "200M"},
		expectLimits:   map[string]string{"cpu": "40m", "memory": "20M", "ephemeral-storage": "200M"},
	},
	{
		name:      "inherit request from limit",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []v1.Container{
			{
				Name: "container1",
				Resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{
						"cpu":               api_resource.MustParse("40m"),
						"memory":            api_resource.MustParse("20M"),
						"ephemeral-storage": api_resource.MustParse("200M"),
					},
				},
			},
		},
		rules:          ruleSetAllResources,
		expectRequests: map[string]string{"cpu": "40m", "memory": "20M", "ephemeral-storage": "200M"},
		expectLimits:   map[string]string{"cpu": "40m", "memory": "20M", "ephemeral-storage": "200M"},
	},
	{
		name:      "2nd rule set all resources on resourceless pod",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []v1.Container{
			{
				Name: "container1",
			},
		},
		rules:          ruleSetAllResourcesOnSecondRule,
		expectRequests: map[string]string{"cpu": "55m", "memory": "11M", "ephemeral-storage": "222M"},
		expectLimits:   map[string]string{"cpu": "111m", "memory": "22M", "ephemeral-storage": "333M"},
	},
	{
		name:      "1st rule set all resources on resourceless pod",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []v1.Container{
			{
				Name: "container1",
			},
		},
		rules:          ruleSetAllResourcesOnFirstRule,
		expectRequests: map[string]string{"cpu": "55m", "memory": "11M", "ephemeral-storage": "222M"},
		expectLimits:   map[string]string{"cpu": "111m", "memory": "22M", "ephemeral-storage": "333M"},
	},
}

const ruleMatchNothing = `
resources:
  - pod:
      namespace: _ # match NOTHING
      name: "" # match anything
      #labels:
      #  a: b
    container: "" # match anything
`

const ruleMatchAllButDontChange = `
resources:
  - pod:
      namespace: "" # match anything
      name: "" # match anything
      #labels:
      #  a: b
    container: "" # match anything
`

const ruleSetAllResources = `
resources:
  - pod:
      namespace: "" # match anything
      name: "" # match anything
      #labels:
      #  a: b
    container: "" # match anything
    memory:
      requests: 11M
      limits:   22M
    cpu:
      requests: 55m
      limits:   111m
    ephemeral-storage:
      requests: 222M
      limits:   333M
`

const ruleSetAllResourcesOnSecondRule = `
resources:
  - pod:
      namespace: _ # match NOTHING
    memory:
      requests: 119M
      limits:   229M
    cpu:
      requests: 559m
      limits:   1119m
    ephemeral-storage:
      requests: 2229M
      limits:   3339M
  - pod:
      namespace: "" # match anything
      name: "" # match anything
      #labels:
      #  a: b
    container: "" # match anything
    memory:
      requests: 11M
      limits:   22M
    cpu:
      requests: 55m
      limits:   111m
    ephemeral-storage:
      requests: 222M
      limits:   333M
`

const ruleSetAllResourcesOnFirstRule = `
resources:
  - pod:
      namespace: "" # match anything
      name: "" # match anything
      #labels:
      #  a: b
    container: "" # match anything
    memory:
      requests: 11M
      limits:   22M
    cpu:
      requests: 55m
      limits:   111m
    ephemeral-storage:
      requests: 222M
      limits:   333M
  - pod:
      namespace: _ # match NOTHING
    memory:
      requests: 119M
      limits:   229M
    cpu:
      requests: 559m
      limits:   1119m
    ephemeral-storage:
      requests: 2229M
      limits:   3339M
`

func TestAddResource(t *testing.T) {

	for i, data := range resourceTestTable {

		name := fmt.Sprintf("%d of %d %s", i+1, len(resourceTestTable), data.name)

		t.Run(name, func(t *testing.T) {

			r, errRule := newRules([]byte(data.rules))
			if errRule != nil {
				t.Errorf("bad rule: %v", errRule)
				return
			}

			const debug = false

			list := addResource(data.namespace, data.podName, data.podLabels, data.containers, r.Resources, debug)

			expectedSize := 0

			if len(data.expectRequests) > 0 {
				expectedSize++
			}

			if len(data.expectLimits) > 0 {
				expectedSize++
			}

			if len(list) != expectedSize {
				t.Errorf("resource list size=%d expected=%d list:%v",
					len(list), expectedSize, list)
				return
			}

			if expectedSize == 0 {
				return
			}

			//
			// parse patch json string list into op list
			//

			var ops []op

			for i, patch := range list {
				var operation op
				errJSON := json.Unmarshal([]byte(patch), &operation)
				if errJSON != nil {
					t.Errorf("patch %d/%d json error: %v", i+1, len(list), errJSON)
					return
				}
				ops = append(ops, operation)
			}

			for i, operation := range ops {
				if operation.Op != "replace" {
					t.Errorf("unexpected operation: %d/%d: %s",
						i+1, len(ops), operation.Op)
					return
				}

				if strings.HasSuffix(operation.Path, "/requests") {
					//
					// found request
					//
					if errCompare := compareResource(data.expectRequests, operation.Value); errCompare != nil {
						t.Errorf("requests compare error: %d/%d: %v",
							i+1, len(ops), errCompare)
						return
					}
					continue
				}

				if strings.HasSuffix(operation.Path, "/limits") {
					//
					// found limit
					//
					if errCompare := compareResource(data.expectLimits, operation.Value); errCompare != nil {
						t.Errorf("limits compare error: %d/%d: %v",
							i+1, len(ops), errCompare)
						return
					}
					continue
				}

				t.Errorf("unexpected operation path=%s: %d/%d: %#v",
					operation.Path, i+1, len(ops), operation)
			}

		})

	}

}

func compareResource(expected map[string]string, got rsrc) error {
	expReq, errJSON := json.Marshal(expected)
	if errJSON != nil {
		return errJSON
	}

	var expReqRsrc rsrc
	if errUnmarshal := json.Unmarshal(expReq, &expReqRsrc); errUnmarshal != nil {
		return errUnmarshal
	}

	if expReqRsrc != got {
		return fmt.Errorf("mismatch: expected=%#v got=%#v",
			expReqRsrc, got)
	}

	return nil
}

type op struct {
	Op    string `json:"op"`
	Path  string `path:"path"`
	Value rsrc   `path:"value"`
}

type rsrc struct {
	CPU              string `json:"cpu"`
	Memory           string `json:"memory"`
	EphemeralStorage string `json:"ephemeral-storage"`
}
