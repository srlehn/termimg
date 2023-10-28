package pty

import (
	"golang.org/x/crypto/ssh"
)

// ApplyTerminalModes applies the given ssh terminal modes to the given file
// descriptor.
func ApplyTerminalModes(fd int, width int, height int, modes ssh.TerminalModes) error {
	if modes == nil {
		return nil
	}
	return applyTerminalModesToFd(fd, width, height, modes)
}

// terminalModeFlagNames maps the SSH terminal mode flags to mnemonic
// names used by the termios package.
var terminalModeFlagNames = map[uint8]string{
	ssh.VINTR:         "intr",
	ssh.VQUIT:         "quit",
	ssh.VERASE:        "erase",
	ssh.VKILL:         "kill",
	ssh.VEOF:          "eof",
	ssh.VEOL:          "eol",
	ssh.VEOL2:         "eol2",
	ssh.VSTART:        "start",
	ssh.VSTOP:         "stop",
	ssh.VSUSP:         "susp",
	ssh.VDSUSP:        "dsusp",
	ssh.VREPRINT:      "rprnt",
	ssh.VWERASE:       "werase",
	ssh.VLNEXT:        "lnext",
	ssh.VFLUSH:        "flush",
	ssh.VSWTCH:        "swtch",
	ssh.VSTATUS:       "status",
	ssh.VDISCARD:      "discard",
	ssh.IGNPAR:        "ignpar",
	ssh.PARMRK:        "parmrk",
	ssh.INPCK:         "inpck",
	ssh.ISTRIP:        "istrip",
	ssh.INLCR:         "inlcr",
	ssh.IGNCR:         "igncr",
	ssh.ICRNL:         "icrnl",
	ssh.IUCLC:         "iuclc",
	ssh.IXON:          "ixon",
	ssh.IXANY:         "ixany",
	ssh.IXOFF:         "ixoff",
	ssh.IMAXBEL:       "imaxbel",
	ssh.IUTF8:         "iutf8",
	ssh.ISIG:          "isig",
	ssh.ICANON:        "icanon",
	ssh.XCASE:         "xcase",
	ssh.ECHO:          "echo",
	ssh.ECHOE:         "echoe",
	ssh.ECHOK:         "echok",
	ssh.ECHONL:        "echonl",
	ssh.NOFLSH:        "noflsh",
	ssh.TOSTOP:        "tostop",
	ssh.IEXTEN:        "iexten",
	ssh.ECHOCTL:       "echoctl",
	ssh.ECHOKE:        "echoke",
	ssh.PENDIN:        "pendin",
	ssh.OPOST:         "opost",
	ssh.OLCUC:         "olcuc",
	ssh.ONLCR:         "onlcr",
	ssh.OCRNL:         "ocrnl",
	ssh.ONOCR:         "onocr",
	ssh.ONLRET:        "onlret",
	ssh.CS7:           "cs7",
	ssh.CS8:           "cs8",
	ssh.PARENB:        "parenb",
	ssh.PARODD:        "parodd",
	ssh.TTY_OP_ISPEED: "tty_op_ispeed",
	ssh.TTY_OP_OSPEED: "tty_op_ospeed",
}
