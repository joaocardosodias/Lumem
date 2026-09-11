package usecase

import "fmt"

type permanentIngestionError struct {
	cause error
}

func (e *permanentIngestionError) Error() string { return e.cause.Error() }
func (e *permanentIngestionError) Unwrap() error { return e.cause }

func permanentIngestionFailure(format string, args ...any) error {
	return &permanentIngestionError{cause: fmt.Errorf(format, args...)}
}
