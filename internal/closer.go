package internal

import (
	"errors"
	"reflect"
	"runtime"
	"sync"

	errorsGo "github.com/go-errors/errors"
)

type Closer interface {
	Close() error
	OnClose(onClose func() error)
	AddClosers(closers ...interface{ Close() error })
}

var _ Closer = (*lifoCloser)(nil)

type lifoCloser struct {
	initObjsMu   sync.Mutex
	onCloseFuncs []func() error
	initObjs     map[initObjKey]struct{}
}

type initObjKey struct {
	p uintptr
	t string
}

func NewCloser() Closer { return newLifoCloser() }

func newLifoCloser() *lifoCloser {
	closer := &lifoCloser{}
	runtime.SetFinalizer(closer, func(cl *lifoCloser) { _ = cl.Close() })
	return closer
}

func (c *lifoCloser) Close() error {
	if c == nil {
		return nil
	}
	var errs []error
	for i := len(c.onCloseFuncs) - 1; i > -1; i-- {
		if onCloseFunc := c.onCloseFuncs[i]; onCloseFunc != nil {
			if err := onCloseFunc(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	c = nil
	return errorsGo.New(errors.Join(errs...))
}

func (c *lifoCloser) OnClose(onClose func() error) {
	if c == nil {
		// TODO log
		return
	}
	c.onCloseFuncs = append(c.onCloseFuncs, onClose)
}

func (c *lifoCloser) AddClosers(closers ...interface{ Close() error }) {
	if c == nil {
		// TODO log
		return
	}
	if len(closers) == 0 {
		return
	}
	c.initObjsMu.Lock()
	defer c.initObjsMu.Unlock()
	if c.initObjs == nil {
		c.initObjs = make(map[initObjKey]struct{})
	}
	for _, cl := range closers {
		cl := cl
		if cl == nil {
			continue
		}
		objType := reflect.TypeOf(cl)
		var ptr any = cl
		switch objType.Kind() {
		// don't use slice, map, func as map keys
		case reflect.Slice, reflect.Map, reflect.Func:
			ptr = &cl
		}
		key := initObjKey{p: reflect.ValueOf(ptr).Pointer(), t: objType.String()}
		_, alreadyAdded := c.initObjs[key]
		if alreadyAdded {
			continue
		}
		c.OnClose(func() error {
			if cl == nil {
				return nil
			}
			err := cl.Close()
			defer func() {
				c.initObjsMu.Lock()
				defer c.initObjsMu.Unlock()
				delete(c.initObjs, key)
			}()
			if err != nil {
				return errorsGo.New(err)
			}
			return nil
		})
		c.initObjs[key] = struct{}{}
	}
}
