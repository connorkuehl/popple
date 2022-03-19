package parse

import (
	"reflect"
	"testing"
)

func TestSubjects(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]int64
	}{
		{
			name:  "simple increment",
			input: "iron++",
			want:  map[string]int64{"iron": 1},
		},
		{
			name:  "simple decrement",
			input: "felt--",
			want:  map[string]int64{"felt": -1},
		},
		{
			name:  "simple parens increment",
			input: "(a word)++",
			want:  map[string]int64{"a word": 1},
		},
		{
			name:  "simple parens decrement",
			input: "(a bird)--",
			want:  map[string]int64{"a bird": -1},
		},
		{
			name:  "multiple non-parens",
			input: "a++ b-- c++",
			want:  map[string]int64{"a": 1, "b": -1, "c": 1},
		},
		{
			name:  "multiple parens",
			input: "(a bird)-- (a plane)++ (superman)++",
			want:  map[string]int64{"a bird": -1, "a plane": 1, "superman": 1},
		},
		{
			name:  "mixed form",
			input: "(one hundred suns)++ lamp--",
			want:  map[string]int64{"one hundred suns": 1, "lamp": -1},
		},
		{
			name:  "abandon when tilde encountered",
			input: "works++ `(doesn't work)++",
			want:  map[string]int64{"works": 1},
		},
		{
			name:  "empty string not allowed non-parens",
			input: "++",
			want:  map[string]int64{},
		},
		{
			name:  "empty string not allowed parens",
			input: "()--",
			want:  map[string]int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Subjects(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
