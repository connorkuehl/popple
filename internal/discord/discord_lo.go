package discord

import (
	"github.com/connorkuehl/popple/internal/env"
)

func tokenFromEnv(f func(key string) (val string)) (Token, error) {
	token, err := env.Get("POPPLE_DISCORD_TOKEN", f)
	if err != nil {
		return "", err
	}

	return Token(token), err
}
