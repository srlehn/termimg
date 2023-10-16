package log

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

const badKey = "!BADKEY"

// ripped from log/slog/record.go (BSD-3 license)
//
// logArgsToAttr turns a prefix of the nonempty args slice into an slog.Attr
// and returns the unconsumed portion of the slice.
// If args[0] is an slog.Attr, it returns it.
// If args[0] is a string, it treats the first two elements as
// a key-value pair.
// Otherwise, it treats args[0] as a value with a missing key.
func logArgsToAttr(args []any) (slog.Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return slog.String(badKey, x), nil
		}
		return slog.Any(x, args[1]), args[2:]

	case slog.Attr:
		return x, args[1:]

	default:
		return slog.Any(badKey, x), args[1:]
	}
}
func logArgsToAttrSlice(args []any) []slog.Attr {
	var (
		attr  slog.Attr
		attrs []slog.Attr
	)
	for len(args) > 0 {
		attr, args = logArgsToAttr(args)
		attrs = append(attrs, attr)
	}
	return attrs
}

func Log(logger *slog.Logger, lvl slog.Level, skip int, msg string, args ...any) {
	if logger == nil || !logger.Enabled(context.Background(), slog.LevelInfo) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(skip, pcs[:])
	r := slog.NewRecord(time.Now(), lvl, msg, pcs[0])
	_ = logger.With(logArgsToAttrSlice(args)).Handler().Handle(context.Background(), r)
}

func IsErr(logger *slog.Logger, lvl slog.Level, err error, args ...any) bool {
	if err != nil {
		if logger != nil {
			if errs, ok := err.(interface{ Unwrap() []error }); ok {
				for _, err := range errs.Unwrap() {
					Log(logger, lvl, 3, err.Error(), args...)
				}
			} else {
				Log(logger, lvl, 3, err.Error(), args...)
			}
		}
		return true
	}
	return false
}
