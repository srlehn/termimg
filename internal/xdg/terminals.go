package xdg

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rkoesters/xdg/desktop"

	"github.com/srlehn/termimg/internal/errors"
)

func InstalledTerminals() ([]*desktop.Entry, error) {
	var xdgDataDirs []string
	xdgDataDirsStr, okDirs := os.LookupEnv(`XDG_DATA_DIRS`)
	if okDirs {
		xdgDataDirs = strings.Split(xdgDataDirsStr, `:`)
	} else {
		xdgDataDirs = []string{`/usr/local/share`, `/usr/share`}
	}
	var desktopFileNames []string
	var entries []*desktop.Entry
	var walkDirFunc fs.WalkDirFunc = func(filename string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d == nil {
			return nil
		}
		if strings.HasSuffix(filename, `.desktop`) {
			desktopFileNames = append(desktopFileNames, filename)
			f, err := os.Open(filename)
			if err != nil {
				return nil
			}
			defer f.Close()
			entry, err := desktop.New(f)
			if err != nil {
				return nil
			}
			if entry == nil {
				return nil
			}
			for _, cat := range entry.Categories {
				if cat == `TerminalEmulator` {
					entries = append(entries, entry)
					return nil
				}
			}
			return nil
		}
		return nil
	}
	for _, xdgDataDir := range xdgDataDirs {
		_ = filepath.WalkDir(filepath.Join(xdgDataDir, `applications`), walkDirFunc)
	}

	return entries, nil
}

func InstalledTerminalsExe() ([]string, error) {
	entries, err := InstalledTerminals()
	if err != nil {
		return nil, errors.New(err)
	}
	var exes []string
	var exe, lastExe string
	for _, entry := range entries {
		lastExe = exe
		// TODO - remove user installed binaries (not in a system path)
		exe = filepath.Base(strings.Split(entry.Exec, ` `)[0])
		if exe != lastExe {
			exes = append(exes, exe)
		}
	}
	sort.Strings(exes)
	return exes, nil
}
