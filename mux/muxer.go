// <Copyright> 2019 Simon Robin Lehn. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package mux

import (
	"math"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/procextra"
	"github.com/srlehn/termimg/internal/propkeys"
)

var (
	wrappers = map[string]func(string) string{
		`tmux`:   tmuxWrap,
		`screen`: screenWrap,
	}
)

type Muxer struct {
	name       string
	procServer *process.Process
	procClient *process.Process
	env        environ.Properties
	ttyInner   string
	isRemote   bool
}

func (m *Muxer) Wrap(s string) string {
	if m == nil || len(m.name) == 0 {
		return s
	}
	wrapFn, ok := wrappers[m.name]
	if !ok || wrapFn == nil {
		return s
	}
	return wrapFn(s)
}
func (m *Muxer) TTY() string {
	if m == nil || len(m.ttyInner) == 0 {
		return ``
	}
	if runtime.GOOS != `windows` {
		return `/dev` + m.ttyInner
	}
	return m.ttyInner
}
func (m *Muxer) IsRemote() bool { return m != nil && m.isRemote }

type Muxers []*Muxer

func (m Muxers) Wrap(s string) string {
	for i := len(m) - 1; i >= 0; i-- {
		if m[i] == nil || m[i].procServer == nil {
			continue
		}
		// skip muxer clients
		tty, err := procextra.TTYOfProc(m[i].procServer)
		if err != nil || len(tty) == 0 {
			continue
		}
		s = m[i].Wrap(s)
	}
	return s
}
func (m Muxers) String() string {
	b := &strings.Builder{}
	for i, muxer := range m {
		if muxer == nil {
			continue
		}
		if i > 0 {
			b.WriteString(`>`)
		}
		b.WriteString(muxer.name)
	}
	return b.String()
}
func (m Muxers) IsRemote() bool {
	for _, muxer := range m {
		if muxer.IsRemote() {
			return true
		}
	}
	return false
}

// TODO: func (m Muxers) (Un)Wrapper() io.ReadWriter

func Wrap(s string, pr environ.Properties) string {
	if pr == nil {
		return s
	}
	passagesStr, ok := pr.Property(propkeys.Passages)
	if !ok {
		return s
	}
	passages := strings.Split(passagesStr, `>`)
	for _, passage := range passages {
		wrap, ok := wrappers[passage]
		if !ok || wrap == nil {
			// TODO should we continue here?
			continue
		}
		s = wrap(s)
	}
	return s
}

// ps -ewwo pid=,ppid=,tty=,comm=

func FindTerminalProcess(pid int32) (procTerm *process.Process, ttyInner string, envInner environ.Properties, passages Muxers, e error) {
	// TODO handle errors
	// TODO make this testable...
	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, ``, nil, nil, errors.New(err)
	}
	var procLast, p *process.Process
	var tty, ttyLast, termType string
	var isRemoteTotal bool
Outer:
	for {
		for {
			ttyLast = tty
			tty, err = procextra.TTYOfProc(proc)
			if err != nil {
				break Outer
			}
			if procLast != nil && tty != ttyLast {
				break
			}
			procLast = proc

			p, err = procextra.ParentOfProc(proc)
			if err != nil || p == nil || p.Pid < 2 {
				// break Outer
				break
			}
			proc = p
		}
		if proc != nil && proc.Pid == pid {
			tty, _ := procextra.TTYOfProc(proc)
			children, err := proc.Children()
			if err == nil {
				shellVar, okShell := os.LookupEnv(`SHELL`)
				if okShell {
					shellVar = filepath.Base(shellVar)
				}
				isShell := func(name string) bool {
					name = filepath.Base(name)
					if okShell {
						return name == shellVar
					} else {
						switch name {
						case `bash`,
							`sh`,
							`ash`, `dash`,
							`csh`, `tcsh`,
							`ksh`, `ksh88`, `ksh93`, `ksh2020`, `pdksh`, `mksh`, `dtksh`, `oksh`, `loksh`, `SKsh`,
							`zsh`,
							`hush`,
							`es`, `rc`,
							`fish`:
							return true
						default:
							return false
						}
					}
				}
				// get first child shell process
				for _, child := range children {
					if child == nil {
						continue
					}
					ttyChild, err := procextra.TTYOfProc(child)
					// login shells on the console
					if err != nil || ttyChild != tty {
						continue
					}
					childName, err := child.Name()
					if err != nil || !isShell(childName) {
						continue
					}
					procLast = child
					break
				}
			}
		}
		// found a tty change
		var isRemote bool
		p, termType, isRemote, err = getClientProc(proc, procLast, termType)
		if isRemote {
			isRemoteTotal = true
		}
		if err != nil {
			break
		}
		if len(termType) > 0 {
			if len(ttyLast) > 0 && !strings.HasSuffix(termType, `-client`) {
				var env []string
				if en, err := proc.Environ(); err == nil {
					env = en
				}
				var procClient *process.Process
				if p != nil && p.Pid != proc.Pid {
					procClient = p
				}
				passages = append(passages, &Muxer{
					name:       strings.TrimSuffix(termType, `-server`),
					procServer: proc,
					procClient: procClient,
					env:        environ.EnvToProperties(env),
					ttyInner:   ttyLast,
					isRemote:   isRemote,
				})
			}
		} else {
			// end station - terminal!
			break Outer
		}
		if p == nil {
			// unable to find client pid
			break
		} else if p.Pid == proc.Pid {
			if p, err := proc.Parent(); err == nil && p != nil {
				termType = ``
				proc = p
			}
		}
		proc, procLast = p, proc
	}
	// ignore error - empty parent env will only prevent cleaning
	var envrnOuter []string
	if proc != nil {
		envrnOuter, _ = procextra.EnvOfProc(proc)
	}
	if procLast == nil {
		if children, err := proc.Children(); err == nil && len(children) > 0 {
			procLast = children[0]
		}
	}
	var errRet error
	if procLast != nil {
		var envrnInner []string
		if procParent, err := procextra.ParentOfProc(procLast); err == nil && procParent != nil {
			envrnInner, err = procextra.EnvOfProc(procLast)
			if err != nil {
				errRet = errors.New(err)
			}
		}
		envInner = environ.CleanEnv(envrnOuter, envrnInner)
	} else {
		envInner = environ.NewProperties()
	}
	if passagesStr := passages.String(); len(passagesStr) > 0 {
		envInner.SetProperty(propkeys.Passages, passagesStr)
	}
	if isRemoteTotal {
		envInner.SetProperty(propkeys.IsRemote, `true`)
	}
	return proc, ttyLast, envInner, passages, errRet
}

func findTTYProc(proc *process.Process) (procTTY, procInner *process.Process, ttyTTY, ttyInner string, e error) {
	if proc == nil || proc.Pid < 2 {
		return nil, nil, ``, ``, errors.New(`nil param or 0 or init pid`)
	}
	tty, err := procextra.TTYOfProc(proc)
	if err != nil {
		return nil, nil, ``, ``, errors.New(err)
	}
	var procParent *process.Process
	var ttyParent string
	for {
		procParent, err = proc.Parent()
		if err != nil {
			return nil, nil, ``, ``, errors.New(err)
		}
		ttyParent, err = procextra.TTYOfProc(procParent)
		if err != nil {
			return nil, nil, ``, ``, errors.New(err)
		}
		if ttyParent != tty {
			break
		}
		proc = procParent
		tty = ttyParent
	}
	return procParent, proc, ttyParent, tty, nil
}

func getClientProc(procTerm, procInner *process.Process, termTypeLast string) (procClient *process.Process, termType string, isRemote bool, e error) {
	// procTerm is a terminal, muxer, etc
	var exe string
	if x, err := procTerm.Exe(); err == nil {
		exe = filepath.Base(x)
	}
	if runtime.GOOS == `windows` {
		exe, _ = strings.CutSuffix(exe, `.exe`)
	}
	var preservedVars []string
	var env environ.Properties
	if parent, err := procInner.Parent(); err == nil && parent.Pid == procTerm.Pid {
		envTerm, err := procTerm.Environ()
		if err != nil && !os.IsPermission(err) {
			return nil, ``, false, errors.New(err)
		}
		envInner, err := procInner.Environ()
		if err != nil && !os.IsPermission(err) {
			return nil, ``, false, errors.New(err)
		}
		env = environ.CleanEnv(envTerm, envInner)
		// preserve ssh variables for mosh, zellij
		for _, v := range envTerm {
			for _, prefix := range []string{`SSH_CLIENT`, `SSH_CONNECTION`, `SSH_TTY`, `ZELLIJ`} {
				if !strings.HasPrefix(v, prefix) {
					continue
				}
				vParts := strings.SplitN(v, `=`, 2)
				if len(vParts) != 2 {
					continue
				}
				env.SetProperty(propkeys.PreservedOuterEnvPrefix+vParts[0], vParts[1])
				preservedVars = append(preservedVars, v)
			}
		}
	}
	if env == nil {
		env = environ.NewProperties()
	}
	envTemp := environ.CloneProperties(env)
	envTemp.MergeProperties(environ.EnvToProperties(preservedVars))
	// run different checks here for muxers, etc...
	var (
		err                        error
		tmuxPaneVarName            = `TMUX_PANE`
		screenSTYVarName           = `STY`
		mtmPIDVarName              = `MTM`
		zellijVarName              = `ZELLIJ`
		neercsVarName              = `CACA_DRIVER`
		dvtmWindowIDVarName        = `DVTM_WINDOW_ID`
		sshTTYVarName              = `SSH_TTY`
		sshClientVarName           = `SSH_CLIENT`
		sshConnVarName             = `SSH_CONNECTION`
		moshServerCreationMaxDur   = 4500 * time.Millisecond // usually around 2s on my system
		zellijServerCreationMaxDur = 3500 * time.Millisecond // usually below <1ms on my system
	)
	if paneID, isTmux := envTemp.LookupEnv(tmuxPaneVarName); isTmux {
		if len(paneID) == 0 {
			return nil, ``, false, errors.New(`found no tmux pane id`)
		}
		info := new(TmuxInfo)
		if err := info.Query(); err != nil {
			return nil, ``, false, err
		}
		termType = `tmux-server`
		pidClient := info.ClientPIDOfPane(paneID)
		procClient, err = process.NewProcess(pidClient)
		if err != nil {
			return nil, ``, false, errors.New(err)
		}
	} else if sty, isScreen := envTemp.LookupEnv(screenSTYVarName); isScreen && len(sty) > 0 {
		user, err := user.Current()
		if err != nil {
			return nil, ``, false, errors.New(err)
		}
		if _, err := os.Stat(`/run/screen/S-` + user.Username + `/` + sty); err != nil {
			// file might not exist --> tmux not screen!
			procClient, err = procTerm.Parent()
			if err != nil {
				return nil, ``, false, errors.New(err)
			}
			termType = `tmux-server`
		} else {
			pidClient, err := getScreenClientPID(procTerm.Pid)
			if err != nil {
				return nil, ``, false, err
			}
			termType = `screen-server`
			procClient, err = process.NewProcess(pidClient)
			if err != nil {
				return nil, ``, false, errors.New(err)
			}
		}
	} else if mtmPid, isMTM := envTemp.LookupEnv(mtmPIDVarName); isMTM && len(mtmPid) > 0 {
		termType = `mtm`
		procClient = procTerm
	} else if zellijID, isZellij := envTemp.LookupEnv(zellijVarName); isZellij && len(zellijID) > 0 {
		// only able to guess clients of newly created sessions based on similar process creation times
		// TODO find newer clients
		termType = `zellij-server`
		var serverCreationTime int64
		processes, err := process.Processes()
		if err != nil {
			goto skipZellijClient
		}
		serverCreationTime, err = procTerm.CreateTime()
		if err != nil {
			goto skipZellijClient
		}
		for _, proc := range processes {
			clientCreationTime, err := proc.CreateTime()
			if err != nil {
				continue
			}
			if proc.Pid == procTerm.Pid {
				continue
			}
			if int64(math.Abs(float64(clientCreationTime-serverCreationTime))) > zellijServerCreationMaxDur.Milliseconds() {
				continue
			}
			x, err := proc.Exe()
			if err != nil {
				continue
			}
			x = filepath.Base(x)
			if runtime.GOOS == `windows` {
				x = strings.TrimSuffix(x, `.exe`)
			}
			if !strings.HasPrefix(strings.ToLower(x), `zellij`) {
				continue
			}
			procClient = proc
			goto endZellijClient
		}
	skipZellijClient:
		return nil, termType, false, errors.New(`unable to find zellij client`)
	endZellijClient:
	} else if cacaDriver, isNeercs := envTemp.LookupEnv(neercsVarName); isNeercs && len(cacaDriver) > 0 {
		termType = `neercs-server`
		// neercs client process is parent of server process
		p, err := procTerm.Parent()
		if err != nil {
			return nil, termType, false, errors.New(`unable to find neercs client`)
		}
		procClient = p
	} else if dvtmWindowID, isDVTM := envTemp.LookupEnv(dvtmWindowIDVarName); isDVTM && len(dvtmWindowID) > 0 {
		termType = `dvtm`
		procClient = procTerm // TODO ?
	} else if sshTTY, isSSHOrMosh := envTemp.LookupEnv(sshTTYVarName); isSSHOrMosh && len(sshTTY) > 0 {
		var ttyInner string
		if procInner != nil {
			if t, err := procextra.TTYOfProc(procInner); err == nil {
				ttyInner = t
			}
		}
		termType = `ssh|mosh`
		if len(ttyInner) > 0 {
			if runtime.GOOS != `windows` {
				ttyInner = `/dev` + ttyInner
			}
			if sshTTY == ttyInner {
				termType = `ssh`
				var lAddr, rAddr net.Addr
				var lAddrPort, rAddrPort int
				processes, err := process.Processes()
				if err != nil {
					goto skipSSHClient
				}
				{
					sshConn, ok := envTemp.LookupEnv(sshConnVarName)
					if !ok {
						goto skipSSHClient
					}
					sshConnParts := strings.Split(sshConn, ` `)
					if len(sshConnParts) != 4 {
						goto skipSSHClient
					}
					rAddrPort, err = strconv.Atoi(sshConnParts[1])
					if err != nil {
						goto skipSSHClient
					}
					lAddrPort, err = strconv.Atoi(sshConnParts[3])
					if err != nil {
						goto skipSSHClient
					}
					rAddr = net.Addr{IP: sshConnParts[0], Port: uint32(rAddrPort)}
					lAddr = net.Addr{IP: sshConnParts[2], Port: uint32(lAddrPort)}
					switch rAddr.IP {
					case `127.0.0.1`, `::1`:
					default:
						isRemote = true
					}
				}
				for _, proc := range processes {
					conns, err := proc.Connections()
					if err != nil {
						continue
					}
					for _, conn := range conns {
						if conn.Laddr == rAddr && conn.Raddr == lAddr {
							procClient = proc
							goto endSSHClient
						}
					}
				}
			skipSSHClient:
				isRemote = true
			endSSHClient:
			} else {
				termType = `mosh`
				// look for useless localhost mosh connections otherwise connection is remote...
				var procs []*process.Process
				var createdServerMSec int64
				conns, err := procTerm.Connections()
				if err != nil || len(conns) == 0 {
					goto skipMoshClient
				}
				createdServerMSec, err = procTerm.CreateTime()
				if err != nil {
					goto skipMoshClient
				}
				procs, err = process.Processes()
				if err != nil || len(procs) == 0 {
					goto skipMoshClient
				}
				for _, proc := range procs {
					createdClientMSec, err := proc.CreateTime()
					if err != nil {
						continue
					}
					if createdServerMSec-createdClientMSec > moshServerCreationMaxDur.Milliseconds() {
						continue
					}
					exe, err := proc.Exe()
					if err != nil {
						continue
					}
					exe = filepath.Base(exe)
					if runtime.GOOS == `windows` {
						exe, _ = strings.CutSuffix(exe, `.exe`)
					}
					if exe != `mosh-client` {
						continue
					}
					// example: mosh-client -# localhost | <127.0.0.1|::1> 60001
					cmdLine, err := proc.Cmdline()
					if err != nil {
						continue
					}
					cmdLineParts := strings.SplitN(cmdLine, ` | `, 2)
					if len(cmdLineParts) != 2 {
						continue
					}
					cmdLine = cmdLineParts[1]
					cmdLineParts = strings.SplitN(cmdLine, ` `, 2)
					if len(cmdLineParts) != 2 {
						continue
					}
					for _, conn := range conns {
						if conn.Laddr.IP == cmdLineParts[0] && strconv.Itoa(int(conn.Laddr.Port)) == cmdLineParts[1] {
							procClient = proc
							goto endMoshClient
						}
					}
				}
			skipMoshClient:
				isRemote = true
			endMoshClient:
			}
		} else {
			if exe == `mosh-client` {
				termType = `mosh`
				p, err := procTerm.Parent()
				if err == nil {
					procClient = p
				}
			}
		}
		for _, varName := range []string{sshConnVarName, sshClientVarName} {
			val, ok := envTemp.LookupEnv(varName)
			if !ok {
				continue
			}
			switch strings.Split(val, ` `)[0] {
			case `127.0.0.1`, `::1`:
			default:
				isRemote = true
			}
		}
		if procClient == nil {
			procClient = procTerm
		}
	} else if tt, found := strings.CutSuffix(termTypeLast, `-server`); found {
		termType = tt + `-client`
	} else {
		switch exe {
		case `abduco`:
			termType = `abduco`
		case `dtach`:
			termType = `dtach`
			var cwd, sock string
			var conns []net.ConnectionStat
			if procInner != nil {
				ttyInner, err := procextra.TTYOfProc(procInner)
				if err != nil {
					goto skipDTachClient
				}
				if err == nil && len(ttyInner) == 0 {
					// client process
					procClient = procTerm
					goto endDTachClient
				}
			}
			conns, err = procTerm.Connections()
			if err != nil {
				goto skipDTachClient
			}
			cwd, err = procTerm.Cwd()
			if err != nil {
				goto skipDTachClient
			}
			for _, conn := range conns {
				s := filepath.Join(cwd, conn.Laddr.IP)
				if _, err := os.Stat(s); err == nil {
					sock = s
					break
				}
			}
			if len(sock) == 0 {
				goto skipDTachClient
			}
			{
				processes, err := process.Processes()
				if err != nil {
					goto skipDTachClient
				}
				for _, proc := range processes {
					if proc != nil && proc.Pid == procTerm.Pid {
						continue
					}
					// socket file name has to be passed to dtach on the command line
					// this might go very wrong...
					cmdLine, err := proc.Cmdline()
					if err != nil {
						continue
					}
					args := strings.Split(cmdLine, ` `)
					if len(args) < 2 {
						continue
					}
					var lastArgWasModeParam bool
					var sockArg string
					for _, arg := range args[1:] {
						switch arg {
						case `-a`, `-A`, `-c`, `-n`, `-N`, `-p`:
							lastArgWasModeParam = true
						default:
							if lastArgWasModeParam {
								sockArg = arg
							}
							lastArgWasModeParam = false
						}
					}
					if !filepath.IsAbs(sockArg) {
						cwd, err := proc.Cwd()
						if err != nil {
							continue
						}
						sockArg = filepath.Join(cwd, sockArg)
					}
					if len(sockArg) == 0 || sock != sockArg {
						continue
					}
					procClient = proc
					goto endDTachClient
				}
			}
		skipDTachClient:
			procClient = procTerm
		endDTachClient:
		}
	}
	if procClient == nil {
		procClient = procTerm
	}
	if procClient.Pid < 2 {
		return nil, ``, false, errors.New(`unable to find client pid`)
	}
	return procClient, termType, isRemote, nil
}
