package term

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
)

type ImageEncoder = internal.ImageEncoder

// Image ...
type Image struct {
	Original     image.Image
	Resized      image.Image
	Cropped      image.Image
	FileName     string          // lazily loaded
	Encoded      []byte          // lazily loaded
	pos          image.Rectangle // image size in cells at resize time, position for cropping
	termSize     image.Point     // terminal size in cells at crop time
	inBandMu     sync.RWMutex
	inBand       map[string]inBandString
	posObjsMu    sync.RWMutex
	posObjs      map[string]posObject
	drawerSpecMu sync.RWMutex
	drawerSpec   map[string]any
	internal.Closer
}

// NewImage ...
func NewImage(img image.Image) *Image {
	if img != nil {
		if m, ok := img.(*Image); ok {
			if m.inBand == nil {
				m.inBand = make(map[string]inBandString)
			}
			if m.posObjs == nil {
				m.posObjs = make(map[string]posObject)
			}
			if m.drawerSpec == nil {
				m.drawerSpec = make(map[string]any)
			}
			if m.Closer == nil {
				m.Closer = internal.NewCloser()
			}
			return m
		}
	}
	return &Image{
		Original:   img,
		inBand:     make(map[string]inBandString),
		posObjs:    make(map[string]posObject),
		drawerSpec: make(map[string]any),
		Closer:     internal.NewCloser(),
	}
}

// NewImageFilename - for lazy loading the file
func NewImageFilename(imgFile string) *Image {
	if imgFilenameAbs, err := filepath.Abs(imgFile); err == nil {
		imgFile = imgFilenameAbs
	}
	return &Image{
		FileName:   imgFile,
		inBand:     make(map[string]inBandString),
		posObjs:    make(map[string]posObject),
		drawerSpec: make(map[string]any),
		Closer:     internal.NewCloser(),
	}
}

// NewImageBytes - for lazy loading the file
func NewImageBytes(imgBytes []byte) *Image {
	return &Image{
		Encoded:    imgBytes,
		inBand:     make(map[string]inBandString),
		posObjs:    make(map[string]posObject),
		drawerSpec: make(map[string]any),
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
		return errors.NilReceiver()
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
			return errors.New(err)
		}
		defer f.Close()
		rdr = f
	}
	// TODO check MIME-Type to recognize animations
	// animated gifs store a delay time
	image, _, err := image.Decode(rdr)
	if err != nil {
		return errors.New(err)
	}
	i.Original = image
	return nil
}

// ColorModel ...
func (i *Image) ColorModel() color.Model {
	if i == nil {
		return color.RGBAModel
	}
	if err := i.Decode(); err != nil {
		// TODO log error
		return color.RGBAModel
	}
	return i.Original.ColorModel()
}

// Bounds ...
func (i *Image) Bounds() image.Rectangle {
	if i == nil {
		return image.Rectangle{}
	}
	if err := i.Decode(); err != nil {
		// TODO log error
		return image.Rectangle{}
	}
	return i.Original.Bounds()
}

// At ...
func (i *Image) At(x, y int) color.Color {
	if i == nil {
		// TODO log error
		return color.RGBA{}
	}
	if err := i.Decode(); err != nil {
		// TODO log error
		return color.RGBA{}
	}
	return i.Original.At(x, y)
}

// Image ...
func (i *Image) Image() (image.Image, error) {
	if i == nil {
		return nil, errors.NilReceiver()
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

// Inband ...
func (i *Image) Inband(placementCells image.Rectangle, d Drawer, t *Terminal) (string, error) {
	if i == nil {
		return ``, errors.NilReceiver()
	}
	if d == nil || t == nil {
		return ``, errors.New(`nil parameter`)
	}
	if placementCells.Dx() == 0 || placementCells.Dy() == 0 {
		return ``, errors.New(`no draw area`)
	}
	if i.inBand == nil {
		i.inBand = make(map[string]inBandString)
		return ``, errors.New(`struct field is nil`)
	}
	k := d.Name() + `_` + t.Name()
	i.inBandMu.RLock()
	defer i.inBandMu.RUnlock()
	v, ok := i.inBand[k]
	if !ok {
		return ``, errors.New(`no entry`)
	}
	if v.placementCells != placementCells {
		return ``, errors.New(`different image placement`)
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

	return ``, errors.New(`no find`)
}

// SetInband ...
func (i *Image) SetInband(placementCells image.Rectangle, inband string, d Drawer, t *Terminal) error {
	// TODO save objects for a restricted count of past placements
	if i == nil {
		return errors.NilReceiver()
	}
	if d == nil || t == nil {
		return errors.New(`nil parameter`)
	}
	if placementCells.Dx() == 0 || placementCells.Dy() == 0 {
		return errors.New(`no draw area`)
	}
	i.inBandMu.Lock()
	defer i.inBandMu.Unlock()
	if i.inBand == nil {
		i.inBand = make(map[string]inBandString)
		return errors.New(`struct field is nil`)
	}
	k := d.Name() + `_` + t.Name()
	v, ok := i.inBand[k]
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

	i.inBand[k] = v

	return nil
}

type posObject struct {
	placementCells        image.Rectangle
	termSizeInCellsWidth  uint
	termSizeInCellsHeight uint
	full                  any
	cropped               any
}

// PosObject ...
func (i *Image) PosObject(placementCells image.Rectangle, d Drawer, t *Terminal) (any, error) {
	if i == nil {
		return ``, errors.NilReceiver()
	}
	if d == nil || t == nil {
		return ``, errors.New(`nil parameter`)
	}
	if placementCells.Dx() == 0 || placementCells.Dy() == 0 {
		return ``, errors.New(`no draw area`)
	}
	i.posObjsMu.RLock()
	defer i.posObjsMu.RUnlock()
	if i.posObjs == nil {
		i.posObjs = make(map[string]posObject)
		return nil, errors.New(`struct field is nil`)
	}
	k := d.Name() + `_` + t.Name()
	v, ok := i.posObjs[k]
	if !ok {
		return ``, errors.New(`no entry`)
	}
	if v.placementCells != placementCells {
		return ``, errors.New(`different image placement`)
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

	return ``, errors.New(`no find`)
}

// SetPosObject ...
func (i *Image) SetPosObject(placementCells image.Rectangle, obj any, d Drawer, t *Terminal) error {
	// TODO save objects for a restricted count of past placements
	if i == nil {
		return errors.NilReceiver()
	}
	if d == nil || t == nil {
		return errors.New(`nil parameter`)
	}
	if placementCells.Dx() == 0 || placementCells.Dy() == 0 {
		return errors.New(`no draw area`)
	}
	i.posObjsMu.Lock()
	defer i.posObjsMu.Unlock()
	if i.posObjs == nil {
		i.posObjs = make(map[string]posObject)
		return errors.New(`struct field is nil`)
	}
	k := d.Name() + `_` + t.Name()
	v, ok := i.posObjs[k]
	if !ok {
		v = posObject{
			placementCells: placementCells,
		}
	}

	tcw, tch, err := t.SizeInCells()
	if err != nil {
		return err
	}
	if p := v.placementCells.Max; uint(p.X) <= tcw && uint(p.Y) <= tch {
		v.full = obj
	} else {
		v.termSizeInCellsWidth = tcw
		v.termSizeInCellsHeight = tch
		v.cropped = obj
	}

	i.posObjs[k] = v

	return nil
}

// DrawerObject ...
func (i *Image) DrawerObject(d Drawer) (any, error) {
	if i == nil || d == nil {
		return nil, errors.New(`nil receiver or parameter`)
	}
	i.drawerSpecMu.RLock()
	defer i.drawerSpecMu.RUnlock()
	if i.drawerSpec == nil {
		i.drawerSpec = make(map[string]any)
		return nil, errors.New(`object not found`)
	}
	obj, exists := i.drawerSpec[d.Name()]
	if !exists {
		return nil, errors.New(`object not found`)
	}
	return obj, nil
}

// SetDrawerObject ...
func (i *Image) SetDrawerObject(obj any, d Drawer) error {
	if i == nil || d == nil {
		return errors.New(`nil receiver or parameter`)
	}
	i.drawerSpecMu.Lock()
	defer i.drawerSpecMu.Unlock()
	if i.drawerSpec == nil {
		i.drawerSpec = make(map[string]any)
	}
	i.drawerSpec[d.Name()] = obj
	return nil
}

// SaveAsFile writes the image to a temporary file.
// Defer Image.Close() or call rm() when no longer needed.
func (i *Image) SaveAsFile(t *Terminal, fileExt string, enc ImageEncoder) (rm func() error, err error) {
	// TODO allow creation of other file types
	if i == nil {
		return nil, errors.NilReceiver()
	}
	if len(i.FileName) > 0 {
		return nil, nil
	}

	if err := i.Decode(); err != nil {
		return nil, err
	}
	fileExt = strings.TrimPrefix(fileExt, `.`)
	if err != nil {
		return nil, errors.New(err)
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
		return errors.NilReceiver()
	}
	if sv == nil {
		i.Cropped = nil
		i.termSize = image.Point{}
		return errors.NilParam()
	}
	w := bounds.Dx()
	h := bounds.Dy()
	if bounds == (image.Rectangle{}) || w <= 0 || h <= 0 {
		i.Cropped = nil
		i.termSize = image.Point{}
		return errors.New(`resize: nil size`)
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
			i.Cropped = nil
			i.termSize = image.Point{}
			return err
		}
		if err := i.Decode(); err != nil {
			i.Cropped = nil
			i.termSize = image.Point{}
			return err
		}
		if i.Original == nil {
			i.Cropped = nil
			i.termSize = image.Point{}
			err := errors.New(consts.ErrNilImage)
			return err
		}
		size := image.Point{X: w * int(cpw), Y: h * int(cph)}
		imgResized, err := rsz.Resize(i.Original, size)
		if err != nil {
			i.Cropped = nil
			i.termSize = image.Point{}
			return err
		}
		if imgResized == nil {
			i.Cropped = nil
			i.termSize = image.Point{}
			return errors.New(consts.ErrNilImage)
		}
		if imgResized.Bounds() == (image.Rectangle{}) {
			i.Cropped = nil
			i.termSize = image.Point{}
			return errors.New(`resize: nil size`)
		}
		i.Resized = imgResized
		i.pos = bounds
	}

crop:
	tcw, tch, err := sv.SizeInCells()
	if err != nil {
		i.Cropped = nil
		i.termSize = image.Point{}
		return err
	}
	if int(tcw) <= i.pos.Min.X || int(tch) <= i.pos.Min.Y {
		i.Cropped = nil
		i.termSize = image.Point{}
		return errors.New(`image outside visible area`)
	}
	if int(tcw) >= (i.pos.Max.X) && int(tch) >= (i.pos.Max.Y) {
		// image fully visible
		i.Cropped = i.Resized
		i.termSize = image.Point{int(tcw), int(tch)}
		return nil
	}

	// cropping needed
	if cpw < 0 || cph < 0 {
		cpw, cph, err = sv.CellSize()
		if err != nil {
			i.Cropped = nil
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
		i.Cropped = nil
		i.termSize = image.Point{}
		return errors.New(consts.ErrNilImage)
	}
	i.Cropped = imgCropped
	i.termSize = image.Point{int(tcw), int(tch)}

	return nil
}

// crops instead of downscales, no upscale
type resizerFallback struct{}

func ResizerDefault() Resizer { return &resizerFallback{} }

func (r *resizerFallback) Resize(img image.Image, size image.Point) (image.Image, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, errors.New("invalid size")
	}

	srcB := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))

	for y := 0; y < size.Y; y++ {
		srcY := srcB.Min.Y + y*srcB.Dy()/size.Y
		for x := 0; x < size.X; x++ {
			srcX := srcB.Min.X + x*srcB.Dx()/size.X
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}

	return dst, nil
}

func cropImage(img image.Image, size image.Point) (image.Image, error) {
	if img == nil {
		return nil, errors.New(consts.ErrNilImage)
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
