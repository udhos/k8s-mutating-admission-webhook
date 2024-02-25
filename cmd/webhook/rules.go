package main

import (
	"os"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

type rulesConfig struct {
	RestrictTolerations []restrictTolerationConfig `yaml:"restrict_tolerations"`
	PlacePods           []placementConfig          `yaml:"place_pods"`
}

type restrictTolerationConfig struct {
	Toleration  tolerationConfigPattern `yaml:"toleration"`
	AllowedPods []podConfig             `yaml:"allowed_pods"`
}

type tolerationConfigPattern struct {
	tolerationConfig `yaml:",inline"`

	key      *pattern
	operator *pattern
	value    *pattern
	effect   *pattern
}

type podConfig struct {
	Namespace string            `yaml:"namespace"`
	Name      string            `yaml:"name"`
	Labels    map[string]string `yaml:"labels"`

	namespace *pattern
	name      *pattern
}

type placementConfig struct {
	Pod podConfig `yaml:"pod"`
	Add addConfig `yaml:"add"`
}

type addConfig struct {
	Tolerations  []tolerationConfig `yaml:"tolerations"`
	NodeSelector map[string]string  `yaml:"node_selector"`
}

type tolerationConfig struct {
	Key      string `yaml:"key"`
	Operator string `yaml:"operator"`
	Value    string `yaml:"value"`
	Effect   string `yaml:"effect"`
}

func (t *tolerationConfigPattern) match(podToleration corev1.Toleration) bool {
	return t.key.matchString(podToleration.Key) &&
		t.operator.matchString(string(podToleration.Operator)) &&
		t.value.matchString(podToleration.Value) &&
		t.effect.matchString(string(podToleration.Effect))
}

func (p *podConfig) match(namespace, podName string, podLabels map[string]string) bool {
	return p.namespace.matchString(namespace) && p.name.matchString(podName) && hasLabels(podLabels, p.Labels)
}

func hasLabels(existingLabels, requiredLabels map[string]string) bool {
	for rk, rv := range requiredLabels {
		if ev, found := existingLabels[rk]; !found || ev != rv {
			return false
		}
	}
	return true
}

func loadRules(path string) (rulesConfig, error) {
	data, errRead := os.ReadFile(path)
	if errRead != nil {
		return rulesConfig{}, errRead
	}
	return newRules(data)
}

func newRules(data []byte) (rulesConfig, error) {
	var r rulesConfig

	if errYaml := yaml.Unmarshal(data, &r); errYaml != nil {
		return r, errYaml
	}

	for i := range r.RestrictTolerations {

		{
			key, errKey := patternCompile(r.RestrictTolerations[i].Toleration.Key)
			if errKey != nil {
				return r, errKey
			}
			r.RestrictTolerations[i].Toleration.key = key
		}

		{
			op, errOp := patternCompile(r.RestrictTolerations[i].Toleration.Operator)
			if errOp != nil {
				return r, errOp
			}
			r.RestrictTolerations[i].Toleration.operator = op
		}

		{
			v, errV := patternCompile(r.RestrictTolerations[i].Toleration.Value)
			if errV != nil {
				return r, errV
			}
			r.RestrictTolerations[i].Toleration.value = v
		}

		{
			eff, errEff := patternCompile(r.RestrictTolerations[i].Toleration.Effect)
			if errEff != nil {
				return r, errEff
			}
			r.RestrictTolerations[i].Toleration.effect = eff
		}

		for j := range r.RestrictTolerations[i].AllowedPods {

			{
				ns, errNs := patternCompile(r.RestrictTolerations[i].AllowedPods[j].Namespace)
				if errNs != nil {
					return r, errNs
				}
				r.RestrictTolerations[i].AllowedPods[j].namespace = ns
			}

			{
				name, errName := patternCompile(r.RestrictTolerations[i].AllowedPods[j].Name)
				if errName != nil {
					return r, errName
				}
				r.RestrictTolerations[i].AllowedPods[j].name = name
			}

		}
	}

	for i := range r.PlacePods {

		{
			ns, errNs := patternCompile(r.PlacePods[i].Pod.Namespace)
			if errNs != nil {
				return r, errNs
			}
			r.PlacePods[i].Pod.namespace = ns
		}

		{
			name, errName := patternCompile(r.PlacePods[i].Pod.Name)
			if errName != nil {
				return r, errName
			}
			r.PlacePods[i].Pod.name = name
		}

	}

	return r, nil
}
