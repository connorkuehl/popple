package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseSubjects(t *testing.T) {
	var tests = []struct {
		name  string
		input string
		want  []Subject
	}{
		{"incrementing a plain subject adds 1 karma", "Test++", []Subject{{"Test", 1}}},
		{"decrementing a plain subject subtracts 1 karma", "Test--", []Subject{{"Test", -1}}},
		{"complicated subjects can be enclosed in parentheses", "(Complicated Karma)--", []Subject{{"Complicated Karma", -1}}},
		{"a subject's name may include parentheses if nested in outer parentheses", "((Nested) (sub)(ject))++", []Subject{{"(Nested) (sub)(ject)", 1}}},
		{"empty parentheses results in nothing", "()", []Subject{}},
		{"leading whitespace is discarded", "   Some Spaces++", []Subject{{"Spaces", 1}}},
		{"an unclosed parentheses does not prevent parsing other subjects", "(unmatched hello++", []Subject{{"hello", 1}}},
		{"a karma event must be followed by whitespace or eof", "no++karma", []Subject{}},
		{"a karma event can be parsed from non-karma events", "yes-- karma", []Subject{{"yes", -1}}},
		{"plain and parentheses-style subjects can be mixed", "A number++ of (subjects with karma)--", []Subject{{"number", 1}, {"subjects with karma", -1}}},
		{"incrementing nothing yields nothing", "++a", []Subject{}},
		{"a karma event cannot be suffixed with a backtick", "hi++`", []Subject{}},
		{"code fences are ignored during parsing", "```code fence``` test++", []Subject{{"test", 1}}},
		{"a single parenthesis yields nothing", "(", []Subject{}},
		{"a parenthesis subject without a karma event yields nothing", "(nothing) (something)++", []Subject{{"something", 1}}},
		{"no karma events results in no subjects", "hi goodbye farewell ", []Subject{}},
		{"a karma event is a valid subject", "++++", []Subject{{"++", 1}}},
		{"empty input yields no subjects", "", []Subject{}},
		{"karma events inside backticks are ignored", "```c++```", []Subject{}},
		{"an increment yields nothing", "++ -- ", []Subject{}},
		{"bumping empty parens yields nothing", "()++ ()--", []Subject{}},
		{"karma events inside of backticks should be ignored", "`all++ of-- this++ should-- be++ ignored--`", []Subject{}},
		{"parser will backtrace if tick is not closed", "`c test++", []Subject{{"test", 1}}},
		{"karma events inside of fences should be ignored", "``` test++ ```", []Subject{}},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("%s %s,%v", tt.name, tt.input, tt.want)
		t.Run(testName, func(t *testing.T) {
			actual := ParseSubjects(tt.input)
			if !reflect.DeepEqual(actual, tt.want) {
				t.Errorf("got %v, want %v", actual, tt.want)
			}
		})
	}
}
