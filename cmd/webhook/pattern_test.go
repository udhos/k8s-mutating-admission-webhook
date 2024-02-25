package main

import (
	"fmt"
	"testing"
)

type patternTestCase struct {
	name           string
	expr           string
	matchStr       string
	expectedResult bool
}

// DO NOT add negated expressions in the table below,
// because TestNegatePattern already negate all expressions,
// but double negation is not supported.

var patternTestTable = []patternTestCase{
	{"match anything (empty) 1", "", "", true},
	{"match anything (empty) 2", "", "x", true},
	{"match anything (empty) 3", "", " ", true},
	{"match anything (empty) 4", "", " x", true},
	{"match literal 1", "abc", "abc", true},
	{"match literal 2", "abc", "abcd", true},
	{"match literal 3", "abc", "ab", false},
	{"match anything 1", ".*", "", true},
	{"match anything 2", ".*", "x", true},
	{"match anything 3", ".*", "xx", true},
	{"match anything 4", ".*", " ", true},
	{"match anything 1", ".?", "", true},
	{"match anything 2", ".?", "x", true},
	{"match anything 3", ".?", "xx", true},
	{"match anything 4", ".?", " ", true},
	{"match exists 1", "^Exists$", "Exists", true},
	{"match exists 2", "^Exists$", "Exist", false},
	{"match exists 3", "^Exists$", "Exists2", false},
	{"match only empty 1", "^$", "", true},
	{"match only empty 2", "^$", " ", false},
	{"match only empty 3", "^$", "a", false},
}

func TestPattern(t *testing.T) {
	for i, data := range patternTestTable {
		testName := fmt.Sprintf("%d: %s:", i, data.name)
		t.Logf("%s expr=%s str=%s expected=%t",
			testName, data.expr, data.matchStr, data.expectedResult)
		p, errCompile := patternCompile(data.expr)
		if errCompile != nil {
			t.Errorf("%s compile error '%s': %v",
				testName, data.expr, errCompile)
		}
		m := p.matchString(data.matchStr)
		if m != data.expectedResult {
			t.Errorf("%s bad result expr='%s' str='%s': got=%t expected=%t",
				testName, data.expr, data.matchStr, m, data.expectedResult)
		}
	}
}

func TestNegatePattern(t *testing.T) {
	for i, data := range patternTestTable {
		testName := fmt.Sprintf("%d: %s:", i, data.name)
		expr := patternNegatePrefix + data.expr // negate pattern
		expected := !data.expectedResult        // negate expected
		t.Logf("%s expr=%s str=%s expected=%t",
			testName, expr, data.matchStr, expected)
		p, errCompile := patternCompile(expr)
		if errCompile != nil {
			t.Errorf("%s compile negate error '%s': %v",
				testName, expr, errCompile)
		}
		m := p.matchString(data.matchStr)
		if m != expected {
			t.Errorf("%s bad negate result expr='%s' str='%s': got=%t expected=%t",
				testName, expr, data.matchStr, m, expected)
		}
	}
}
