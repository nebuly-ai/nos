package gpu

import (
	"fmt"
	"strings"
)

type errorCode string

const (
	errorCodeNotFound = "resource-not-found"
	errorCodeGeneric  = "generic"
)

var (
	NotFoundErr = errorImpl{code: errorCodeNotFound}
	GenericErr  = errorImpl{code: errorCodeGeneric}
)

type Error interface {
	error
	IsNotFound() bool
}

type ErrorList []Error

func (l ErrorList) Error() string {
	if len(l) == 0 {
		return "no errors"
	}
	sb := strings.Builder{}
	sb.WriteString("errors: ")
	for _, e := range l {
		sb.WriteString(fmt.Sprintf("{ %s } ", e))
	}
	return sb.String()
}

type errorImpl struct {
	code errorCode
	err  error
}

func (e errorImpl) Error() string {
	return fmt.Sprintf("code: %s err: %s", e.code, e.err.Error())
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
