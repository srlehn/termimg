package queries

const (
	// https://terminalguide.namepad.de/seq/csi_sc/
	DA1 = CSI + `0c`
	// https://terminalguide.namepad.de/seq/csi_sc__q/
	DA2 = CSI + `>0c`
	// https://terminalguide.namepad.de/seq/csi_sc__r/
	DA3 = CSI + `=0c`
	// https://invisible-island.net/xterm/terminfo-contents.html#tic-_Report_xterm_name_and_version__X_T_V_E_R_S_I_O_N_
	XTVERSION = CSI + `>0q`
	// https://iterm2.com/documentation-escape-codes.html
	ITerm2CellSize = OSC + `1337;ReportCellSize` + ST
	// https://iterm2.com/utilities/it2check
	// https://github.com/mintty/mintty/issues/881#issuecomment-614601911
	// n-terminated, implemented by iterm2, tmux
	ITerm2PropVersion = CSI + `1337n`
	// https://sw.kovidgoyal.net/kitty/graphics-protocol/#querying-support-and-available-transmission-mediums
	KittyTest = APC + `Ga=q,i=1,t=d;Cg==` + ST
	// DECSCNM - https://vt100.net/docs/vt510-rm/DECSCNM.html
	TerminologySize       = "\033}qs" + TerminologyTerminator // https://github.com/borisfaure/terminology#extended-escapes-for-terminology-only
	TerminologyTerminator = "\000"
	Foreground            = OSC + `10;?` + ST
	Background            = OSC + `11;?` + ST
	ReverseVideo          = CSI + `?5$p`
)

const (
	SP = ` ` // https://vt100.net/docs/vt510-rm/chapter4.html - 4.3.1 SP = space

	// https://en.wikipedia.org/wiki/ANSIescapecode
	// Popular C0 control codes (not an exhaustive list)
	BEL = "\x07" // Bell
	BS  = "\x08" // Backspace
	HT  = "\x09" // Tab
	LF  = "\x0A" // Line Feed
	FF  = "\x0C" // Form Feed
	CR  = "\x0D" // Carriage Return
	ESC = "\x1B" // Escape

	// Some type Fe (C1 set element) ANSI escape sequences (not an exhaustive list)
	// Moves the cursor down one line in the same column. If the cursor is at the bottom margin, the page scrolls up.
	IND = ESC + `D`  //  // Index - Moves the cursor down one line in the same column. If the cursor is at the bottom margin, the page scrolls up.
	RI  = ESC + `M`  //  // Reverse Index - Moves the cursor up one line in the same column. If the cursor is at the top margin, the page scrolls down.
	NEL = ESC + `E`  //  // Next Line - Moves the cursor to the first position on the next line. If the cursor is at the bottom margin, the page scrolls up.
	SS2 = ESC + "N"  // "\x8E" // Single Shift Two
	SS3 = ESC + "O"  // "\x8F" // Single Shift Three
	DCS = ESC + "P"  // "\x90" // Device Control String - Terminated by ST
	CSI = ESC + "["  // "\x9B" // Control Sequence Introducer
	ST  = ESC + "\\" // "\x9C" // String Terminator
	OSC = ESC + "]"  // "\x9D" // Operating System Command
	SOS = ESC + "X"  // "\x98" // Start of String - Terminated by ST
	PM  = ESC + "^"  // "\x9E" // Privacy Message - Terminated by ST
	APC = ESC + "_"  // "\x9F" // Application Program Command - Terminated by ST

	// Some popular private sequences
	SCP   = CSI + `s` // Save Current Cursor Position
	SCOSC = SCP       // Save Current Cursor Position
	RCP   = CSI + `u` // Restore Saved Cursor Position
	SCORC = RCP       // Restore Saved Cursor Position

	// Some type Fs (independent function) ANSI escape sequences recognised by terminals (not an exhaustive list)
	RIS = ESC + `c` // Reset to Initial State

	// Some type Fp (private-use) escape sequences recognised by the VT100, its successors, and/or terminal emulators such as xterm
	DECSC = ESC + `7` // DEC Save Cursor
	DECRC = ESC + `8` // DEC Restore Cursor

	// Some type 0Ft (announcement) ANSI escape sequences recognised by terminals
	ACS6  = ESC + SP + `F` // Announce Code Structure 6
	S7C1T = ACS6           // Send 7-bit C1 Control Character to the Host
	ACS7  = ESC + SP + `G` // Announce Code Structure 7
	S8C1T = ACS7           // Send 8-bit C1 Control Character to the Host
)

// https://vt100.net/docs/vt510-rm/chapter4.html
