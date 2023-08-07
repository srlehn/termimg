package iterm2

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/go-errors/errors"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/mux"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerITerm2{}) }

var _ term.Drawer = (*drawerITerm2)(nil)

type drawerITerm2 struct{}

func (d *drawerITerm2) Name() string     { return `iterm2` }
func (d *drawerITerm2) New() term.Drawer { return &drawerITerm2{} }

func (d *drawerITerm2) IsApplicable(inp term.DrawerCheckerInput) bool {
	if inp == nil {
		return false
	}
	switch inp.Name() {
	case `iterm2`,
		`macterm`,
		// `wayst`, // untested
		`wezterm`:
		return true
	default:
		return false
	}
}

func (d *drawerITerm2) Draw(img image.Image, bounds image.Rectangle, rsz term.Resizer, tm *term.Terminal) error {
	if d == nil || tm == nil || img == nil {
		return errors.New(`nil parameter`)
	}
	timg, ok := img.(*term.Image)
	if !ok {
		timg = term.NewImage(img)
	}
	if timg == nil {
		return errors.New(internal.ErrNilImage)
	}

	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return err
	}

	tcw, tch, err := tm.SizeInCells()
	if err != nil {
		fmt.Println(err)
		return err
	}
	if tcw == 0 || tch == 0 {
		return errors.New("could not query terminal dimensions")
	}

	buf := new(bytes.Buffer)
	if tm.Name() == `wezterm` {
		// error for png image:
		// ERROR  wezterm_gui::glyphcache     > Error decoding image: inconsistent 600x450 -> 810000
		if err = jpeg.Encode(buf, timg.Fitted, &jpeg.Options{Quality: 100}); err != nil {
			return err
		}
	} else {
		if err = png.Encode(buf, timg.Fitted); err != nil {
			return err
		}
	}
	imgBytes := buf.Bytes()
	imgSize := len(imgBytes)
	imgBase64 := base64.StdEncoding.EncodeToString(imgBytes)
	// imgls breaks the base64 encoded file after 76 chars ending with a \n before \a
	breakLine := false
	if breakLine {
		var imgBase64NL string
		lineLen := 76
		imgBase64Len := len(imgBase64)
		for i := 0; i < imgBase64Len-lineLen; i += lineLen {
			imgBase64NL += imgBase64[i:i+lineLen] + "\n"
		}

		if rest := imgBase64Len % lineLen; rest != 0 {
			imgBase64NL += imgBase64[imgBase64Len-rest:] + "\n"
		}
		imgBase64 = imgBase64NL
	}
	imageTitle := timg.FileName
	/*if len(imageTitle) == 0 {
		imageTitle = `image.png`
	}*/
	nameBase64 := base64.StdEncoding.EncodeToString([]byte(imageTitle))
	// nameBase64 := base64.URLEncoding.EncodeToString([]byte(imageTitle))

	// imgls uses "true" instead
	// 0 for stretching - 1 for no stretching
	preserveAspectRatio := 0

	// protocol extension (wezterm)
	// requires wezterm 20220319-142410-0fcdea07
	var keepPosWezTerm string
	if tm.Name() == `wezterm` {
		// https://wezfurlong.org/wezterm/imgcat.html
		keepPosWezTerm = `;doNotMoveCursor=1`
	}
	iterm2String := mux.Wrap( // TODO check if wrap is necessary
		fmt.Sprintf(
			"\033]1337;File=name=%s;inline=1;width=%d;height=%d;size=%d;preserveAspectRatio=%d%s:%s\a",
			nameBase64,
			bounds.Dx(), bounds.Dy(),
			imgSize,
			preserveAspectRatio,
			keepPosWezTerm,
			imgBase64,
		),
		tm,
	)
	// for width, height:   "auto"   ||   N: N character cells   ||   Npx: N pixels   ||   N%: N percent of terminal width/height
	iterm2String = fmt.Sprintf("\033[%d;%dH%s", bounds.Min.Y+1, bounds.Min.X+1, iterm2String)
	timg.SetInband(bounds, iterm2String, d, tm)

	tm.Printf(`%s`, iterm2String)

	return nil
}

// https://iterm2.com/documentation-images.html
//
// imgcat:
// Empty string or file type like "application/json" or ".js"
// "type=..." "
