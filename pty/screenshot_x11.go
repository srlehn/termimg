//go:build !windows && !android && !darwin && !js

package pty

import (
	"bytes"
	"image"
	"sync"
	"time"

	imagingOrig "github.com/kovidgoyal/imaging"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

// DrawFuncOnlyPicture ...
func DrawFuncOnlyPicture(img image.Image, cellBounds image.Rectangle) DrawFunc {
	return func(tm *term.Terminal, dr term.Drawer, rsz term.Resizer, cpw, cph uint) (areaOfInterest image.Rectangle, scaleX, scaleY float64, e error) {
		if img == nil {
			return image.Rectangle{}, 0, 0, errors.New(consts.ErrNilImage)
		}
		if tm == nil || rsz == nil {
			return image.Rectangle{}, 0, 0, errors.NilParam()
		}
		if cpw == 0 || cph == 0 {
			return image.Rectangle{}, 0, 0, errors.New(`cell box side length of 0`)
		}
		if cellBounds.Dx() == 0 || cellBounds.Dy() == 0 {
			return image.Rectangle{}, 0, 0, errors.New(`area of size 0`)
		}
		var (
			waitingTimeDrawing = 3000 * time.Millisecond
		)

		imgSize := img.Bounds().Size()
		scaleX = float64(imgSize.X) / float64(int(cpw)*cellBounds.Dx())
		scaleY = float64(imgSize.Y) / float64(int(cph)*cellBounds.Dy())

		if err := term.Draw(img, cellBounds, tm, dr); err != nil {
			return image.Rectangle{}, 0, 0, err
		}
		time.Sleep(waitingTimeDrawing)

		return cellBounds, scaleX, scaleY, nil
	}
}

// TakeScreenshot ...
func TakeScreenshot(termName string, termProvider TermProviderFunc, drawerName string, drawFuncProvider DrawFuncProvider, imgBytes []byte, cellBounds image.Rectangle, rsz term.Resizer) (image.Image, error) {
	var termChecker term.TermChecker
	if len(termName) == 0 {
		wm.SetImpl(wmimpl.Impl())
		tm, err := termimg.Terminal()
		if err != nil {
			return nil, err
		}
		defer termimg.CleanUp()
		termChecker = term.RegisteredTermChecker(tm.Name())
	} else {
		termChecker = term.RegisteredTermChecker(termName)
	}
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, errors.New(err)
	}

	var imgRet image.Image
	imgRetChan := make(chan image.Image)
	var errRet error

	x11ScrFunc := TakeScreenshotFunc(termProvider, termChecker, drawerName, drawFuncProvider(img, cellBounds), rsz, imgRetChan)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		imgRet = <-imgRetChan
		if imgRet == nil {
			errRet = errors.New(consts.ErrNilImage)
		}
		wg.Done()
	}()

	if len(termName) == 0 {
		if err := x11ScrFunc(``, 0); err != nil {
			if err != nil {
				return nil, err
			}
		}
	} else {
		var exeName string
		if exer, okExe := termChecker.(interface {
			Exe(term.Properties) string
		}); okExe {
			exeName = exer.Exe(nil) // TODO
		} else {
			exeName = termName
		}

		termCmd := []string{exeName}
		if arger, okArg := termChecker.(internal.Arger); okArg && arger != nil {
			termCmd = append(termCmd, arger.Args()...)
		}
		if err := PTYRun(x11ScrFunc, termCmd...); err != nil {
			return nil, err
		}
	}

	wg.Wait()

	if errRet != nil {
		return nil, errRet
	}

	return imgRet, nil
}

type imageWithMetadata struct {
	termName   string
	drawerName string
	image.Image
}

func (i *imageWithMetadata) TerminalName() string {
	if i == nil {
		return ``
	}
	return i.termName
}

func (i *imageWithMetadata) DrawerName() string {
	if i == nil {
		return ``
	}
	return i.drawerName
}

// TakeScreenshotFunc displays an image in a pseudo-terminal and captures the displayed version for comparison.
func TakeScreenshotFunc(termProvider TermProviderFunc, termChecker term.TermChecker, drawerName string, drawFunc DrawFunc, rsz term.Resizer, imgChan chan<- image.Image) PTYRunFunc {
	// TODO sometimes screenshots wrong window when using x11 drawer
	return func(pty string, pid uint) (errRet error) {
		var (
			cpw, cph                                            float64
			tpw, tph, edgeThickness, menuBarHeight, extraBorder uint
			scaleX, scaleY                                      float64
			err                                                 error
			tm                                                  *term.Terminal
			dr                                                  term.Drawer
			conn                                                wm.Connection
			ximgBounds, imgPosBounds, areaOfInterest            image.Rectangle
			ximgCroppedSize                                     image.Point
			ximg, ximgResized, imgRet                           image.Image
			ximgCropped                                         *image.NRGBA
		)
		if termProvider == nil || termChecker == nil || drawFunc == nil || rsz == nil || imgChan == nil {
			errRet = errors.NilParam()
			goto end
		}

		tm, err = termProvider(pty)
		if err != nil {
			errRet = err
			goto end
		}
		if tm == nil {
			errRet = errors.New(`unable to find terminal`)
			goto end
		}
		defer tm.Close()

		conn, err = wm.NewConn(tm)
		if err != nil {
			errRet = err
			goto end
		}
		defer conn.Close()

		// TODO consider window maximization for better cell size values
		cpw, cph, err = tm.CellSize()
		if err != nil {
			errRet = err
			goto end
		}
		tpw, tph, err = tm.SizeInPixels()
		if err != nil {
			errRet = err
			goto end
		}

		if len(drawerName) == 0 {
			if drs := tm.Drawers(); len(drs) > 0 {
				dr = tm.Drawers()[0]
			}
		} else {
			dr = term.GetRegDrawerByName(drawerName)
		}
		if dr == nil {
			errRet = errors.New(`nil drawer`)
			goto end
		}

		areaOfInterest, scaleX, scaleY, err = drawFunc(tm, dr, rsz, uint(cpw), uint(cph))
		if err != nil {
			errRet = err
			goto end
		}

		// TODO
		ximg, err = tm.Window().Screenshot()
		if err != nil {
			errRet = err
			goto end
		}
		ximgBounds = ximg.Bounds()

		// assumption: (hopefully no scrollbars)
		// ┌──────────────────┐
		// │ File ... About   │
		// │ Tab1 Tab2 ...    │
		// ├──────────────────┤
		// │ $                │
		// │                  │
		// │                  │
		// │                  │
		// └──────────────────┘
		//
		// assume there are no scrollbars and that the bottom edge
		// has the same thickness as the left and right edge
		edgeThickness = uint((ximgBounds.Dx() - int(tpw)) / 2)
		menuBarHeight = uint(ximgBounds.Dy() - int(tph) - int(edgeThickness))
		extraBorder = 1
		imgPosBounds = image.Rect(
			int(cpw)*(areaOfInterest.Min.X-int(extraBorder)), int(cph)*(areaOfInterest.Min.Y-int(extraBorder)),
			int(cpw)*(areaOfInterest.Max.X+int(extraBorder)), int(cph)*(areaOfInterest.Max.Y+int(extraBorder)),
		).
			Add(image.Pt(int(edgeThickness), int(menuBarHeight)))
		ximgCropped = imagingOrig.Crop(ximg, imgPosBounds)
		if ximgCropped == nil {
			errRet = errors.New(consts.ErrNilImage)
			goto end
		}
		ximgCroppedSize = ximgCropped.Bounds().Size()
		ximgCroppedSize.X = int(float64(ximgCroppedSize.X) * scaleX)
		ximgCroppedSize.Y = int(float64(ximgCroppedSize.Y) * scaleY)
		rsz = &rdefault.Resizer{}
		ximgResized, err = rsz.Resize(ximgCropped, ximgCroppedSize)
		if err != nil {
			errRet = err
			goto end
		}
		imgRet = ximgResized

	end:
		if errRet != nil {
			close(imgChan)
			return
		}
		imgChan <- &imageWithMetadata{termName: tm.Name(), drawerName: dr.Name(), Image: imgRet}
		return
	}
}
