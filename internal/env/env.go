package env

import "errors"

var ErrKeyNotFound = errors.New("not found")

func Get(key string, f func(key string) (val string)) (string, error) {
	v := f(key)
	if v == "" {
		return "", ErrKeyNotFound
	}

	return v, nil
}
