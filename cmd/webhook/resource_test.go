package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	api_resource "k8s.io/apimachinery/pkg/api/resource"
)

type resourceTestCase struct {
	name string

	namespace string
	podName   string
	podLabels map[string]string

	rules string

	containers []containerTest
}

type containerTest struct {
	container      v1.Container
	expectRequests map[string]string
	expectLimits   map[string]string
}

var resourceTestTable = []resourceTestCase{
	{
		name:      "empty rule does not affect resources",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []containerTest{
			{
				container: v1.Container{
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
				expectRequests: nil,
				expectLimits:   nil,
			},
		},
		rules: "",
	},
	{
		name:      "match-NOTHING rule does not affect resources",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []containerTest{
			{
				container: v1.Container{
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
				expectRequests: nil,
				expectLimits:   nil,
			},
		},
		rules: ruleMatchNothing,
	},
	{
		name:      "match-all rule preserves pod resources",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []containerTest{
			{
				container: v1.Container{
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
				expectRequests: nil,
				expectLimits:   nil,
			},
		},
		rules: ruleMatchAllButDontChange,
	},
	{
		name:      "set all resources on resourceless pod",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []containerTest{
			{
				container: v1.Container{
					Name: "container1",
				},
				expectRequests: map[string]string{"cpu": "55m", "memory": "11M", "ephemeral-storage": "222M"},
				expectLimits:   map[string]string{"cpu": "111m", "memory": "22M", "ephemeral-storage": "333M"},
			},
		},
		rules: ruleSetAllResources,
	},
	{
		name:      "set all cannot change existing resources",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []containerTest{
			{
				container: v1.Container{
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
				expectRequests: nil,
				expectLimits:   nil,
			},
		},
		rules: ruleSetAllResources,
	},
	{
		name:      "inherit limit from request",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []containerTest{
			{
				container: v1.Container{
					Name: "container1",
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							"cpu":               api_resource.MustParse("40m"),
							"memory":            api_resource.MustParse("20M"),
							"ephemeral-storage": api_resource.MustParse("200M"),
						},
					},
				},
				expectRequests: map[string]string{"cpu": "40m", "memory": "20M", "ephemeral-storage": "200M"},
				expectLimits:   map[string]string{"cpu": "40m", "memory": "20M", "ephemeral-storage": "200M"},
			},
		},
		rules: ruleSetAllResources,
	},
	{
		name:      "inherit request from limit",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []containerTest{
			{
				container: v1.Container{
					Name: "container1",
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							"cpu":               api_resource.MustParse("40m"),
							"memory":            api_resource.MustParse("20M"),
							"ephemeral-storage": api_resource.MustParse("200M"),
						},
					},
				},
				expectRequests: map[string]string{"cpu": "40m", "memory": "20M", "ephemeral-storage": "200M"},
				expectLimits:   map[string]string{"cpu": "40m", "memory": "20M", "ephemeral-storage": "200M"},
			},
		},
		rules: ruleSetAllResources,
	},
	{
		name:      "2nd rule set all resources on resourceless pod",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []containerTest{
			{
				container: v1.Container{
					Name: "container1",
				},
				expectRequests: map[string]string{"cpu": "55m", "memory": "11M", "ephemeral-storage": "222M"},
				expectLimits:   map[string]string{"cpu": "111m", "memory": "22M", "ephemeral-storage": "333M"},
			},
		},
		rules: ruleSetAllResourcesOnSecondRule,
	},
	{
		name:      "1st rule set all resources on resourceless pod",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []containerTest{
			{
				container: v1.Container{
					Name: "container1",
				},
				expectRequests: map[string]string{"cpu": "55m", "memory": "11M", "ephemeral-storage": "222M"},
				expectLimits:   map[string]string{"cpu": "111m", "memory": "22M", "ephemeral-storage": "333M"},
			},
		},
		rules: ruleSetAllResourcesOnFirstRule,
	},
	{
		name:      "set all resources on all pod resourceless containers",
		namespace: "default",
		podName:   "pod-",
		podLabels: nil,
		containers: []containerTest{
			{
				container: v1.Container{
					Name: "container1",
				},
				expectRequests: map[string]string{"cpu": "55m", "memory": "11M", "ephemeral-storage": "222M"},
				expectLimits:   map[string]string{"cpu": "111m", "memory": "22M", "ephemeral-storage": "333M"},
			},
			{
				container: v1.Container{
					Name: "container2",
				},
				expectRequests: map[string]string{"cpu": "55m", "memory": "11M", "ephemeral-storage": "222M"},
				expectLimits:   map[string]string{"cpu": "111m", "memory": "22M", "ephemeral-storage": "333M"},
			},
		},
		rules: ruleSetAllResources,
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

			// build container list for addResource()
			var containerList []v1.Container
			for _, c := range data.containers {
				containerList = append(containerList, c.container)
			}

			const debug = false

			list := addResource(data.namespace, data.podName, data.podLabels, containerList, r.Resources, debug)

			expectedSize := 0
			for _, c := range data.containers {
				if len(c.expectRequests) > 0 {
					expectedSize++
				}
				if len(c.expectLimits) > 0 {
					expectedSize++
				}
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

			// scan test containers

			//
			// ops: list of patch ops
			// data.containers: list of test containers
			//

			var countFoundOp int

			for dataContainerID, c := range data.containers {

				containerName := fmt.Sprintf("%s(%d)", c.container.Name, dataContainerID)

				// search patch op for container

				for i, operation := range ops {
					if operation.Op != "replace" {
						t.Errorf("container=%s unexpected operation: %d/%d: %s",
							containerName, i+1, len(ops), operation.Op)
						return
					}

					// "/spec/containers/0/resources/requests"

					fields := strings.SplitN(operation.Path, "/", 6)

					reqLim := fields[5]
					containerID := fields[3]
					cID, errConv := strconv.Atoi(containerID)
					if errConv != nil {
						t.Errorf("container=%s parse container ID from path=%s error: %d/%d: %v",
							containerName, operation.Path, i+1, len(ops), errConv)
						return
					}

					if cID != dataContainerID {
						continue // keep searching
					}

					if reqLim == "requests" {
						//
						// found request
						//
						if errCompare := compareResource(c.expectRequests, operation.Value); errCompare != nil {
							t.Errorf("container=%s requests compare error: %d/%d: %v",
								containerName, i+1, len(ops), errCompare)
							return
						}
						countFoundOp++
						continue // next op
					}

					if reqLim == "limits" {
						//
						// found limit
						//
						if errCompare := compareResource(c.expectLimits, operation.Value); errCompare != nil {
							t.Errorf("container=%s limits compare error: %d/%d: %v",
								containerName, i+1, len(ops), errCompare)
							return
						}
						countFoundOp++
						continue // next op
					}

					t.Errorf("container=%s unexpected operation path=%s: %d/%d: %#v",
						containerName, operation.Path, i+1, len(ops), operation)
					return

				} // search patch op for container

			} // scan test containers

			if countFoundOp != expectedSize {
				t.Errorf("operation count mismatch: expected=%d got=%d",
					expectedSize, countFoundOp)
				return
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
