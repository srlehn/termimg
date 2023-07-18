//go:build windows

package wndws

// Windows types:
// https://learn.microsoft.com/en-us/windows/win32/winprog/windows-data-types
// https://justen.codes/breaking-all-the-rules-using-go-to-call-windows-api-2cbfd8c79724
type (
	BOOL          uint32
	BOOLEAN       byte
	BYTE          byte
	DWORD         uint32
	DWORD64       uint64
	HANDLE        uintptr
	HLOCAL        uintptr
	LARGE_INTEGER int64
	LONG          int32
	LPVOID        uintptr
	SIZE_T        uintptr
	UINT          uint32
	ULONG_PTR     uintptr
	ULONGLONG     uint64
	WORD          uint16
)

type CoOrd struct {
	// A COORD structure that contains the width and height of each character in the font,
	// in logical units.
	// The X member contains the width, while the Y member contains the height.
	X int16
	Y int16
}
