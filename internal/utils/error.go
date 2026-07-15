// Package utils provides shared utility types and functions.
package utils

import (
	"errors"
	"fmt"
	"os"
)

var (
	// ErrNotClusterServiceVersion is the error returned with a source isn't a CSV.
	ErrNotClusterServiceVersion = errors.New("not a ClusterServiceVersion")

	// ErrNotFound is the error returned when a file is not found
	ErrNotFound = errors.New("path not found")

	// ErrPathExpectedDifferentType is the error returned when the path expected a different type.
	ErrPathExpectedDifferentType = errors.New("path expected different type")

	// ErrNoOperatorManifests is returned when no CSV is found in the operator manifests.
	ErrNoOperatorManifests = errors.New("missing ClusterServiceVersion in operator manifests")

	// ErrTooManyCSVs is returned when more than one CSV file is found in an operator bundle.
	ErrTooManyCSVs = errors.New("operator bundle may contain only 1 CSV file, but contains more")

	// ErrImageIsARequiredProperty is returned when image is missing from a pullspec.
	ErrImageIsARequiredProperty = errors.New("'image' is a required property")
)

type errBase struct {
	cause error
	err   error
}

// NewError creates a new wrapped error with a formatted message.
func NewError(cause error, format string, args ...any) error {
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

// NewErrIsNotDirectoryOrDoesNotExist returns an error indicating the path is not a directory or doesn't exist.
func NewErrIsNotDirectoryOrDoesNotExist(path string) error {
	return errors.New(path + " is not a directory or does not exist")
}

// CheckIfDirectoryExists verifies that the given path is an existing directory.
func CheckIfDirectoryExists(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrIsNotDirectoryOrDoesNotExist(path)
		}

		return err
	}

	if !stat.IsDir() {
		return NewErrIsNotDirectoryOrDoesNotExist(path)
	}

	return nil
}

// NewErrImageDoesNotExist returns an error indicating that the image could not be inspected.
func NewErrImageDoesNotExist(imageName string, err error) error {
	return fmt.Errorf("failed to inspect %s: make sure it exists and is accessible: %w", imageName, err)
}
