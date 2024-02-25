package main

import (
	"regexp"
	"strings"
)

type pattern struct {
	re     *regexp.Regexp
	negate bool
}

func (p *pattern) matchString(s string) bool {
	m := p.re.MatchString(s)
	if p.negate {
		return !m
	}
	return m
}

const patternNegatePrefix = "_"

func patternCompile(s string) (*pattern, error) {
	p := &pattern{}
	if strings.HasPrefix(s, patternNegatePrefix) {
		s = strings.TrimPrefix(s, patternNegatePrefix)
		p.negate = true
	}
	re, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}
	p.re = re
	return p, nil
}
