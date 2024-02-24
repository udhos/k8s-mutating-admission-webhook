package main

import (
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

type rulesConfig struct {
	RestrictTolerations []restrictTolerationConfig `yaml:"restrict_tolerations"`
}

type restrictTolerationConfig struct {
	Toleration  tolerationConfig `yaml:"toleration"`
	AllowedPods []podConfig      `yaml:"allowed_pods"`
}

type tolerationConfig struct {
	Key      string `yaml:"key"`
	Operator string `yaml:"operator"`
	Value    string `yaml:"value"`
	Effect   string `yaml:"effect"`

	key      *regexp.Regexp
	operator *regexp.Regexp
	value    *regexp.Regexp
	effect   *regexp.Regexp
}

type podConfig struct {
	Namespace string `yaml:"namespace"`
	Name      string `yaml:"name"`

	namespace *regexp.Regexp
	name      *regexp.Regexp
}

func (t *tolerationConfig) match(podToleration corev1.Toleration) bool {
	return t.key.MatchString(podToleration.Key) &&
		t.operator.MatchString(string(podToleration.Operator)) &&
		t.value.MatchString(podToleration.Value) &&
		t.effect.MatchString(string(podToleration.Effect))
}

func (p *podConfig) match(namespace, podName string) bool {
	return p.namespace.MatchString(namespace) && p.name.MatchString(podName)
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
			key, errKey := regexp.Compile(r.RestrictTolerations[i].Toleration.Key)
			if errKey != nil {
				return r, errKey
			}
			r.RestrictTolerations[i].Toleration.key = key
		}

		{
			op, errOp := regexp.Compile(r.RestrictTolerations[i].Toleration.Operator)
			if errOp != nil {
				return r, errOp
			}
			r.RestrictTolerations[i].Toleration.operator = op
		}

		{
			v, errV := regexp.Compile(r.RestrictTolerations[i].Toleration.Value)
			if errV != nil {
				return r, errV
			}
			r.RestrictTolerations[i].Toleration.value = v
		}

		{
			eff, errEff := regexp.Compile(r.RestrictTolerations[i].Toleration.Effect)
			if errEff != nil {
				return r, errEff
			}
			r.RestrictTolerations[i].Toleration.effect = eff
		}

		for j := range r.RestrictTolerations[i].AllowedPods {

			{
				ns, errNs := regexp.Compile(r.RestrictTolerations[i].AllowedPods[j].Namespace)
				if errNs != nil {
					return r, errNs
				}
				r.RestrictTolerations[i].AllowedPods[j].namespace = ns
			}

			{
				name, errName := regexp.Compile(r.RestrictTolerations[i].AllowedPods[j].Name)
				if errName != nil {
					return r, errName
				}
				r.RestrictTolerations[i].AllowedPods[j].name = name
			}

		}
	}

	return r, nil
}
