package main

import (
	"os"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

type rulesConfig struct {
	RestrictTolerations []restrictTolerationConfig `yaml:"restrict_tolerations"`
	PlacePods           []placementConfig          `yaml:"place_pods"`
	Resources           []setResource              `yaml:"resources"`
}

type setResource struct {
	Pod              podConfig `yaml:"pod"`
	Container        string    `yaml:"container"`
	Memory           resource  `yaml:"memory"`
	CPU              resource  `yaml:"cpu"`
	EphemeralStorage resource  `yaml:"ephemeral-storage"`

	container *pattern
}

type resource struct {
	Request string `yaml:"requests"`
	Limit   string `yaml:"limits"`
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

			p, errCompile := compilePod(r.RestrictTolerations[i].AllowedPods[j])
			if errCompile != nil {
				return r, errCompile
			}

			r.RestrictTolerations[i].AllowedPods[j] = p
		}
	}

	for i := range r.PlacePods {

		for j := range r.PlacePods[i].Pods {

			p, errCompile := compilePod(r.PlacePods[i].Pods[j])
			if errCompile != nil {
				return r, errCompile
			}

			r.PlacePods[i].Pods[j] = p
		}

	}

	for i := range r.Resources {

		p, errCompile := compilePod(r.Resources[i].Pod)
		if errCompile != nil {
			return r, errCompile
		}
		r.Resources[i].Pod = p

		c, errC := patternCompile(r.Resources[i].Container)
		if errC != nil {
			return r, errC
		}
		r.Resources[i].container = c
	}

	return r, nil
}

func compilePod(p podConfig) (podConfig, error) {

	{
		ns, errNs := patternCompile(p.Namespace)
		if errNs != nil {
			return p, errNs
		}
		p.namespace = ns
	}

	{
		name, errName := patternCompile(p.Name)
		if errName != nil {
			return p, errName
		}
		p.name = name
	}

	and, errAnd := compileAnd(p.And)
	if errAnd != nil {
		return p, errAnd
	}

	p.And = and

	return p, nil
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
