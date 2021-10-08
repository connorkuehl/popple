package popple

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseSubjects(t *testing.T) {
	var tests = []struct {
		name  string
		input string
		want  []subject
	}{
		{"incrementing a plain subject adds 1 karma", "Test++", []subject{{"Test", 1}}},
		{"decrementing a plain subject subtracts 1 karma", "Test--", []subject{{"Test", -1}}},
		{"complicated subjects can be enclosed in parentheses", "(Complicated Karma)--", []subject{{"Complicated Karma", -1}}},
		{"a subject's name may include parentheses if nested in outer parentheses", "((Nested) (sub)(ject))++", []subject{{"(Nested) (sub)(ject)", 1}}},
		{"empty parentheses results in nothing", "()", []subject{}},
		{"leading whitespace is discarded", "   Some Spaces++", []subject{{"Some", 0}, {"Spaces", 1}}},
		{"an unclosed parentheses does not prevent parsing other subjects", "(unmatched hello++", []subject{{"unmatched", 0}, {"hello", 1}}},
		{"a karma event must be followed by whitespace or eof", "no++karma", []subject{{"no++karma", 0}}},
		{"a karma event can be parsed from non-karma events", "yes-- karma", []subject{{"yes", -1}, {"karma", 0}}},
		{"plain and parentheses-style subjects can be mixed", "A number++ of (subjects with karma)--", []subject{{"A", 0}, {"number", 1}, {"of", 0}, {"subjects with karma", -1}}},
		{"incrementing nothing yields nothing", "++a", []subject{{"++a", 0}}},
		{"a parenthesis subject without a karma event yields nothing", "(nothing) (something)++", []subject{{"nothing", 0}, {"something", 1}}},
		{"no karma events results in no subjects", "hi goodbye farewell ", []subject{{"hi", 0}, {"goodbye", 0}, {"farewell", 0}}},
		{"a karma event is a valid subject", "++++", []subject{{"++", 1}}},
		{"empty input yields no subjects", "", []subject{}},
		{"an increment yields nothing", "++ -- ", []subject{}},
		{"bumping empty parens yields nothing", "()++ ()--", []subject{}},
		{"a paren subject ending with a single character is taken as plaintext", "(hi)+", []subject{{"(hi)+", 0}}},
		{"a paren subject can have a leading @", "(@hi)++", []subject{{"@hi", 1}}},
		{"a plaintext subject will have a leading @ stripped", "@hi++", []subject{{"hi", 1}}},
		{"@++", "@++", []subject{}},
		{"stop parsing when a backtick is found", "asdf++ ` oh hi++", []subject{{"asdf", 1}}},
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
