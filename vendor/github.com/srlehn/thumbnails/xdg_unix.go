//go:build unix && !android && !ios

package thumbnails

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/rkoesters/xdg/keyfile"
)

func runNativeThumbnailer(filename, fileURI, filenameThumbnail string, size uint) (mimeType string, _ error) {
	mt, err := mimetype.DetectFile(filename)
	if err != nil {
		return ``, err
	}
	mimeType = mt.String()

	var exc, tryExc string
	_ = tryExc // TODO
	walkFunc := fs.WalkDirFunc(func(path string, d fs.DirEntry, err error) error {
		_ = err
		if !strings.HasSuffix(path, `.thumbnailer`) {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		kf, err := keyfile.New(f)
		if err != nil || kf == nil {
			return nil
		}
		_ = f.Close()
		thEntryKey := `Thumbnailer Entry`
		execKey := `Exec`
		tryExecKey := `TryExec`
		mimeTypeKey := `MimeType`
		var ex, tryEx, mimeTypeStr string
		if kf.KeyExists(thEntryKey, mimeTypeKey) {
			mimeTypeStr, err = kf.String(thEntryKey, mimeTypeKey)
			if err != nil {
				return nil
			}
		}
		mimeTypes := strings.Split(mimeTypeStr, `;`)
		var match bool
		for _, mt := range mimeTypes {
			if mimeType == mt {
				match = true
				break
			}
		}
		if !match {
			return nil
		}
		if kf.KeyExists(thEntryKey, execKey) {
			ex, err = kf.String(thEntryKey, execKey)
			if err != nil {
				return nil
			}
		}
		if kf.KeyExists(thEntryKey, tryExecKey) {
			tryEx, _ = kf.String(thEntryKey, tryExecKey)
		}
		exc = ex
		tryExc = tryEx
		return fs.SkipAll
	})
	if err := filepath.WalkDir(`/usr/share/thumbnailers/`, walkFunc); err != nil {
		return ``, err
	}
	repl := strings.NewReplacer(
		`%%`, `%`,
		`%i`, filename,
		`%u`, fileURI,
		`%o`, filenameThumbnail,
		`%s`, strconv.Itoa(int(size)),
	)
	execParts := strings.Split(exc, ` `)
	var execArgs []string
	for _, ep := range execParts {
		e := repl.Replace(ep)
		if len(e) > 0 {
			execArgs = append(execArgs, e)
		}
	}
	if len(execArgs) < 1 {
		return ``, errors.New(`null command`)
	}
	cmd := exec.Command(execArgs[0], execArgs[1:]...)
	if err := cmd.Run(); err != nil {
		return ``, err
	}
	return mimeType, nil
}
