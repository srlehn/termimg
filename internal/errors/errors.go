package errors

import (
	"errors"
	"runtime"

	errorsGo "github.com/go-errors/errors"
)

var ErrUnsupported = errors.ErrUnsupported

func As(err error, target any) bool { return errorsGo.As(err, target) }

func Is(err, target error) bool { return errorsGo.Is(err, target) }

func Join(errs ...error) error {
	// not implemented by github.com/go-errors/errors
	if err := errorsGo.Join(errs...); err != nil {
		if errGo, okErrGo := err.(*errorsGo.Error); okErrGo {
			return errGo
		}
		return errorsGo.Wrap(err, 1)
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

func Unwrap(err error) error { return errorsGo.Unwrap(err) }

// remaining "github.com/go-errors/errors" symbols

type Error = errorsGo.Error

func Errorf(format string, a ...interface{}) *Error { return errorsGo.Errorf(format, a...) }

func ParsePanic(text string) (*Error, error) { return errorsGo.ParsePanic(text) }

func Wrap(e interface{}, skip int) *Error { return errorsGo.Wrap(e, skip+1) }

func WrapPrefix(e interface{}, prefix string, skip int) *Error {
	return errorsGo.WrapPrefix(e, prefix, skip)
}

type StackFrame = errorsGo.StackFrame

func NewStackFrame(pc uintptr) (frame StackFrame) { return errorsGo.NewStackFrame(pc) }

// NilReceiver returns an error with the function name if any of the arguments are nil
func NilReceiver(args ...any) error {
	return errMsgNilTester(`nil receiver or struct field`, 3, args...)
}

// NilParam returns an error with the function name if any of the arguments are nil
func NilParam(args ...any) error {
	return errMsgNilTester(`nil parameter`, 3, args...)
}

// NotImplemented returns an error with the function name
func NotImplemented() error {
	return errMsgNilTester(`not implemented`, 3)
}

func errMsgNilTester(msg string, skip int, args ...any) error {
	for i := range args {
		if args[i] == nil {
			goto anyNil
		}
	}
	return nil
anyNil:
	return errMsg(msg, skip)
}

func errMsg(msg string, skip int) error {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return Wrap(msg, skip)
	}
	return Wrap(msg+`: `+runtime.FuncForPC(pc).Name()+`()`, skip)
}
