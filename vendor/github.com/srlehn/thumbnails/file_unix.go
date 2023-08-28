//go:build unix && !android && !ios

package thumbnails

import (
	"errors"
	"image"
	"net/url"
	"strconv"
	"strings"
	"time"

	pngstructure "github.com/dsoprea/go-png-image-structure/v2"
)

type thumbnailFile struct {
	chunks *pngstructure.ChunkSlice
}

func newThumbnailFile(filename string) (*thumbnailFile, error) {
	pmp := pngstructure.NewPngMediaParser()
	csMC, err := pmp.ParseFile(filename)
	if err != nil {
		return nil, err
	}
	chunks, ok := csMC.(*pngstructure.ChunkSlice)
	if !ok {
		return nil, errors.New(`type not *pngstructure.ChunkSlice`)
	}
	return &thumbnailFile{chunks: chunks}, nil
}
func (t *thumbnailFile) textChunk(chunkName string) (string, error) {
	if t == nil {
		return ``, errors.New(`nil receiver`)
	}
	for _, chunk := range t.chunks.Chunks() {
		chunkStr := string(chunk.Data)
		if chunk.Type == `tEXt` && strings.HasPrefix(chunkStr, chunkName+"\x00") {
			return chunkStr[len(chunkName)+1:], nil
		}
	}

	return ``, nil
}
func (t *thumbnailFile) uri() (string, error) {
	uriStr := "Thumb::URI"
	return t.textChunk(uriStr)
}
func (t *thumbnailFile) filename() (string, error) { // original filename
	uri, err := t.mTimeStr()
	if err != nil {
		return ``, err
	}
	path, found := strings.CutPrefix(uri, fileURLScheme)
	if !found {
		return ``, errors.New(`not a file uri`)
	}
	path, err = url.QueryUnescape(path)
	if err != nil {
		return ``, err
	}
	return path, nil
}
func (t *thumbnailFile) mTimeStr() (string, error) {
	mtimeStr := "Thumb::MTime"
	return t.textChunk(mtimeStr)
}
func (t *thumbnailFile) mTime() (time.Time, error) {
	var mTime time.Time
	mTimeStr, err := t.mTimeStr()
	if err != nil {
		return mTime, err
	}
	mTimeUnix, err := strconv.ParseInt(mTimeStr, 10, 64)
	if err != nil {
		return mTime, err
	}
	mTime = time.Unix(mTimeUnix, 0)
	return mTime, nil
}
func (t *thumbnailFile) fileSize() (uint, error) {
	chunkName := "Thumb::Size"
	sizeStr, err := t.textChunk(chunkName)
	if err != nil {
		return 0, err
	}
	fileSize, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(fileSize), nil
}
func (t *thumbnailFile) mimeType() (string, error) {
	chunkName := "Thumb::Mimetype"
	return t.textChunk(chunkName)
}
func (t *thumbnailFile) imageSize() (image.Point, error) {
	var sz image.Point
	chunkNameImgWidthStr := "Thumb::Image::Width"
	chunkNameImgHeightStr := "Thumb::Image::Height"
	ws, err := t.textChunk(chunkNameImgWidthStr)
	if err != nil {
		return sz, err
	}
	w, err := strconv.Atoi(ws)
	if err != nil {
		return sz, err
	}
	hs, err := t.textChunk(chunkNameImgHeightStr)
	if err != nil {
		return sz, err
	}
	h, err := strconv.Atoi(hs)
	if err != nil {
		return sz, err
	}
	sz = image.Point{X: w, Y: h}
	return sz, nil
}
func (t *thumbnailFile) documentPages() (uint, error) {
	chunkName := "Thumb::Document::Pages"
	docPagesStr, err := t.textChunk(chunkName)
	if err != nil {
		return 0, err
	}
	docPages, err := strconv.ParseUint(docPagesStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(docPages), nil
}
func (t *thumbnailFile) movieLength() (time.Duration, error) {
	var movLen time.Duration
	chunkName := "Thumb::Movie::Length"
	movLenStr, err := t.textChunk(chunkName)
	if err != nil {
		return 0, err
	}
	movLenI64, err := strconv.ParseInt(movLenStr, 10, 64)
	if err != nil {
		return 0, err
	}
	movLen = time.Duration(movLenI64) * time.Second
	return movLen, nil
}
