package errors

import (
	"errors"
)

var (
	ErrAlreadyExists   = errors.New("already exists")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrMissingArgument = errors.New("missing argument")
	ErrNotFound        = errors.New("not found")
)
