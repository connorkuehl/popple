package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrUnknownKey indicates the config file contains an unexpected/unknown
// key.
type ErrUnknownKey struct {
	Line int
	Key  string
}

// Error returns the error string for ErrUnknownKey types.
func (e ErrUnknownKey) Error() string {
	return "unknown config key"
}

// ErrMissingValue indicates that a value was not supplied with a given config
// key.
type ErrMissingValue struct {
	Line   int
	ForKey string
}

// Error returns the error string for ErrMissingValue instances.
func (e ErrMissingValue) Error() string {
	return "missing value for config key"
}

// Config holds the program's configuration.
type Config struct {
	Token      string
	DBPath     string
	HTTPHealth string
}

// Load extracts the config key, value pairs from the given reader.
//
// Lines starting with "#" are considered comments and are ignored.
func Load(r io.Reader) (Config, error) {
	var config Config

	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)

	var lineNumber int
	for ; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())

		// empty line or comment
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		wordScanner := bufio.NewScanner(strings.NewReader(line))
		wordScanner.Split(bufio.ScanWords)

		if ok := wordScanner.Scan(); !ok {
			return Config{}, fmt.Errorf("expected config key: %w", wordScanner.Err())
		}
		key := wordScanner.Text()

		if ok := wordScanner.Scan(); !ok {
			return Config{}, ErrMissingValue{Line: lineNumber, ForKey: key}
		}
		val := wordScanner.Text()

		switch key {
		case "token":
			config.Token = val
		case "database":
			config.DBPath = val
		case "http_health":
			config.HTTPHealth = val
		default:
			err := ErrUnknownKey{
				Key:  key,
				Line: lineNumber,
			}
			return Config{}, err
		}
	}

	return config, nil
}

// LoadFromFile is a convenience function for reading a Config from a file.
func LoadFromFile(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	return Load(f)
}
