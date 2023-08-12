// This file is subject to a 1-clause BSD license.
// Its contents can be found in the enclosed LICENSE file.

package framebuffer

import (
	"fmt"
	"syscall"
	"unsafe"
)

func ioctl(fd, name uintptr, data interface{}) error {
	var v uintptr

	switch dd := data.(type) {
	case unsafe.Pointer:
		v = uintptr(dd)

	case int:
		v = uintptr(dd)

	case uintptr:
		v = dd

	default:
		return fmt.Errorf("ioctl: Invalid argument.")
	}

	_, _, errno := syscall.RawSyscall(syscall.SYS_IOCTL, fd, name, v)
	if errno == 0 {
		return nil
	}

	return errno
}

const (
	_IOC_NONE      = 0x0
	_IOC_WRITE     = 0x1
	_IOC_READ      = 0x2
	_IOC_NRBITS    = 8
	_IOC_TYPEBITS  = 8
	_IOC_SIZEBITS  = 14
	_IOC_DIRBITS   = 2
	_IOC_NRSHIFT   = 0
	_IOC_NRMASK    = (1 << _IOC_NRBITS) - 1
	_IOC_TYPEMASK  = (1 << _IOC_TYPEBITS) - 1
	_IOC_SIZEMASK  = (1 << _IOC_SIZEBITS) - 1
	_IOC_DIRMASK   = (1 << _IOC_DIRBITS) - 1
	_IOC_TYPESHIFT = _IOC_NRSHIFT + _IOC_NRBITS
	_IOC_SIZESHIFT = _IOC_TYPESHIFT + _IOC_TYPEBITS
	_IOC_DIRSHIFT  = _IOC_SIZESHIFT + _IOC_SIZEBITS
	_IOC_IN        = _IOC_WRITE << _IOC_DIRSHIFT
	_IOC_OUT       = _IOC_READ << _IOC_DIRSHIFT
	_IOC_INOUT     = (_IOC_WRITE | _IOC_READ) << _IOC_DIRSHIFT
	_IOCSIZE_MASK  = _IOC_SIZEMASK << _IOC_SIZESHIFT
)

func _IOC(dir, t, nr, size int) int {
	return (dir << _IOC_DIRSHIFT) | (t << _IOC_TYPESHIFT) |
		(nr << _IOC_NRSHIFT) | (size << _IOC_SIZESHIFT)
}

func _IO(t, nr int) int {
	return _IOC(_IOC_NONE, t, nr, 0)
}

func _IOR(t, nr, size int) int {
	return _IOC(_IOC_READ, t, nr, size)
}

func _IOW(t, nr, size int) int {
	return _IOC(_IOC_WRITE, t, nr, size)
}

func _IOWR(t, nr, size int) int {
	return _IOC(_IOC_READ|_IOC_WRITE, t, nr, size)
}

func _IOC_DIR(nr int) int {
	return ((nr) >> _IOC_DIRSHIFT) & _IOC_DIRMASK
}

func _IOC_TYPE(nr int) int {
	return ((nr) >> _IOC_TYPESHIFT) & _IOC_TYPEMASK
}

func _IOC_NR(nr int) int {
	return ((nr) >> _IOC_NRSHIFT) & _IOC_NRMASK
}

func _IOC_SIZE(nr int) int {
	return ((nr) >> _IOC_SIZESHIFT) & _IOC_SIZEMASK
}
