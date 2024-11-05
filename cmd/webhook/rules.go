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

	And []podConfig `yaml:"and"`

	namespace *pattern
	name      *pattern
}

type placementConfig struct {
	Pods []podConfig `yaml:"pods"`
	Add  addConfig   `yaml:"add"`
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

func (pc *placementConfig) match(namespace, podName string, podLabels map[string]string) bool {
	for _, podC := range pc.Pods {
		if podC.match(namespace, podName, podLabels) {
			return true
		}
	}
	return false
}

func (p *podConfig) match(namespace, podName string, podLabels map[string]string) bool {

	if len(p.And) > 0 {
		for _, sub := range p.And {
			if !sub.match(namespace, podName, podLabels) {
				return false
			}
		}
	}

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

			and, errAnd := compileAnd(r.RestrictTolerations[i].AllowedPods[j].And)
			if errAnd != nil {
				return r, errAnd
			}

			r.RestrictTolerations[i].AllowedPods[j].And = and
		}
	}

	for i := range r.PlacePods {

		for j := range r.PlacePods[i].Pods {
			ns, errNs := patternCompile(r.PlacePods[i].Pods[j].Namespace)
			if errNs != nil {
				return r, errNs
			}
			r.PlacePods[i].Pods[j].namespace = ns

			name, errName := patternCompile(r.PlacePods[i].Pods[j].Name)
			if errName != nil {
				return r, errName
			}
			r.PlacePods[i].Pods[j].name = name
		}

	}

	return r, nil
}

func compileAnd(list []podConfig) ([]podConfig, error) {
	for i, p := range list {

		{
			ns, errNs := patternCompile(p.Namespace)
			if errNs != nil {
				return list, errNs
			}
			list[i].namespace = ns
		}

		{
			name, errName := patternCompile(p.Name)
			if errName != nil {
				return list, errName
			}
			list[i].name = name
		}

		and, errAnd := compileAnd(p.And)
		if errAnd != nil {
			return list, errAnd
		}

		list[i].And = and
	}

	return list, nil
}
