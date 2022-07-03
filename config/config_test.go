package config

import (
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("it loads a well-defined config", func(t *testing.T) {
		config := `
		token 1234
		database /etc/popple/good-file`

		got, err := Load(strings.NewReader(config))
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		want := Config{
			Token:  "1234",
			DBPath: "/etc/popple/good-file",
		}

		if got != want {
			t.Errorf("want %+v, got %+v", want, got)
		}
	})

	t.Run("it returns ErrUnknownKey when the config contains an unknown key", func(t *testing.T) {
		config := `
		token 1337
		potato-pirate poe
		`

		_, err := Load(strings.NewReader(config))

		want := ErrUnknownKey{Line: 2, Key: "potato-pirate"}
		if got := err.(ErrUnknownKey); got != want {
			t.Errorf("want %#v, got %#v", want, got)
		}
	})

	t.Run("it returns ErrMissingValue when the config key does not have a value", func(t *testing.T) {
		config := `
		database hjkl
		token
		`

		_, err := Load(strings.NewReader(config))

		want := ErrMissingValue{Line: 2, ForKey: "token"}
		if got := err.(ErrMissingValue); got != want {
			t.Errorf("want %#v, got %#v", want, got)
		}
	})

	t.Run("it ignores lines starting with a #", func(t *testing.T) {
		config := `
		token 8529
		# this will be ignored
		# database also-ignored
		database potato
		`

		got, err := Load(strings.NewReader(config))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		want := Config{Token: "8529", DBPath: "potato"}
		if got != want {
			t.Errorf("want %#v, got %#v", want, got)
		}
	})
}
