package popple

import (
	"errors"
	"reflect"
	"testing"

	poperr "github.com/connorkuehl/popple/errors"
)

func TestParseAnnounceArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "on",
			input: " on",
			want:  true,
		},
		{
			name:  "yes",
			input: " yes",
			want:  true,
		},
		{
			name:  "off",
			input: " off",
			want:  false,
		},
		{
			name:  "no",
			input: " no",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAnnounceArgs(tt.input)
			if err != nil {
				t.Errorf("unexpected err: %v", err)
			}

			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("argument is required", func(t *testing.T) {
		_, err := ParseAnnounceArgs("")
		if !errors.Is(err, poperr.ErrMissingArgument) {
			t.Errorf("got %v, want %v", err, poperr.ErrMissingArgument)
		}
	})

	t.Run("argument must be valid", func(t *testing.T) {
		_, err := ParseAnnounceArgs("asdf")
		if !errors.Is(err, poperr.ErrInvalidArgument) {
			t.Errorf("got %v, want %v", err, poperr.ErrMissingArgument)
		}
	})
}

func TestParseBumpKarmaArgs(t *testing.T) {
}

func TestParseKarmaArgs(t *testing.T) {
	t.Run("it needs at least one name", func(t *testing.T) {
		_, err := ParseKarmaArgs("")
		if !errors.Is(err, poperr.ErrMissingArgument) {
			t.Errorf("got %v, want %v", err, poperr.ErrMissingArgument)
		}
	})

	t.Run("it copies over entities that it has found", func(t *testing.T) {
		got, _ := ParseKarmaArgs("mare (musk hogg)")
		want := map[string]struct{}{
			"mare":      {},
			"musk hogg": {},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestParseBoardArgs(t *testing.T) {
	t.Run("it sets the limit to a default value when one is not supplied", func(t *testing.T) {
		prev := defaultLeaderboardSize
		defer func() { defaultLeaderboardSize = prev }()
		defaultLeaderboardSize = 2

		got, _ := ParseBoardArgs("")
		want := uint(2)
		if got != want {
			t.Errorf("got %d, want %d", got, want)
		}
	})

	t.Run("it returns an error if the argument is 0", func(t *testing.T) {
		_, got := ParseBoardArgs("0")
		if !errors.Is(got, poperr.ErrInvalidArgument) {
			t.Errorf("got %v, want %v", got, poperr.ErrInvalidArgument)
		}
	})

	t.Run("it returns an error when the argument is not an integer", func(t *testing.T) {
		_, got := ParseBoardArgs("asdf")
		if !errors.Is(got, poperr.ErrInvalidArgument) {
			t.Errorf("got %v, want %v", got, poperr.ErrInvalidArgument)
		}
	})

	t.Run("it accepts a positive non-zero integer as a limit", func(t *testing.T) {
		got, _ := ParseBoardArgs("1")
		want := uint(1)
		if got != want {
			t.Errorf("got %d, want %d", got, want)
		}
	})
}
