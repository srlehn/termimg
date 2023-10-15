package wm

import (
	"image"

	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
)

type Implementation interface {
	Name() string
	Conn(env environ.Properties) (Connection, error)
	CreateWindow(env environ.Properties, name, class, instance string, isWindow IsWindowFunc) Window
}

type Connection interface {
	Close() error
	Conn() any
	Windows() ([]Window, error)
	DisplayImage(img image.Image, windowName string)
	Resources() (environ.Properties, error)
}

type Window interface {
	WindowConn() Connection
	WindowFind() error
	WindowType() string // x11, windows, ...
	WindowName() string
	WindowClass() string
	WindowInstance() string
	WindowID() uint64
	WindowPID() uint64
	DeviceContext() uintptr
	Screenshot() (image.Image, error)
	Close() error
}

// type HWND uintptr
// type HDC uintptr

type IsWindowFunc = func(Window) (is bool, p environ.Properties)

// WindowProvider...
//
// env contains infos about the window:
// env.LookupEnv(): "WINDOWID"
// env.Property(): propkeys.TerminalPID ("general_termPID")
type WindowProvider = func(isWindow IsWindowFunc, env environ.Properties) Window

var implem Implementation

func SetImpl(impl Implementation) {
	if impl != nil {
		implem = impl
	}
}

func NewWindow(isWindow IsWindowFunc, env environ.Properties) Window {
	if implem == nil {
		return nil
	}
	return implem.CreateWindow(env, ``, ``, ``, isWindow)
}

func CreateWindow(name, class, instance string) Window {
	if implem == nil {
		return nil
	}
	return implem.CreateWindow(nil, name, class, instance, nil)
}

func NewConn(env environ.Properties) (Connection, error) {
	if implem == nil {
		return nil, errors.New(`no wm.Implementation set`)
	}
	return implem.Conn(env)
}
