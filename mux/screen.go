package mux

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	"github.com/srlehn/termimg/internal/exc"
)

func screenWrap(s string) string {
	/*
	   # https://github.com/chromium/hterm/blob/6846a85/etc/osc52.sh#L23
	   #
	   #  (v4.2.1) Apr 2014 - today: 768 bytes
	   #  Aug 2008 - Apr 2014 (v4.2.0): 512 bytes
	   #  ??? - Aug 2008 (v4.0.3): 256 bytes
	   # Since v4.2.0 is only ~4 years old, we'll use the 256 limit.
	   # We can probably switch to the 768 limit in 2022.
	*/
	// # DCS("\033P") + (256|512|768)-4 + ST("\033\\")
	// TODO split containted ST ("\033\\")!
	DCS := "\033P"
	ST := "\033\\"
	var limit int = 256
	ver := getScreenVersion()
	if ver[0] >= 4 {
		if ver[1] > 2 || (ver[1] == 2 && ver[2] >= 1) {
			limit = 768
		} else if ver[2] >= 3 {
			limit = 512
		}
	}
	limitScreen := limit
	limit -= len(DCS) + 2*len(ST)
	parts := strings.Split(s, ST)
	b := &strings.Builder{}
	b.Grow(len(s) + (len(parts)+len(s)/limitScreen+1)*len(DCS+ST))
	initPart := 1
	b.WriteString(DCS)
	for _, part := range parts {
		if initPart == 0 {
			b.WriteString("\033" + ST + DCS + `\`)
		}
		initPart = 0
		i := limit + initPart
		j := 0
		var newPart = true
		for ; i < len(part); i += limit + initPart {
			if initPart == 0 && !newPart {
				b.WriteString(ST + DCS)
			}
			b.WriteString(part[j:i])
			j = i
			newPart = false
		}
		if !newPart {
			b.WriteString(ST + DCS)
		}
		b.WriteString(part[j:])
	}
	b.WriteString(ST)

	return b.String()
}

func getScreenClientPID(pidServer int32) (int32, error) {
	lsofExeAbs, err := exc.LookSystemDirs(`lsof`)
	if err != nil {
		return 0, err
	}
	lsofReplTTY, err := exec.Command(
		lsofExeAbs,
		`-p`, strconv.Itoa(int(pidServer)),
		`-a`,
		`+D`, `/dev/pts/`,
		`-Fn`,
	).Output() // TODO use go
	_ = err // lsof exits with exit code > 0 even when successful
	scannerTTY := bufio.NewScanner(bytes.NewBuffer(lsofReplTTY))
	var tty string
	for scannerTTY.Scan() {
		line := scannerTTY.Text()
		t, found := strings.CutPrefix(line, `n`)
		if found {
			// TODO multi clients
			tty = t
			break
		}
	}
	if len(tty) == 0 {
		return 0, errors.New(`unable to find tty attached to screen server`)
	}
	lsofReplPIDClient, err := exec.Command(lsofExeAbs, `-Fpc`, tty).Output() // TODO use go
	if err != nil {
		return 0, errors.New(err)
	}
	scannerPIDClient := bufio.NewScanner(bytes.NewBuffer(lsofReplPIDClient))
	var pidStr string
	var succeeded bool
	pidServerStr := `p` + strconv.Itoa(int(pidServer))
	for scannerPIDClient.Scan() {
		line := scannerPIDClient.Text()
		if strings.HasPrefix(line, `p`) {
			pidStr = line
			continue
		}
		if line == `c`+`screen` && pidStr != pidServerStr {
			// TODO multi clients
			succeeded = true
			break
		}
	}
	if !succeeded || len(pidStr) == 0 {
		return 0, errors.New(`unable to find process which have "` + tty + `" opened`)
	}
	pidClient, err := strconv.Atoi(strings.TrimPrefix(pidStr, `p`))
	if err != nil {
		return 0, errors.New(err)
	}
	return int32(pidClient), nil
}

var screenVersion [3]uint

func getScreenVersion() [3]uint {
	// e.g.: "Screen version 4.08.00 (GNU) 05-Feb-20"
	if screenVersion != [3]uint{} {
		return screenVersion
	}
	var ver [3]uint
	screenExe, err := exc.LookSystemDirs(`screen`)
	if err != nil {
		return ver
	}
	cmd := exec.Command(screenExe, `-v`)
	cmd.Env = []string{`LC_ALL=C`}
	repl, err := cmd.Output()
	if err != nil {
		return ver
	}
	screenVersionPrefix := `Screen version `
	verParts := strings.Split(strings.SplitN(strings.TrimPrefix(string(repl), screenVersionPrefix), ` `, 2)[0], `.`)
	if len(verParts) != 3 {
		return ver
	}
	major, err := strconv.ParseUint(verParts[0], 10, 64)
	if err != nil {
		return ver
	}
	minor, err := strconv.ParseUint(verParts[1], 10, 64)
	if err != nil {
		return ver
	}
	patch, err := strconv.ParseUint(verParts[2], 10, 64)
	if err != nil {
		return ver
	}
	ver = [3]uint{uint(major), uint(minor), uint(patch)}
	screenVersion = ver
	return ver
}

// https://savannah.gnu.org/bugs/index.php?56063#comment3
