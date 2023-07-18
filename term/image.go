package term

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"io"
	"os"
	"strings"

	errorsGo "github.com/go-errors/errors"

	"github.com/srlehn/termimg/internal"
)

type ImageEncoder = internal.ImageEncoder

// Image ...
type Image struct {
	Original   image.Image
	Resized    image.Image
	Fitted     image.Image
	InBand     map[string]inBandString
	FileName   string          // lazily loaded
	Encoded    []byte          // lazily loaded
	pos        image.Rectangle // image size in cells at resize time, position for cropping
	termSize   image.Point     // terminal size in cells at crop time
	DrawerSpec map[string]any
	internal.Closer
}

// NewImage ...
func NewImage(img image.Image) *Image {
	if img != nil {
		if m, ok := img.(*Image); ok {
			if m.InBand == nil {
				m.InBand = make(map[string]inBandString)
			}
			return m
		}
	}
	return &Image{
		Original:   img,
		InBand:     make(map[string]inBandString),
		DrawerSpec: make(map[string]any),
		Closer:     internal.NewCloser(),
	}
}

// NewImageFileName - for lazy loading the file
func NewImageFileName(imgFile string) *Image {
	return &Image{
		FileName:   imgFile,
		InBand:     make(map[string]inBandString),
		DrawerSpec: make(map[string]any),
		Closer:     internal.NewCloser(),
	}
}

// NewImageBytes - for lazy loading the file
func NewImageBytes(imgBytes []byte) *Image {
	return &Image{
		Encoded:    imgBytes,
		InBand:     make(map[string]inBandString),
		DrawerSpec: make(map[string]any),
		Closer:     internal.NewCloser(),
	}
}

// Decode decodes and stores the image file in the struct.
// this is not required for some drawers where the
// file path is passed to the terminal.
//
// Decode requires registration of image decoders.
func (i *Image) Decode() error {
	if i == nil {
		return errorsGo.New(internal.ErrNilReceiver)
	}
	if i.Original != nil {
		return nil
	}
	var rdr io.Reader
	if len(i.Encoded) > 0 {
		if len(i.FileName) > 0 {
			return errors.New(`image contains 2 sources`)
		}
		rdr = bytes.NewReader(i.Encoded)
	} else if len(i.FileName) > 0 {
		f, err := os.Open(i.FileName)
		if err != nil {
			return err
		}
		defer f.Close()
		rdr = f
	}
	image, _, err := image.Decode(rdr)
	if err != nil {
		return err
	}
	i.Original = image
	return nil
}

// ColorModel ...
func (i *Image) ColorModel() color.Model {
	if i == nil {
		panic(errorsGo.New(internal.ErrNilReceiver))
	}
	if err := i.Decode(); err != nil {
		panic(err)
	}
	return i.Original.ColorModel()
}

// Bounds ...
func (i *Image) Bounds() image.Rectangle {
	if i == nil {
		panic(errorsGo.New(internal.ErrNilReceiver))
	}
	if err := i.Decode(); err != nil {
		panic(err)
	}
	return i.Original.Bounds()
}

// At ...
func (i *Image) At(x, y int) color.Color {
	if i == nil {
		panic(errorsGo.New(internal.ErrNilReceiver))
	}
	if err := i.Decode(); err != nil {
		panic(err)
	}
	return i.Original.At(x, y)
}

// Image ...
func (i *Image) Image() (image.Image, error) {
	if i == nil {
		return nil, errorsGo.New(internal.ErrNilReceiver)
	}
	if err := i.Decode(); err != nil {
		return nil, err
	}
	return i.Original, nil
}

// inBandString stores inband strings like ANSI escape sequences for reuse
type inBandString struct {
	placementCells        image.Rectangle
	termSizeInCellsWidth  uint
	termSizeInCellsHeight uint
	full                  string
	cropped               string
}

// GetInband ...
func (i *Image) GetInband(placementCells image.Rectangle, d Drawer, t *Terminal) (string, error) {
	if i == nil {
		return ``, errorsGo.New(internal.ErrNilReceiver)
	}
	if d == nil || t == nil {
		return ``, errorsGo.New(`nil parameter`)
	}
	if placementCells.Dx() == 0 || placementCells.Dy() == 0 {
		return ``, errorsGo.New(`no draw area`)
	}
	if i.InBand == nil {
		i.InBand = make(map[string]inBandString)
		return ``, errorsGo.New(internal.ErrNilReceiver)
	}
	k := d.Name() + `_` + t.Name()
	v, ok := i.InBand[k]
	if !ok {
		return ``, errorsGo.New(`no entry`)
	}
	// fmt.Println(v.placementCells, placementCells) // TODO rm
	// fmt.Printf("%+#v\n", v) // TODO rm
	if v.placementCells != placementCells {
		return ``, errorsGo.New(`different image placement`)
	}

	tcw, tch, err := t.SizeInCells()
	if err != nil {
		return ``, err
	}
	tpw, tph, err := t.SizeInPixels()
	if err != nil {
		return ``, err
	}
	cpw, cph, err := t.CellSize()
	if err != nil {
		return ``, err
	}
	if p := v.placementCells.Max; uint(float64(p.X)*cpw) <= tpw && uint(float64(p.Y)*cph) <= tph {
		return v.full, nil
	} else if v.termSizeInCellsWidth == tcw && v.termSizeInCellsHeight == tch {
		return v.cropped, nil
	}

	return ``, errorsGo.New(`no find`)
}

// SetInband ...
func (i *Image) SetInband(placementCells image.Rectangle, inband string, d Drawer, t *Terminal) error {
	if i == nil {
		return errorsGo.New(internal.ErrNilReceiver)
	}
	if d == nil || t == nil {
		return errorsGo.New(`nil parameter`)
	}
	if placementCells.Dx() == 0 || placementCells.Dy() == 0 {
		return errorsGo.New(`no draw area`)
	}
	if i.InBand == nil {
		i.InBand = make(map[string]inBandString)
		return errorsGo.New(internal.ErrNilReceiver)
	}
	k := d.Name() + `_` + t.Name()
	v, ok := i.InBand[k]
	if !ok {
		v = inBandString{
			placementCells: placementCells,
		}
	}

	tcw, tch, err := t.SizeInCells()
	if err != nil {
		return err
	}
	if p := v.placementCells.Max; uint(p.X) <= tcw && uint(p.Y) <= tch {
		v.full = inband
	} else {
		v.termSizeInCellsWidth = tcw
		v.termSizeInCellsHeight = tch
		v.cropped = inband
	}

	i.InBand[k] = v

	return nil
}

// SaveAsFile writes the image to a temporary file.
// Defer Image.Close() or call rm() when no longer needed.
func (i *Image) SaveAsFile(t *Terminal, fileExt string, enc ImageEncoder) (rm func() error, err error) {
	// TODO allow creation of other file types
	if i == nil {
		return nil, errorsGo.New(internal.ErrNilReceiver)
	}
	if len(i.FileName) > 0 {
		return nil, nil
	}

	if err := i.Decode(); err != nil {
		return nil, err
	}
	fileExt = strings.TrimPrefix(fileExt, `.`)
	if err != nil {
		return nil, errorsGo.New(err)
	}
	f, err := t.CreateTemp(`*.` + fileExt)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := enc.Encode(f, i.Original, fileExt); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	fileName := f.Name()
	i.FileName = fileName
	rm = func() error { i.FileName = ``; return os.Remove(fileName) }
	i.OnClose(rm)

	return rm, nil
}

////////////////////////////////////////////////////////////////////////////////

// Resizer resizes images
type Resizer interface {
	Resize(img image.Image, size image.Point) (image.Image, error)
}

// nil Resizer is allowed (default resizer crops instead of resize)
func (i *Image) Fit(bounds image.Rectangle, rsz Resizer, sv Surveyor) error {
	if i == nil {
		return errorsGo.New(internal.ErrNilReceiver)
	}
	if sv == nil {
		i.Fitted = nil
		i.termSize = image.Point{}
		return errorsGo.New(internal.ErrNilParam)
	}
	w := bounds.Dx()
	h := bounds.Dy()
	if bounds == (image.Rectangle{}) || w <= 0 || h <= 0 {
		i.Fitted = nil
		i.termSize = image.Point{}
		return errorsGo.New(`resize: nil size`)
	}
	var cpw, cph float64
	if w == i.pos.Dx() && h == i.pos.Dy() {
		goto crop
	}

	// resizing needed
	{
		if rsz == nil {
			rsz = &resizerFallback{} // TODO resize instead of just crop...
		}
		var err error
		cpw, cph, err = sv.CellSize()
		if err != nil {
			i.Fitted = nil
			i.termSize = image.Point{}
			return err
		}
		if err := i.Decode(); err != nil {
			i.Fitted = nil
			i.termSize = image.Point{}
			return err
		}
		if i.Original == nil {
			i.Fitted = nil
			i.termSize = image.Point{}
			err := errorsGo.New(internal.ErrNilImage)
			return err
		}
		size := image.Point{X: w * int(cpw), Y: h * int(cph)}
		imgResized, err := rsz.Resize(i.Original, size)
		if err != nil {
			i.Fitted = nil
			i.termSize = image.Point{}
			return err
		}
		if imgResized == nil {
			i.Fitted = nil
			i.termSize = image.Point{}
			return errorsGo.New(internal.ErrNilImage)
		}
		i.Resized = imgResized
		i.pos = bounds
	}

crop:
	tcw, tch, err := sv.SizeInCells()
	if err != nil {
		i.Fitted = nil
		i.termSize = image.Point{}
		return err
	}
	if int(tcw) <= i.pos.Min.X || int(tch) <= i.pos.Min.Y {
		i.Fitted = nil
		i.termSize = image.Point{}
		return errorsGo.New(`image outside visible area`)
	}
	if int(tcw) >= (i.pos.Max.X) && int(tch) >= (i.pos.Max.Y) {
		// image fully visible
		i.Fitted = i.Resized
		i.termSize = image.Point{int(tcw), int(tch)}
		return nil
	}

	// cropping needed
	if cpw < 0 || cph < 0 {
		cpw, cph, err = sv.CellSize()
		if err != nil {
			i.Fitted = nil
			i.termSize = image.Point{}
			return err
		}
	}
	sizeCropped := image.Point{
		X: int(float64(int(tcw)-i.pos.Min.X) * cpw),
		Y: int(float64(int(tch)-i.pos.Min.Y) * cph),
	}
	imgCropped, err := cropImage(i.Resized, sizeCropped)
	if err != nil {
		return err
	}
	if imgCropped == nil {
		i.Fitted = nil
		i.termSize = image.Point{}
		return errorsGo.New(internal.ErrNilImage)
	}
	i.Fitted = imgCropped
	i.termSize = image.Point{int(tcw), int(tch)}

	return nil
}

// crops instead of downscales, no upscale
type resizerFallback struct{}

func ResizerDefault() Resizer { return &resizerFallback{} }

func (r *resizerFallback) Resize(img image.Image, size image.Point) (image.Image, error) {
	return cropImage(img, size)
}

func cropImage(img image.Image, size image.Point) (image.Image, error) {
	if img == nil {
		return nil, errorsGo.New(internal.ErrNilImage)
	}
	b := img.Bounds()
	if simg, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		return simg.SubImage(image.Rectangle{Min: b.Min, Max: b.Min.Add(size)}), nil
	}
	m := image.NewNRGBA(image.Rectangle{Max: image.Point{X: size.X, Y: size.Y}})
	draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
	return m, nil
}
