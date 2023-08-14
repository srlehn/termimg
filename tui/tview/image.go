//go:build dev

package tviewimg

import (
	"image"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// based on: github.com/rivo/tview/image.go (MIT License)

// TODO implement

// Image implements a widget that displays one image. The original image
// (specified with [Image.SetImage]) is resized according to the specified size
// (see [Image.SetSize]), using the specified number of colors (see
// [Image.SetColors]), while applying dithering if necessary (see
// [Image.SetDithering]).
//
// Images are approximated by graphical characters in the terminal. The
// resolution is therefore limited by the number and type of characters that can
// be drawn in the terminal and the colors available in the terminal. The
// quality of the final image also depends on the terminal's font and spacing
// settings, none of which are under the control of this package. Results may
// vary.
type Image struct {
	*tview.Box

	// The image to be displayed. If nil, the widget will be empty.
	image image.Image

	// The size of the image. If a value is 0, the corresponding size is chosen
	// automatically based on the other size while preserving the image's aspect
	// ratio. If both are 0, the image uses as much space as possible. A
	// negative value represents a percentage, e.g. -50 means 50% of the
	// available space.
	width, height int

	// The number of colors to use. If 0, the number of colors is chosen based
	// on the terminal's capabilities.
	colors int

	// The dithering algorithm to use, one of the constants starting with
	// "ImageDithering".
	dithering int

	// The width of a terminal's cell divided by its height.
	aspectRatio float64

	// Horizontal and vertical alignment, one of the "Align" constants.
	alignHorizontal, alignVertical int

	// The text to be displayed before the image.
	label string

	// The label style.
	labelStyle tcell.Style

	// The screen width of the label area. A value of 0 means use the width of
	// the label text.
	labelWidth int

	// The actual image size (in cells) when it was drawn the last time.
	lastWidth, lastHeight int

	// A callback function set by the Form class and called when the user leaves
	// this form item.
	finished func(tcell.Key)
}

// NewImage returns a new image widget with an empty image (use [Image.SetImage]
// to specify the image to be displayed). The image will use the widget's entire
// available space. The dithering algorithm is set to Floyd-Steinberg dithering.
// The terminal's cell aspect ratio defaults to 0.5.
func NewImage() *Image {
	return &Image{
		Box:         tview.NewBox(),
		aspectRatio: 0.5,
	}
}

// SetImage sets the image to be displayed. If nil, the widget will be empty.
func (i *Image) SetImage(image image.Image) *Image {
	i.image = image
	i.lastWidth, i.lastHeight = 0, 0
	return i
}

// SetSize sets the size of the image. Positive values refer to cells in the
// terminal. Negative values refer to a percentage of the available space (e.g.
// -50 means 50%). A value of 0 means that the corresponding size is chosen
// automatically based on the other size while preserving the image's aspect
// ratio. If both are 0, the image uses as much space as possible while still
// preserving the aspect ratio.
func (i *Image) SetSize(rows, columns int) *Image {
	i.width = columns
	i.height = rows
	return i
}

// GetFieldWidth returns this primitive's field width. This is the image's width
// or, if the width is 0 or less, the proportional width of the image based on
// its height as returned by [Image.GetFieldHeight]. If there is no image, 0 is
// returned.
func (i *Image) GetFieldWidth() int {
	if i.width <= 0 {
		if i.image == nil {
			return 0
		}
		bounds := i.image.Bounds()
		height := i.GetFieldHeight()
		return bounds.Dx() * height / bounds.Dy()
	}
	return i.width
}

// GetFieldHeight returns this primitive's field height. This is the image's
// height or 8 if the height is 0 or less.
func (i *Image) GetFieldHeight() int {
	if i.height <= 0 {
		return 8
	}
	return i.height
}

func (i *Image) GetLabel() string { return `` }

// SetFormAttributes sets a number of item attributes at once.
func (i *Image) SetFormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) tview.FormItem {
	return i
}

// SetDisabled sets whether or not the item is disabled / read-only.
func (i *Image) SetDisabled(disabled bool) tview.FormItem {
	return i // Images are always read-only.
}

// SetFinishedFunc sets a callback invoked when the user leaves this form item.
func (i *Image) SetFinishedFunc(handler func(key tcell.Key)) tview.FormItem {
	i.finished = handler
	return i
}

// Focus is called when this primitive receives focus.
func (i *Image) Focus(delegate func(p tview.Primitive)) {
	// If we're part of a form, there's nothing the user can do here so we're
	// finished.
	if i.finished != nil {
		i.finished(-1)
		return
	}

	i.Box.Focus(delegate)
}

// Draw draws this primitive onto the screen.
func (i *Image) Draw(screen tcell.Screen) {
	i.DrawForSubclass(screen, i)

	// Regenerate image if necessary.
	// i.render()
}
