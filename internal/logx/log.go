package logx

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/srlehn/termimg/internal/errors"
)

func Log(msg string, logger *slog.Logger, lvl slog.Level, skip int, args ...any) {
	if logger == nil || !logger.Enabled(context.Background(), slog.LevelInfo) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(skip, pcs[:])
	r := slog.NewRecord(time.Now(), lvl, msg, pcs[0])
	_ = logger.With(args...).Handler().Handle(context.Background(), r)
}

func Debug(msg string, loggerProv LoggerProvider, args ...any) {
	if loggerProv == nil {
		return
	}
	Log(msg, loggerProv.Logger(), slog.LevelDebug, 3, args...)
}
func Info(msg string, loggerProv LoggerProvider, args ...any) {
	if loggerProv == nil {
		return
	}
	Log(msg, loggerProv.Logger(), slog.LevelInfo, 3, args...)
}
func Warn(msg string, loggerProv LoggerProvider, args ...any) {
	if loggerProv == nil {
		return
	}
	Log(msg, loggerProv.Logger(), slog.LevelWarn, 3, args...)
}
func Error(msg string, loggerProv LoggerProvider, args ...any) {
	if loggerProv == nil {
		return
	}
	Log(msg, loggerProv.Logger(), slog.LevelError, 3, args...)
}

func IsErr(err error, loggerProv LoggerProvider, lvl slog.Level, args ...any) bool {
	if err != nil {
		if loggerProv != nil {
			logger := loggerProv.Logger()
			if logger != nil {
				if errs, ok := err.(interface{ Unwrap() []error }); ok {
					for _, err := range errs.Unwrap() {
						Log(err.Error(), logger, lvl, 3, args...)
					}
				} else {
					Log(err.Error(), logger, lvl, 3, args...)
				}
			}
		}
		return true
	}
	return false
}

func Err(err error, loggerProv LoggerProvider, lvl slog.Level, args ...any) error {
	if IsErr(err, loggerProv, lvl, args...) {
		return err
	}
	return nil
}

func TimeIt(fn func() error, msg string, loggerProv LoggerProvider, args ...any) error {
	if fn == nil {
		return errors.New(`provided nil func`)
	}
	if len(msg) == 0 {
		msg = `duration measurement for function`
	}
	start := time.Now()
	err := fn()
	Info(msg, loggerProv, append([]any{`duration`, time.Since(start)}, args...)...)
	return err
}

func TimeIt2[T any](fn func() (T, error), msg string, loggerProv LoggerProvider, args ...any) (T, error) {
	var ret T
	if fn == nil {
		return ret, errors.New(`provided nil func`)
	}
	if len(msg) == 0 {
		msg = `duration measurement for function`
	}
	start := time.Now()
	ret, err := fn()
	Info(msg, loggerProv, append([]any{`duration`, time.Since(start)}, args...)...)
	return ret, err
}

type LoggerProvider interface{ Logger() *slog.Logger }

var _ LoggerProvider = (*loggerProvider)(nil)

type loggerProvider struct{ logger *slog.Logger }

func (p *loggerProvider) Logger() *slog.Logger { return p.logger }

func Prov(logger *slog.Logger) LoggerProvider { return &loggerProvider{logger: logger} }
