package errors

import (
	"errors"

	errorsGo "github.com/go-errors/errors"
)

var ErrUnsupported = errors.ErrUnsupported

func As(err error, target any) bool { return errors.As(err, target) }

func Is(err, target error) bool { return errors.Is(err, target) }

func Join(errs ...error) error {
	// not implemented by github.com/go-errors/errors
	if err := errors.Join(errs...); err != nil {
		return New(err)
	} else {
		return nil
	}
}

func New(obj any) *Error {
	// return nil for nil unlike github.com/go-errors/errors.New()
	if obj == nil {
		return nil
	}
	// don't overwrite origin of failure
	if errGo, okErrGo := obj.(*errorsGo.Error); okErrGo {
		return errGo
	}
	return errorsGo.Wrap(obj, 1)
}

func Unwrap(err error) error { return errors.Unwrap(err) }

// remaining "github.com/go-errors/errors" symbols

type Error = errorsGo.Error

func Errorf(format string, a ...interface{}) *Error { return errorsGo.Errorf(format, a...) }

func ParsePanic(text string) (*Error, error) { return errorsGo.ParsePanic(text) }

func Wrap(e interface{}, skip int) *Error { return errorsGo.Wrap(e, skip) }

func WrapPrefix(e interface{}, prefix string, skip int) *Error {
	return errorsGo.WrapPrefix(e, prefix, skip)
}

type StackFrame = errorsGo.StackFrame

func NewStackFrame(pc uintptr) (frame StackFrame) { return errorsGo.NewStackFrame(pc) }
