//go:build windows

package wndws

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/go-errors/errors"
	// "github.com/lxn/win"
)

// https://stackoverflow.com/a/33652557
// https://learn.microsoft.com/en-us/windows/console/getcurrentconsolefont
// BOOL WINAPI GetCurrentConsoleFont(HANDLE hConsoleOutput, BOOL bMaximumWindow, PCONSOLE_FONT_INFO lpConsoleCurrentFont)
func GetCurrentConsoleFont(hConsoleOutput windows.Handle) (*ConsoleFontInfo, error) {
	var bMaximumWindow int
	lpConsoleCurrentFont := &ConsoleFontInfo{}
	ret, _, _ := getCurrentConsoleFontProc.Call(uintptr(hConsoleOutput), uintptr(bMaximumWindow), uintptr(unsafe.Pointer(lpConsoleCurrentFont)))
	if ret == 0 || lpConsoleCurrentFont == nil {
		return nil, errors.New(`GetCurrentConsoleFont failed`)
	}
	return lpConsoleCurrentFont, nil
}

// https://learn.microsoft.com/en-us/windows/console/console-font-info-str
type ConsoleFontInfo struct {
	NFont      uint32 //  The index of the font in the system's console font table
	DwFontSize CoOrd
}
