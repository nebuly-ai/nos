package gpu

import "fmt"

type errorCode uint

const (
	errorCodeNotFound = iota
	errorCodeGeneric
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

func Errorf(format string, args ...any) Error {
	return errorImpl{
		err:  fmt.Errorf(format, args...),
		code: errorCodeGeneric,
	}
}

func NewGenericError(err error) Error {
	return errorImpl{
		err:  err,
		code: errorCodeGeneric,
	}
}
