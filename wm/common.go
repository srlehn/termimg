package wm

import (
	"image"

	"github.com/srlehn/termimg/internal/environ"
)

type Implementation interface {
	Name() string
	Conn() (Connection, error)
	CreateWindow(env environ.Proprietor, name, class, instance string, isWindow IsWindowFunc) Window
}

type Connection interface {
	Close() error
	Conn() any
	Windows() ([]Window, error)
	DisplayImage(img image.Image, windowName string)
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

type IsWindowFunc = func(Window) (is bool, p environ.Proprietor)

// WindowProvider...
//
// env contains infos about the window:
// env.LookupEnv(): "WINDOWID"
// env.Property(): propkeys.TerminalPID ("general_termPID")
type WindowProvider = func(isWindow IsWindowFunc, env environ.Proprietor) Window

var implem Implementation

func SetImpl(impl Implementation) {
	if impl != nil {
		implem = impl
	}
}

func NewWindow(isWindow IsWindowFunc, env environ.Proprietor) Window {
	return implem.CreateWindow(env, ``, ``, ``, isWindow)
}

func CreateWindow(name, class, instance string) Window {
	return implem.CreateWindow(nil, ``, ``, ``, nil)
}

func NewConn() (Connection, error) { return implem.Conn() }
