//go:build windows

package wndws

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/go-errors/errors"
)

// https://learn.microsoft.com/en-us/windows/win32/api/shlwapi/nf-shlwapi-shcreatememstream
// IStream * SHCreateMemStream([in, optional] const BYTE *pInit, [in] UINT cbInit);
func ShCreateMemStream(data []byte) (uintptr, error) {
	ret, _, err := shlwapiSHCreateMemStreamProc.Call(
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
	)
	if err != nil && err != windows.ERROR_SUCCESS {
		return 0, err
	}
	if ret == 0 {
		return 0, errors.New(`SHCreateMemStream failed`)
	}
	return ret, nil
}
