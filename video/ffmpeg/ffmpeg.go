// unstable API

package ffmpeg

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

func ExtractFrames(ctx context.Context, vidFilename string, sizePixels image.Point, fps int) (pattern string, cleanUp func() error, _ error) {
	name := filepath.Base(vidFilename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	dir, err := os.MkdirTemp(``, consts.LibraryName+`_*`)
	if err != nil {
		return ``, nil, errors.New(err)
	}
	cleanUp = func() error { return os.RemoveAll(dir) }
	pattern = filepath.Join(dir, name) + `_%06d.png`

	cmd := exec.CommandContext(
		ctx,
		`ffmpeg`,
		`-i`, vidFilename,
		`-hide_banner`,
		`-loglevel`, `quiet`,
		`-s`, strconv.Itoa(sizePixels.X)+`x`+strconv.Itoa(sizePixels.Y),
		`-vf`, `fps=`+strconv.Itoa(fps),
		pattern,
	)
	if err := cmd.Run(); err != nil {
		_ = cleanUp()
		return ``, nil, errors.New(err)
	}

	return pattern, cleanUp, nil
}

func StreamFrames(ctx context.Context, vidFilename string, sizePixels image.Point, fps int) (<-chan image.Image, error) {
	cmd := exec.CommandContext(
		ctx,
		`ffmpeg`,
		`-i`, vidFilename,
		`-hide_banner`,
		`-loglevel`, `quiet`,
		`-s`, strconv.Itoa(sizePixels.X)+`x`+strconv.Itoa(sizePixels.Y),
		`-vf`, `fps=`+strconv.Itoa(fps),
		// `-r`, `1`,
		`-s`, strconv.Itoa(sizePixels.X)+`x`+strconv.Itoa(sizePixels.Y),
		`-f`, `image2pipe`,
		// `-vcodec`, `mjpeg`, // fails (change in ffmpeg v6.0)
		// `-vcodec`, `png`, // works sometimes
		`-c:v`, `png`, `-pix_fmt`, `rgb48be`,
		`pipe:`,
	)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	// cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	vid := make(chan image.Image)
	go func() {
		defer close(vid)
		defer pw.Close()
		defer cmd.Wait()
		i := 0
		for ; ; i++ {
			img, err := png.Decode(pr)
			if err == io.EOF {
				break
			}
			vid <- img
		}
	}()

	return vid, nil
}

func ImageStream(ctx context.Context, rsz term.Resizer, sizePixels image.Point, pattern string) (<-chan image.Image, error) {
	dir := filepath.Dir(pattern)
	ds, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	vid := make(chan image.Image)
	go func() {
		defer close(vid)
		for i := 1; i <= len(ds); i++ {
			img, err := rsz.Resize(term.NewImageFilename(fmt.Sprintf(pattern, i)), sizePixels)
			if err != nil {
				fmt.Println(err)
				if os.IsNotExist(err) {
					break
				}
			}
			vid <- img
		}
	}()
	return vid, nil
}

type Frame struct {
	ID int
	image.Image
	error
}

func (f *Frame) Err() error {
	if f == nil {
		return errors.New(consts.ErrNilReceiver)
	}
	return f.error
}
