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
	return p.negate != p.re.MatchString(s)
}

const patternNegatePrefix = "_"

func patternCompile(s string) (*pattern, error) {
	p := &pattern{}
	if after, ok := strings.CutPrefix(s, patternNegatePrefix); ok {
		s = after
		p.negate = true
	}
	re, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}
	p.re = re
	return p, nil
}
