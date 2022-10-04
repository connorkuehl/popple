package command

import (
	"errors"
	"reflect"
	"testing"

	"github.com/connorkuehl/popple"
)

func TestParseSetAnnounceArgs(t *testing.T) {
	type result struct {
		arg SetAnnounceArgs
		err error
	}

	tests := []struct {
		input string
		want  result
	}{
		{
			input: "on",
			want:  result{arg: SetAnnounceArgs{NoAnnounce: false}},
		},
		{
			input: "yes",
			want:  result{arg: SetAnnounceArgs{NoAnnounce: false}},
		},
		{
			input: "off",
			want:  result{arg: SetAnnounceArgs{NoAnnounce: true}},
		},
		{
			input: "no",
			want:  result{arg: SetAnnounceArgs{NoAnnounce: true}},
		},
		{
			input: "",
			want:  result{err: ErrMissingArgument},
		},
		{
			input: "bogus",
			want:  result{err: ErrInvalidArgument},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var got SetAnnounceArgs
			err := got.ParseArg(tt.input)

			if !errors.Is(err, tt.want.err) {
				t.Errorf("want err=%v, got err=%v", tt.want.err, err)
			}

			if got != tt.want.arg {
				t.Errorf("want arg=%v, got arg=%v", tt.want.arg, got)
			}
		})
	}
}

func TestParseChangeKarmaArgs(t *testing.T) {
	tests := []struct {
		input string
		want  popple.Increments
	}{
		{
			input: "a++ b-- (c and d)++",
			want: popple.Increments{
				"a":       1,
				"b":       -1,
				"c and d": 1,
			},
		},
		{
			// The g should be dropped from the increments map.
			input: "e-- f++ g",
			want: popple.Increments{
				"e": -1,
				"f": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var got ChangeKarmaArgs

			err := got.ParseArg(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tt.want, got.Increments) {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}
