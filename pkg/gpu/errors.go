package gpu

import "fmt"

type errorCode uint

const (
	errorCodeNotFound = iota
	errorCodeGeneric
)

var (
	NotFoundError = errorImpl{code: errorCodeNotFound}
	GenericError  = errorImpl{code: errorCodeNotFound}
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

func NewGenericError(err error) Error {
	return errorImpl{
		err:  err,
		code: errorCodeGeneric,
	}
}
