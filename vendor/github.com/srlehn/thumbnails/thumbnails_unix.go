//go:build unix && !android && !ios

package thumbnails

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"

	pngstructure "github.com/dsoprea/go-png-image-structure/v2"
	"github.com/nfnt/resize"
)

const (
	software = `go-icon-setter`

	fileURLScheme = `file://`
)

var (
	ps     string = string(os.PathSeparator)
	scales        = []string{`xx-large`, `x-large`, `large`, `normal`}
)

func OpenThumbnail(filename string, size image.Point, runThumbnailer bool) (image.Image, error) {
	portable := false
	img, _, err := openThumbnail(filename, size, runThumbnailer, portable)
	return img, err
}

func openThumbnail(filename string, size image.Point, runThumbnailer, portable bool) (image.Image, string, error) {
	f, err := os.Stat(filename)
	if err != nil {
		return nil, ``, err
	}
	if (size.X > 0 && size.Y > 0) || (size.X < 1 && size.Y < 1) {
		return nil, ``, errors.New(`either size.X or size.Y has to be 0`)
	}
	mTime := fmt.Sprintf(`%d`, f.ModTime().Unix())

	md5Sum, fileURL := hashFilename(filename, portable)

	var thumbnailDirBase string
	if portable {
		thumbnailDirBase = filepath.Join(filepath.Dir(filename), `.sh_thumbnails`)
	} else {
		if xch := os.Getenv(`XDG_CACHE_HOME`); len(xch) > 0 {
			thumbnailDirBase = xch
		} else if home := os.Getenv(`HOME`); len(home) > 0 {
			thumbnailDirBase = filepath.Join(home, `.cache`)
		} else {
			return nil, ``, errors.New(`unable to determine cache directory`)
		}
		thumbnailDirBase = filepath.Join(thumbnailDirBase, `thumbnails`)
	}
	var scale string
	switch {
	case size.X >= 512 || size.Y >= 512:
		scale = `xx-large`
	case size.X >= 256 || size.Y >= 256:
		scale = `x-large`
	case size.X >= 128 || size.Y >= 128:
		scale = `large`
	default:
		scale = `normal`
	}

	sc := scale
	thumbnailDir := filepath.Join(thumbnailDirBase, sc)
	thumbnailFilename := filepath.Join(thumbnailDir, md5Sum+`.png`)
	var img image.Image
	if _, err := os.Stat(thumbnailFilename); os.IsNotExist(err) {
		// write thumbnail
		m, err := writeImage(filename, fileURL, thumbnailDirBase, md5Sum, sc, mTime, runThumbnailer)
		if err != nil {
			return nil, ``, err
		}
		if m != nil {
			img = m
			goto resize
		}
	} else {
		t, err := newThumbnailFile(thumbnailFilename)
		if err != nil {
			return nil, ``, errors.New(`unable to open thumbnail image`)
		}
		mTimeThumbnail, err := t.mTimeStr()
		if err != nil {
			return nil, ``, errors.New(`unable to get modification time of thumbnail image`)
		}
		if mTime != mTimeThumbnail {
			// rewrite thumbnail
			m, err := writeImage(filename, fileURL, thumbnailDirBase, md5Sum, sc, mTime, runThumbnailer)
			if err != nil {
				return nil, ``, err
			}
			if m != nil {
				img = m
				goto resize
			}
		} else {
			// file already exists
			fi, err := os.Open(thumbnailFilename)
			if err != nil {
				return nil, ``, err
			}
			defer fi.Close()
			m, _, err := image.Decode(fi)
			if err != nil {
				return nil, ``, err
			}
			if m != nil {
				img = m
				goto resize
			}
		}
	}
	if img == nil {
		return nil, ``, errors.New(`unable to create thumbnail`)
	}

resize:
	if size.X > 1 {
		img = resize.Resize(uint(size.X), 0, img, resize.Lanczos3)
	} else {
		img = resize.Resize(0, uint(size.Y), img, resize.Lanczos3)
	}
	return img, thumbnailFilename, nil
}

func hashFilename(filename string, portable bool) (md5hash, thumbnailFileURL string) {
	fileURL := fileURLScheme

	if portable {
		// this makes the file path relative and the created file uri might be invalid(?)
		// see https://specifications.freedesktop.org/thumbnail-spec/thumbnail-spec-0.8.0.html#SHARED // TODO rm
		// see https://specifications.freedesktop.org/thumbnail-spec/thumbnail-spec-latest.html#SHARED
		filename = filepath.Base(filename)
	}
	for _, r := range filename {
		switch {
		// needs checking!
		case (r >= 'a' && r <= 'z') || (r >= '@' && r <= 'Z') || (r >= '&' && r <= ':') || r == '!' || r == '$' || r == '=' || r == '_' || r == '~':
			fileURL += string(r)
		default:
			fileURL += url.QueryEscape(string(r))
		}
	}

	h := md5.New()
	io.WriteString(h, fileURL)
	md5sum := fmt.Sprintf(`%x`, h.Sum(nil))

	return md5sum, fileURL
}

func writeImage(filename, fileURL, thumbnailDirBase, md5Sum, scale, mTime string, runThumbnailer bool) (image.Image, error) {
	// TODO look if larger thumbnails exist before creating one
	var softwareStr string
	thumbnailFilename := filepath.Join(thumbnailDirBase, scale, md5Sum+`.png`)
	thumbnailDir := filepath.Join(thumbnailDirBase, scale)
	{
		o := strings.Split(os.Args[0], ps)
		softwareStr = software + `::` + o[len(o)-1]
	}

	// var alpha uint8 = 0xff   // enforced

	if err := os.MkdirAll(thumbnailDir, 0700); err != nil {
		return nil, err
	}
	var img image.Image
	var buf bytes.Buffer
	if runThumbnailer {
		sz := 1 << (10 - slices.Index(scales, scale)) // 128, 256, 512, 1024
		mimeType, err := runNativeThumbnailer(filename, fileURL, thumbnailFilename, uint(sz))
		if err != nil {
			switch mimeType {
			case `image/png`,
				`image/jpeg`,
				`image/bmp`,
				`image/gif`,
				`image/tiff`,
				`image/webp`:
			default:
				return nil, err
			}
		} else {
			ft, err := os.Open(thumbnailFilename)
			if err != nil {
				return nil, err
			}
			defer ft.Close()
			if _, err := io.Copy(&buf, ft); err != nil {
				return nil, err
			}
			if _, err := ft.Seek(0, io.SeekStart); err != nil {
				return nil, err
			}
			img, _, err = image.Decode(ft)
			if err != nil {
				return nil, err
			}
			_ = ft.Close()
			// TODO check if correctly sized
		}
	}
	// add exif tags if natively created
	if img == nil {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		m, _, err := image.Decode(f)
		if err != nil {
			return nil, err
		}
		_ = f.Close()
		img = m

		bounds := img.Bounds()
		sz := 1 << (10 - slices.Index(scales, scale)) // 128, 256, 512, 1024
		if bounds.Dx() >= bounds.Dy() {
			img = resize.Resize(uint(sz), 0, img, resize.Lanczos3)
		} else {
			img = resize.Resize(0, uint(sz), img, resize.Lanczos3)
		}
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	}

	pmp := pngstructure.NewPngMediaParser()
	csTmp, err := pmp.ParseBytes(buf.Bytes())
	if err != nil {
		return nil, err
	}
	cs, okCS := csTmp.(*pngstructure.ChunkSlice)
	if !okCS {
		return nil, errors.New(`type not *pngstructure.ChunkSlice`)
	}

	chunks := cs.Chunks()
	textChunks := []*pngstructure.Chunk{
		// TODO set global constants for chunk names
		{Type: `tEXt`, Data: []uint8("Thumb::URI\x00" + fileURL)},
		{Type: `tEXt`, Data: []uint8("Thumb::MTime\x00" + mTime)},
		{Type: `tEXt`, Data: []uint8("Software\x00" + softwareStr)},
	}
	for _, textChunk := range textChunks {
		textChunk.Length = uint32(len(textChunk.Data))
		textChunk.UpdateCrc32()
	}
	// insert tEXt chunks after IHDR
	chunks = append(
		chunks[:1],
		append(
			textChunks,
			chunks[1:]...,
		)...,
	)

	cs = pngstructure.NewChunkSlice(chunks)
	ft, err := os.OpenFile(thumbnailFilename+`.tmp`, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer ft.Close()
	if err := cs.WriteTo(ft); err != nil {
		return nil, err
	}
	if err := os.Rename(thumbnailFilename+`.tmp`, thumbnailFilename); err != nil {
		return nil, err
	}

	return img, nil
}
