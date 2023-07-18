package util

import (
	"encoding/json"
	"fmt"
	"log"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"

	"github.com/go-errors/errors"
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

func TryClose(obj any) error {
	closer, ok := any(obj).(interface{ Close() error })
	if !ok {
		return nil
	}
	return closer.Close()
}

func Must(err error) {
	if err != nil {
		log.Fatalln(errors.Wrap(err, 1).ErrorStack())
	}
}
func Must2[T any](obj T, err error) T {
	var r T
	if err != nil {
		log.Fatalln(errors.Wrap(err, 1).ErrorStack())
		return r
	}
	return obj
}
func Must3[T, U any](obj1 T, obj2 U, err error) (T, U) {
	var r1 T
	var r2 U
	if err != nil {
		log.Fatalln(errors.Wrap(err, 1).ErrorStack())
		return r1, r2
	}
	return obj1, obj2
}
func Must4[T, U, V any](o1 T, o2 U, o3 V, err error) (T, U, V) {
	var r1 T
	var r2 U
	var r3 V
	if err != nil {
		log.Fatalln(errors.Wrap(err, 1).ErrorStack())
		return r1, r2, r3
	}
	return o1, o2, o3
}
func Must5[T, U, V, W any](o1 T, o2 U, o3 V, o4 W, err error) (T, U, V, W) {
	var r1 T
	var r2 U
	var r3 V
	var r4 W
	if err != nil {
		log.Fatalln(errors.Wrap(err, 1).ErrorStack())
		return r1, r2, r3, r4
	}
	return o1, o2, o3, o4
}
func Maybe2[T any](obj T, err error) T {
	return obj
}
func Maybe3[T, U any](obj1 T, obj2 U, err error) (T, U) {
	return obj1, obj2
}
func Maybe4[T, U, V any](o1 T, o2 U, o3 V, err error) (T, U, V) {
	return o1, o2, o3
}
func Maybe5[T, U, V, W any](o1 T, o2 U, o3 V, o4 W, err error) (T, U, V, W) {
	return o1, o2, o3, o4
}

func Println(a ...any) {
	_, modulepath, _, _ := runtime.Caller(0)
	modulepath = path.Dir(path.Dir(path.Dir(modulepath)))
	_, filename, line, _ := runtime.Caller(1)
	fmt.Println(append([]any{strings.TrimPrefix(filename, modulepath+`/`) + `:` + strconv.Itoa(line) + `: `}, a...)...)
}

func Printf(format string, a ...any) {
	_, modulepath, _, _ := runtime.Caller(0)
	modulepath = path.Dir(path.Dir(path.Dir(modulepath)))
	_, filename, line, _ := runtime.Caller(1)
	fmt.Printf(strings.TrimPrefix(filename, modulepath+`/`)+`:`+strconv.Itoa(line)+`: `+format, a...)
}

func Printfv(a ...any) {
	_, modulepath, _, _ := runtime.Caller(0)
	modulepath = path.Dir(path.Dir(path.Dir(modulepath)))
	_, filename, line, _ := runtime.Caller(1)
	sep := ` - `
	format := strings.TrimSuffix(strings.Repeat(`%+#v`+sep, len(a)), sep) + "\n"
	fmt.Printf(strings.TrimPrefix(filename, modulepath+`/`)+`:`+strconv.Itoa(line)+`: `+format, a...)
}

func Printfq(a ...any) {
	_, modulepath, _, _ := runtime.Caller(0)
	modulepath = path.Dir(path.Dir(path.Dir(modulepath)))
	_, filename, line, _ := runtime.Caller(1)
	sep := ` - `
	format := strings.TrimSuffix(strings.Repeat(`%q`+sep, len(a)), sep) + "\n"
	fmt.Printf(strings.TrimPrefix(filename, modulepath+`/`)+`:`+strconv.Itoa(line)+`: `+format, a...)
}

func Printfj(a ...any) {
	_, modulepath, _, _ := runtime.Caller(0)
	modulepath = path.Dir(path.Dir(path.Dir(modulepath)))
	_, filename, line, _ := runtime.Caller(1)
	sep := "\n"
	prefix := strings.TrimPrefix(filename, modulepath+`/`) + `:` + strconv.Itoa(line) + `: `
	var strs []string
	for _, o := range a {
		b, err := json.MarshalIndent(o, ``, `  `)
		if err != nil {
			strs = append(strs, prefix+fmt.Sprintf("error: %q\n  %v", err.Error(), o))
		} else {
			strs = append(strs, prefix+string(b))
		}
	}
	fmt.Println(strings.Join(strs, sep))
}

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

	errRet := errorsGo.New(errors.Join(errs...))
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
