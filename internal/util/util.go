package util

import (
	"fmt"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"

	"github.com/srlehn/termimg/internal/errors"
)

func MapsKeysSorted[M ~map[K]V, K constraints.Ordered, V any](m M) []K {
	if m == nil {
		return nil
	}
	keys := maps.Keys(m)
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func MaybeCast[T any](obj any) T {
	var ret T
	reflect.ValueOf(ret).Kind()
	objT, okT := any(obj).(T)
	if okT {
		ret = objT
	}
	return ret
}

func MaybeCastDefault[T any](obj any, fallback T) T {
	var ret T
	reflect.ValueOf(ret).Kind()
	objT, okT := any(obj).(T)
	if okT {
		ret = objT
	} else {
		ret = fallback
	}
	return ret
}

// AnyOf returns first non-nil object
func AnyOf[T any](objs ...T) (_ T, found bool) {
	var ret T
	for _, obj := range objs {
		if any(obj) != nil {
			return obj, true
		}
	}
	return ret, false
}

func TryClose(obj any) error {
	if obj == nil {
		return nil
	}
	closer, ok := any(obj).(interface{ Close() error })
	if ok {
		return closer.Close()
	}
	closerIrregular, okIrregular := any(obj).(interface{ Close() })
	if okIrregular {
		closerIrregular.Close()
	}
	return nil
}

func Must(err error) {
	if err != nil {
		err = errors.Wrap(err, 1)
		panic(err)
	}
}
func Must2[T any](obj T, err error) T {
	if err != nil {
		err = errors.Wrap(err, 1)
		panic(err)
	}
	return obj
}
func Must3[T, U any](obj1 T, obj2 U, err error) (T, U) {
	if err != nil {
		err = errors.Wrap(err, 1)
		panic(err)
	}
	return obj1, obj2
}
func Must4[T, U, V any](o1 T, o2 U, o3 V, err error) (T, U, V) {
	if err != nil {
		err = errors.Wrap(err, 1)
		panic(err)
	}
	return o1, o2, o3
}
func Must5[T, U, V, W any](o1 T, o2 U, o3 V, o4 W, err error) (T, U, V, W) {
	if err != nil {
		err = errors.Wrap(err, 1)
		panic(err)
	}
	return o1, o2, o3, o4
}
func IgnoreError[T any](obj T, err error) T {
	return obj
}

var modulePath string

func fileLinePrefix(skip int) string {
	if len(modulePath) == 0 {
		_, mp, _, okModule := runtime.Caller(0)
		if !okModule {
			return ``
		}
		mp = path.Dir(path.Dir(path.Dir(mp))) // depends on depth of this source file within the module!
		modulePath = mp
	}
	_, filename, line, okCaller := runtime.Caller(skip)
	if !okCaller {
		return ``
	}
	return strings.TrimPrefix(filename, modulePath+`/`) + `:` + strconv.Itoa(line) + `: `
}

func StringToBytes(s string) []byte { return unsafe.Slice(unsafe.StringData(s), len(s)) }
func BytesToString(b []byte) string { return unsafe.String(unsafe.SliceData(b), len(b)) }

func storePosAndJumpToPosStr(x, y uint) string { return fmt.Sprintf("\033[s\033[%d;%dH", y, x) }

const restorePosStr = "\033[u"

/*
func RestoreTTY(ptyName string) error {
	var errs []error

	// if unix
	err := restoreTTYUnix(ptyName)
	if err == nil {
		return nil
	}
	errs = append(errs, err)

	// try posix: stty echo

	errRet := errors.New(errors.Join(errs...))
	return errRet
}

const (
	// TODO build tags
	ioctlWriteTermios = unix.TCSETS // unix.TIOCSETA on BSD
	ioctlReadTermios  = unix.TCGETS // unix.TIOCGETA on BSD
)

func restoreTTYUnix(ptyName string) error {
	// TODO: tag unix && !windows && !plan9
	in, err := os.Open(ptyName)
	if err != nil {
		return err
	}

	termios, err := unix.IoctlGetTermios(int(in.Fd()), uint(ioctlReadTermios)) // get termios
	unix.IoctlSetTermios(int(in.Fd()), uint(ioctlWriteTermios), termios)       // set termios

	return nil
}
*/
