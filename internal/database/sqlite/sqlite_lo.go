package sqlite

import "github.com/connorkuehl/popple/internal/env"

func pathFromEnv(f func(key string) (val string)) (Path, error) {
	path, err := env.Get("POPPLE_SQLITE_DB_PATH", f)
	if err != nil {
		return "", err
	}
	return Path(path), nil
}
