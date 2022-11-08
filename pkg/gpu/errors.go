package gpu

import (
	"fmt"
)

type errorCode uint

const (
	errorCodeNotFound = iota
	errorCodeGeneric
)

var (
	NotFoundError = errorImpl{code: errorCodeNotFound}
	GenericError  = errorImpl{code: errorCodeGeneric}
)

type Error interface {
	error
	IsNotFound() bool
}

type errorImpl struct {
	code errorCode
	err  error
}

func (e errorImpl) Error() string {
	return e.err.Error()
}

func (e errorImpl) IsNotFound() bool {
	return e.code == errorCodeNotFound
}

func (e errorImpl) Errorf(format string, args ...any) Error {
	e.err = fmt.Errorf(format, args...)
	return e
}

func IgnoreNotFound(err Error) Error {
	if err == nil {
		return nil
	}
	if err.IsNotFound() {
		return nil
	}
	return err
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	gpuErr, ok := err.(Error)
	if !ok {
		return false
	}
	return gpuErr.IsNotFound()
}

func NewGenericError(err error) Error {
	return errorImpl{
		err:  err,
		code: errorCodeGeneric,
	}
}
