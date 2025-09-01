package iterm2

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log/slog"
	"strconv"
	"time"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/mux"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerITerm2{}) }

var _ term.Drawer = (*drawerITerm2)(nil)

type drawerITerm2 struct{}

func (d *drawerITerm2) Name() string     { return `iterm2` }
func (d *drawerITerm2) New() term.Drawer { return &drawerITerm2{} }

func (d *drawerITerm2) IsApplicable(inp term.DrawerCheckerInput) (bool, term.Properties) {
	if inp == nil {
		return false, nil
	}
	switch inp.Name() {
	case `iterm2`,
		`mintty`,
		`macterm`,
		// `wayst`, // untested
		`wezterm`:
		return true, nil
	case `konsole`:
		// https://konsole.kde.org/changelogs/22.04.html
		// https://invent.kde.org/utilities/konsole/-/merge_requests/594
		verMajStr, okMaj := inp.Property(propkeys.KonsoleVersionMajorXTVersion)
		verMinStr, okMin := inp.Property(propkeys.KonsoleVersionMinorXTVersion)
		if !okMaj || !okMin {
			return false, nil
		}
		verMaj, err := strconv.ParseUint(verMajStr, 10, 64)
		if err != nil || verMaj < 22 {
			return false, nil
		}
		if verMaj >= 23 {
			return true, nil
		}
		verMin, err := strconv.ParseUint(verMinStr, 10, 64)
		return err == nil && verMin >= 4, nil
	case `vscode`:
		// VSCode sets image support as a whole together with dimension reporting
		sixelCapable, isSixelCapable := inp.Property(propkeys.SixelCapable)
		if !isSixelCapable || sixelCapable != `true` {
			return false, nil
		}
		// VSCode v1.80.0 required
		// https://code.visualstudio.com/updates/v1_80#_image-supportm
		verMajStr, okMaj := inp.Property(propkeys.VSCodeVersionMajor)
		verMinStr, okMin := inp.Property(propkeys.VSCodeVersionMinor)
		if !okMaj || !okMin {
			return false, nil
		}
		verMaj, err := strconv.ParseUint(verMajStr, 10, 64)
		if err != nil || verMaj < 1 {
			return false, nil
		}
		if verMaj >= 2 {
			return true, nil
		}
		verMin, err := strconv.ParseUint(verMinStr, 10, 64)
		return err == nil && verMin >= 79, nil
	case `hyper`:
		// Hyper v4.0.0-canary.4 required
		// https://github.com/vercel/hyper/releases/tag/v4.0.0-canary.4
		verMajStr, okMaj := inp.Property(propkeys.HyperVersionMajor)
		verMinStr, okMin := inp.Property(propkeys.HyperVersionMinor)
		if !okMaj || !okMin {
			return false, nil
		}
		verMaj, err := strconv.ParseUint(verMajStr, 10, 64)
		if err != nil || verMaj < 4 {
			return false, nil
		}
		if verMaj >= 5 {
			return true, nil
		}
		verMin, err := strconv.ParseUint(verMinStr, 10, 64)
		if err != nil {
			return false, nil
		}
		if verMin >= 1 {
			return true, nil
		}
		_, okPtc := inp.Property(propkeys.HyperVersionPatch)
		verCnrStr, okCnr := inp.Property(propkeys.HyperVersionCanary)
		if okPtc && !okCnr {
			return true, nil
		}
		verCnr, err := strconv.ParseUint(verCnrStr, 10, 64)
		return err == nil && verCnr >= 4, nil
	case `laterminal`:
		return true, nil // TODO
	default:
		return false, nil
	}
}

func (d *drawerITerm2) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *drawerITerm2) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
	if d == nil || tm == nil || img == nil || ctx == nil {
		return nil, errors.New(`nil parameter`)
	}
	start := time.Now()
	timg, ok := img.(*term.Image)
	if !ok {
		timg = term.NewImage(img)
	}
	if timg == nil {
		return nil, errors.New(consts.ErrNilImage)
	}

	rsz := tm.Resizer()
	if rsz == nil {
		return nil, errors.New(`nil resizer`)
	}
	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return nil, err
	}

	tcw, tch, err := tm.SizeInCells()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	if tcw == 0 || tch == 0 {
		return nil, errors.New("could not query terminal dimensions")
	}

	buf := new(bytes.Buffer)
	if tm.Name() == `wezterm` {
		// error for png image:
		// ERROR  wezterm_gui::glyphcache     > Error decoding image: inconsistent 600x450 -> 810000
		if err = jpeg.Encode(buf, timg.Cropped, &jpeg.Options{Quality: 100}); err != nil {
			return nil, err
		}
	} else {
		if err = png.Encode(buf, timg.Cropped); err != nil {
			return nil, err
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

	logx.Debug(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		_, err := tm.Printf(`%s`, iterm2String)
		return logx.Err(err, tm, slog.LevelInfo)
	}

	return drawFn, nil
}

// https://iterm2.com/documentation-images.html
//
// imgcat:
// Empty string or file type like "application/json" or ".js"
// "type=..." "
