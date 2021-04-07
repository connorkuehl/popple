package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseKarma(t *testing.T) {
	var tests = []struct {
		input string
		want  int
	}{
		{"++", 1},
		{"--", -1},
		{"+++", 1},
		{"++--", 0},
		{"++++++", 3},
		{"++++a++", 2},
		{"+a", 0},
		{"", 0},
		{"--a++", -1},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s,%d", tt.input, tt.want)
		t.Run(testname, func(t *testing.T) {
			actual := parseKarma(strings.NewReader(tt.input))
			if actual != tt.want {
				t.Errorf("got %d, want %d", actual, tt.want)
			}
		})
	}
}

func TestParseSubjectTilWhitespaceOrKarma(t *testing.T) {
	var tests = []struct {
		input string
		want  string
	}{
		{"PoeThePotatoPirate", "PoeThePotatoPirate"},
		{"Poe++", "Poe"},
		{"Poe++Hello", "Poe"},
		{"Po-e", "Po"},
		{"Hello World", "Hello"},
		{"@Poe", "Poe"},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s,%s", tt.input, tt.want)
		t.Run(testname, func(t *testing.T) {
			actual, _ := parseSubjectTilWhitespaceOrKarma(strings.NewReader(tt.input))
			if actual != tt.want {
				t.Errorf("got %s, want %s", actual, tt.want)
			}
		})
	}
}

func TestParseSubjectInParens(t *testing.T) {
	var tests = []struct {
		input string
		want  string
		wOk   bool
	}{
		{"(Poe the Potato Pirate)", "Poe the Potato Pirate", true},
		{"(Poe the Potato Pirate", "", false},
		{"()", "", false},
		{"(Poe the Potato Pirate)++", "Poe the Potato Pirate", true},
		{"(Poe the Potato Pirate++)", "Poe the Potato Pirate++", true},
		{"", "", false},
		{"Poe the Potato Pirate", "", false},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s,%s/%v", tt.input, tt.want, tt.wOk)
		t.Run(testname, func(t *testing.T) {
			actual, ok := parseSubjectInParens(strings.NewReader(tt.input))
			if actual != tt.want || ok != tt.wOk {
				t.Errorf("got %s/%v, want %s/%v", actual, ok, tt.want, tt.wOk)
			}
		})
	}
}

func TestParseModifier(t *testing.T) {
	var tests = []struct {
		input       string
		wantSubject string
		wantKarma   int
		wantOk      bool
	}{
		{"(Poe)++", "Poe", 1, true},
		{"Poe++", "Poe", 1, true},
		{"(Poe the Potato Pirate)++--a++", "Poe the Potato Pirate", 0, true},
		{"(Hello World++234", "", 0, false},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s,%s/%d,%v", tt.input, tt.wantSubject, tt.wantKarma, tt.wantOk)
		t.Run(testname, func(t *testing.T) {
			s, k, ok := parseModifier(strings.NewReader(tt.input))
			if s != tt.wantSubject || k != tt.wantKarma || ok != tt.wantOk {
				t.Errorf("got %s/%d,%v, want %s/%d,%v", s, k, ok, tt.wantSubject, tt.wantKarma, tt.wantOk)
			}
		})
	}
}

func TestParseModifiers(t *testing.T) {
	var tests = []struct {
		input string
		want  map[string]int
	}{
		{"Poe++", map[string]int{
			"Poe": 1,
		}},
		{"None of this, except++", map[string]int{
			"None":   0,
			"except": 1,
			"of":     0,
			"this,":  0,
		}},
		{"       hi++", map[string]int{
			"hi": 1,
		}},
		{"     hello    world--         bye----", map[string]int{
			"hello": 0,
			"world": -1,
			"bye":   -2,
		}},
		{"         (Poe the Potato Pirate)++++    foo--++          bar++", map[string]int{
			"Poe the Potato Pirate": 2,
			"foo":                   0,
			"bar":                   1,
		}},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s,%v", tt.input, tt.want)
		t.Run(testname, func(t *testing.T) {
			actual := ParseModifiers(tt.input)
			if !reflect.DeepEqual(actual, tt.want) {
				t.Errorf("got %v, want %v", actual, tt.want)
			}
		})
	}
}
