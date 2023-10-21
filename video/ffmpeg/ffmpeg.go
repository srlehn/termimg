// unstable API

package ffmpeg

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

func ExtractFrames(ctx context.Context, vidFilename string, sizePixels image.Point, fps int) (pattern string, _ error) {
	name := filepath.Base(vidFilename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	dir, err := os.MkdirTemp(``, consts.LibraryName+`_*`)
	if err != nil {
		return ``, errors.New(err)
	}
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
		return ``, errors.New(err)
	}

	return pattern, nil
}

func ImageStream(ctx context.Context, c chan<- image.Image, rsz term.Resizer, sizePixels image.Point, pattern string) error {
	dir := filepath.Dir(pattern)
	ds, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	fmt.Println(pattern, dir, len(ds), sizePixels)
	for i := 1; i <= len(ds); i++ {
		img, err := rsz.Resize(term.NewImageFilename(fmt.Sprintf(pattern, i)), sizePixels)
		if err != nil {
			fmt.Println(err)
			break
		}
		c <- img
	}
	return nil
}
