//go:build windows

package wndws

import (
	"golang.org/x/sys/windows"
)

var (
	shlwapiDLL  = windows.NewLazySystemDLL("shlwapi")
	kernel32DLL = windows.NewLazySystemDLL("kernel32.dll")
	ntDLL       = windows.NewLazySystemDLL("ntdll.dll")
)

var (
	shlwapiSHCreateMemStreamProc = shlwapiDLL.NewProc("SHCreateMemStream")
	getCurrentConsoleFontProc    = kernel32DLL.NewProc("GetCurrentConsoleFont")
	wineVerProc                  = ntDLL.NewProc("wine_get_version")
)
