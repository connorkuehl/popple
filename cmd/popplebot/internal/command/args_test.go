package command

import (
	"errors"
	"testing"
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
