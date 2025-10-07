package types

import "errors"

var (
	// ErrInvalidConfig is returned when configuration is invalid
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrNotFound is returned when a resource is not found
	ErrNotFound = errors.New("not found")

	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("operation timed out")

	// ErrInvalidSignature is returned when a signature is invalid
	ErrInvalidSignature = errors.New("invalid signature")

	// ErrTaskFailed is returned when a task fails
	ErrTaskFailed = errors.New("task failed")
)
