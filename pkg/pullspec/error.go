package pullspec

import (
	"errors"
	"fmt"
)

var (
	// ErrNotClusterServiceVersion is the error returned with a source isn't a CSV.
	ErrNotClusterServiceVersion = errors.New("Not a ClusterServiceVersion")

	// ErrNotFound is the error returned when a file is not found
	ErrNotFound = errors.New("path not found")

	// ErrPathExpectedDifferentType is the error returned when the path expected a different type.
	ErrPathExpectedDifferentType = errors.New("path expected different type")
)

type errBase struct {
	cause error
	err   error
}

func newError(cause error, format string, args ...interface{}) error {
	return errBase{
		err:   fmt.Errorf(format, args...),
		cause: cause,
	}
}

func (e errBase) Error() string {
	return e.err.Error()
}

func (e errBase) Unwrap() error {
	return e.cause
}
