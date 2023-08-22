// Copyright 2013 Konstantin Kulikov. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package framebuffer is an interface to linux framebuffer device.
package framebuffer

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"syscall"
	"unsafe"

	"github.com/srlehn/termimg/internal/errors"
)

// Framebuffer contains information about framebuffer.
type Framebuffer struct {
	dev   *os.File
	finfo fixedScreenInfo
	vinfo variableScreenInfo
	data  []byte
}

// Init opens framebuffer device, maps it to memory and saves its current contents.
func Init(dev string) (*Framebuffer, error) {
	var (
		fb  = new(Framebuffer)
		err error
	)
	fb.dev, err = os.OpenFile(dev, os.O_RDWR, os.ModeDevice)
	if err != nil {
		return nil, errors.New(err)
	}
	err = ioctl(fb.dev.Fd(), getFixedScreenInfo, unsafe.Pointer(&fb.finfo))
	if err != nil {
		fb.dev.Close()
		return nil, errors.New(err)
	}
	err = ioctl(fb.dev.Fd(), getVariableScreenInfo, unsafe.Pointer(&fb.vinfo))
	if err != nil {
		fb.dev.Close()
		return nil, errors.New(err)
	}
	fb.data, err = syscall.Mmap(int(fb.dev.Fd()), 0, int(fb.finfo.Smem_len+uint32(fb.finfo.Smem_start&uint64(syscall.Getpagesize()-1))), protocolRead|protocolWrite, mapShared)
	if err != nil {
		fb.dev.Close()
		return nil, errors.New(err)
	}
	return fb, nil
}

// Close closes framebuffer device and restores its contents.
func (fb *Framebuffer) Close() error {
	if fb == nil {
		return nil
	}
	err := errors.Join(syscall.Munmap(fb.data), fb.dev.Close())
	if err != nil {
		return errors.New(err)
	}
	return nil
}

var _ image.Image = (*Framebuffer)(nil)

func (fb *Framebuffer) ColorModel() color.Model { return color.NRGBAModel }

// Size returns dimensions of a framebuffer.
func (fb *Framebuffer) Bounds() image.Rectangle {
	if fb == nil {
		return image.Rectangle{}
	}
	return image.Rectangle{Max: image.Point{X: int(fb.vinfo.Xres), Y: int(fb.vinfo.Yres)}}
}

func (fb *Framebuffer) At(x, y int) color.Color {
	if fb == nil || !(image.Point{x, y}.In(fb.Bounds())) {
		return color.RGBA{}
	}
	offset := (int(fb.vinfo.Xoffset)+x)*(int(fb.vinfo.Bits_per_pixel)/8) +
		(int(fb.vinfo.Yoffset)+y)*int(fb.finfo.Line_length)
	c := color.NRGBA{
		R: fb.data[offset+2],
		G: fb.data[offset+1],
		B: fb.data[offset],
		A: 255,
	}
	return c
}

var _ draw.Image = (*Framebuffer)(nil)

// Set changes pixel at x, y to specified color.
func (fb *Framebuffer) Set(x, y int, c color.Color) {
	if fb == nil || c == nil || !(image.Point{x, y}.In(fb.Bounds())) {
		return
	}
	offset := (int(fb.vinfo.Xoffset)+x)*(int(fb.vinfo.Bits_per_pixel)/8) +
		(int(fb.vinfo.Yoffset)+y)*int(fb.finfo.Line_length)
	red, green, blue, alpha := c.RGBA()
	fb.data[offset] = byte(blue)
	fb.data[offset+1] = byte(green)
	fb.data[offset+2] = byte(red)
	fb.data[offset+3] = byte(alpha)
}

// Clear fills screen with specified color
func (fb *Framebuffer) Clear(c color.Color) {
	if fb == nil {
		return
	}
	bounds := fb.Bounds()
	for x := 0; x < bounds.Dx(); x++ {
		for y := 0; y < bounds.Dy(); y++ {
			fb.Set(x, y, c)
		}
	}
}

func ioctl(fd uintptr, cmd uintptr, data unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, uintptr(data))
	if errno != 0 {
		if err := os.NewSyscallError("IOCTL", errno); err != nil {
			return errors.New(err)
		}
		return nil
	}
	return nil
}

// TODO other pixel formats
