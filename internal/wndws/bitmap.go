//go:build windows

package wndws

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/go-errors/errors"
	"github.com/gonutz/w32/v2"
	"github.com/lxn/win"
)

type GpBitmap win.GpBitmap

// based on github.com/AllenDang/w32.GdipCreateBitmapFromStream
func GetGpBitmapFromBytes(data []byte) (gpBmp *GpBitmap, close func() error, e error) {
	imgStreamUintPtr, err := ShCreateMemStream(data)
	if err != nil {
		return nil, nil, errors.New(err)
	}
	imgStream := (*w32.IStream)(unsafe.Pointer(imgStreamUintPtr))
	// TODO: defer imgStream.Release() in caller
	closeFunc := func() error {
		_ = imgStream.Release()
		return nil
	}

	gpBmpPtr, err := w32.GdipCreateBitmapFromStream(imgStream)
	if err != nil {
		_ = closeFunc()
		return nil, nil, errors.New(err)
	}
	gpBmp = (*GpBitmap)(unsafe.Pointer(gpBmpPtr))
	return gpBmp, closeFunc, nil
}

func GetGpBitmapFromFile(filename string) (*GpBitmap, error) {
	var gpBmp *GpBitmap
	filePathUTF16, err := syscall.UTF16FromString(filename)
	if err != nil {
		return nil, errors.New(err)
	}
	filePathUTF16Ptr := &filePathUTF16[0]
	gpBmpWin := (*win.GpBitmap)(gpBmp)
	if status := win.GdipCreateBitmapFromFile(filePathUTF16Ptr, &gpBmpWin); status != win.Ok {
		return nil, errors.New(fmt.Sprintf("GdipCreateBitmapFromFile failed with status '%s' for file '%s'", status, filename))
	}
	return gpBmp, nil
}

func (img *GpBitmap) Dispose() win.GpStatus {
	return win.GdipDisposeImage((*win.GpImage)(img))
}

// gdi bitmaps
// https://www-user.tu-chemnitz.de/~heha/petzold/ch14e.htm
