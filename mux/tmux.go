package mux

import (
	"bufio"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/iancoleman/strcase"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/exc"
)

func tmuxWrap(s string) string {
	return "\033Ptmux;" + strings.Replace(s, "\033", "\033\033", -1) + "\033\\"
}

type TmuxPane struct {
	// for usage with tmux command list-panes
	PaneWidth  uint
	PaneHeight uint
	PaneLeft   uint
	// PaneRight uint
	PaneTop uint
	// PaneBottom uint
	PaneID    string
	PaneIndex uint
	PanePID   string
	PaneTTY   string
	// PanePipe    bool
	// PaneCurrentCommand string
	// AlternateOn bool
	// OriginFlag bool
	WindowID    string
	WindowIndex uint
	// WindowName string
	SessionID string
	// SessionName string
	PID uint // server pid
}

func (p *TmuxPane) Args() []string { return []string{`list-panes`, `-a`} }

type TmuxClient struct {
	// for usage with tmux command list-clients
	SessionID        string
	SessionName      string
	PID              uint
	SessionWindows   uint
	ClientPID        uint
	ClientTermname   string
	ClientName       string
	ClientTTY        string
	ClientUTF8       bool
	ClientWidth      uint
	ClientHeight     uint
	ClientCellWidth  uint
	ClientCellHeight uint
}

func (p *TmuxClient) Args() []string { return []string{`list-clients`} }

type TmuxInfo struct {
	time    time.Time
	panes   []*TmuxPane
	clients []*TmuxClient
}

func (i *TmuxInfo) ClientPIDOfPane(paneID string) int32 {
	var sessID string
	for _, p := range i.panes {
		if paneID != p.PaneID {
			continue
		}
		sessID = p.SessionID
	}
	var pidClient uint
	for _, c := range i.clients {
		if sessID != c.SessionID {
			continue
		}
		pidClient = c.ClientPID
	}
	return int32(pidClient)
}

const validityDuration = 3 * time.Second

func (i *TmuxInfo) Query() error {
	// Query takes around 5ms on my system
	if i == nil {
		i = &TmuxInfo{}
	}
	if time.Since(i.time) < validityDuration && len(i.panes) > 0 && len(i.clients) > 0 {
		return nil
	}
	sep := `|`
	replPanes, err := execTmux[TmuxPane](sep)
	if err != nil {
		return err
	}
	panes, err := tmuxParseOutput[TmuxPane](replPanes, sep)
	if err != nil {
		return err
	}
	i.panes = panes
	replClients, err := execTmux[TmuxClient](sep)
	if err != nil {
		return err
	}
	clients, err := tmuxParseOutput[TmuxClient](replClients, sep)
	if err != nil {
		return err
	}
	i.clients = clients
	return nil
}

func getFieldNames(p any) []string {
	t := reflect.TypeOf(p)
	if t.Kind() != reflect.Ptr {
		return nil
	}
	var fields []string
	for i := 0; i < t.Elem().NumField(); i++ {
		fieldName := t.Elem().Field(i).Name
		fieldNameSnakeCase := strcase.ToSnake(fieldName)
		// don't split "UTF8"
		var fieldNameSnakeCase2 string
		var d rune
		for _, c := range fieldNameSnakeCase {
			if d == 0 || (d == '_' && c >= '0' && c <= '9') {
				d = c
				continue
			}
			fieldNameSnakeCase2 += string(d)
			d = c
		}
		fieldNameSnakeCase2 += string(d)

		fields = append(fields, fieldNameSnakeCase2)
	}
	return fields
}

func execTmux[T any](sep string) (string, error) {
	tmuxAbs, err := exc.LookSystemDirs(`tmux`)
	if err != nil {
		return ``, err
	}
	if len(sep) == 0 {
		sep = `|`
	}
	obj := new(T)
	fields := getFieldNames(obj)
	fmtStr := `#{` + strings.Join(fields, `}`+sep+`#{`) + `}`
	var args []string
	if arger, ok := any(obj).(internal.Arger); ok {
		args = arger.Args()
	}
	command := exec.Command(
		tmuxAbs,
		append(
			// append([]string{`-S`, `/tmp/tmux-` + strconv.Itoa(os.Getuid()) + `/default`}, args...),
			args,
			[]string{
				`-F`,
				fmtStr,
			}...)...,
	)
	repl, err := command.Output()
	if err != nil {
		return ``, errors.New(err)
	}
	return string(repl), nil
}

func tmuxParseOutput[T any](str, sep string) ([]*T, error) {
	var ret []*T
	var researchObject T
	numFields := reflect.TypeOf(researchObject).NumField()

	scanner := bufio.NewScanner(strings.NewReader(str))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), sep)
		if len(parts) != numFields {
			return nil, errors.New(`invalid field count`)
		}
		obj := new(T)
		for i := 0; i < numFields; i++ {
			field := reflect.ValueOf(obj).Elem().Field(i)
			switch field.Kind() {
			case reflect.String:
				field.SetString(parts[i])
			case reflect.Uint:
				uintPart, err := strconv.ParseUint(parts[i], 10, 64)
				if err != nil {
					// TODO log
					continue
				}
				field.SetUint(uintPart)
			case reflect.Bool:
				boolPart, err := strconv.ParseBool(parts[i])
				if err != nil {
					// TODO log
					continue
				}
				field.SetBool(boolPart)
			default:
				// TODO log
			}
		}
		ret = append(ret, obj)
	}

	return ret, nil
}
