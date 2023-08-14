package propkeys

const (
	QueryCachePrefix            = `queryCache_`
	CheckPrefix                 = `check`
	CheckTermPrefix             = CheckPrefix + `Term`
	CheckTermEnvExclPrefix      = CheckTermPrefix + `EnvExclude_`
	CheckTermQueryIsPrefix      = CheckTermPrefix + `EnvIs_`
	CheckTermWindowIsPrefix     = CheckTermPrefix + `WindowIs_`
	CheckTermCompletePrefix     = CheckTermPrefix + `Complete_`
	EnvPrefix                   = `env_`
	PreservedOuterEnvPrefix     = GeneralPrefix + `envOuterPreserved_`
	GeneralPrefix               = `general_`
	EnvIsLoaded                 = GeneralPrefix + `envIsLoaded`
	Mode                        = GeneralPrefix + `mode` /// "tui", "cli" (default)
	ManualComposition           = GeneralPrefix + `manualComposition`
	TerminalPID                 = GeneralPrefix + `termPID`
	TerminalTTY                 = GeneralPrefix + `termTTY` // directly provided tty by the terminal
	Passages                    = GeneralPrefix + `passages`
	IsRemote                    = GeneralPrefix + `isRemote`
	AvoidANSI                   = GeneralPrefix + `avoidANSI`
	AvoidDA1                    = GeneralPrefix + `avoidDA1`
	AvoidDA2                    = GeneralPrefix + `avoidDA2`
	AvoidDA3                    = GeneralPrefix + `avoidDA3`
	DeviceAttributesWereQueried = GeneralPrefix + `deviceAttributesWereQueried`
	DeviceAttributes            = GeneralPrefix + `deviceAttributes`
	DeviceClass                 = GeneralPrefix + `deviceClass`
	ReGISCapable                = GeneralPrefix + `regisCapable`
	SixelCapable                = GeneralPrefix + `sixelCapable`
	WindowingCapable            = GeneralPrefix + `windowingCapable`
	DA3ID                       = GeneralPrefix + `DA3ID`
	XTGETTCAPPrefix             = GeneralPrefix + `XTGETTCAP_`
	XTGETTCAPKeyNamePrefix      = XTGETTCAPPrefix + `keyName_`
	XTGETTCAPSpecialPrefix      = XTGETTCAPPrefix + `special_`
	XTGETTCAPSpecialTN          = XTGETTCAPSpecialPrefix + `TN`
	XTGETTCAPSpecialCo          = XTGETTCAPSpecialPrefix + `Co`
	XTGETTCAPSpecialRGB         = XTGETTCAPSpecialPrefix + `RGB`
	XTGETTCAPInvalidPrefix      = XTGETTCAPPrefix + `invalid_`
	// TODO positioning of uncropped image is dependent on terminal size (e.g. urxvt)
	FullImgPosDepOnSize = GeneralPrefix + `fullImagePositionDependentOnSize`

	SystemD    = `platform_systemd`
	RunsOnWine = `platform_wine`

	// Terminal type properties
	AppleTermVersion      = `apple_version` // CFBundleVersion of Terminal.app
	ContourVersion        = `contour_version`
	DomTermPrefix         = `domterm_`
	DomTermLibWebSockets  = DomTermPrefix + `libwebsockets`
	DomTermSession        = DomTermPrefix + `session`
	DomTermTTY            = DomTermPrefix + `tty`
	DomTermPID            = DomTermPrefix + `pid`
	DomTermVersion        = DomTermPrefix + `version`
	DomTermWindowName     = DomTermPrefix + `windowName`
	DomTermWindowInstance = DomTermPrefix + `windowInstance`
	ITerm2Version         = `iterm2_version`
	KittyWindowID         = `kitty_windowID`      // tab id
	MacTermBuildNr        = `macterm_buildNumber` // YYYYMMDD
	MltermVersion         = `mlterm_version`
	MinttyPrefix          = `mintty_`
	MinttyShortcut        = MinttyPrefix + `shortcut`
	MinttyVersion         = MinttyPrefix + `version`
	URXVTPrefix           = `urxvt_`
	URXVTExeName          = URXVTPrefix + `executableName`
	URXVTVerFirstChar     = URXVTPrefix + `versionFirstChar`
	URXVTVerThirdChar     = URXVTPrefix + `versionThirdChar`
	VSCodePrefix          = `vscode_`
	VSCodeVersion         = VSCodePrefix + `version`
	VSCodeVersionMajor    = VSCodeVersion + `Major`
	VSCodeVersionMinor    = VSCodeVersion + `Minor`
	VSCodeVersionPatch    = VSCodeVersion + `Patch`
	VTEPrefix             = `vte_`
	VTEVersion            = VTEPrefix + `version`
	VTEVersionMajor       = VTEVersion + `Major`
	VTEVersionMinor       = VTEVersion + `Minor`
	VTEVersionPatch       = VTEVersion + `Patch`
	WezTermPrefix         = `wezterm_`
	WezTermExe            = WezTermPrefix + `executable`
	WezTermExeDir         = WezTermPrefix + `executableDir`
	WezTermPane           = WezTermPrefix + `pane`
	WezTermUnixSocket     = WezTermPrefix + `unixSocket`
	XTermVersion          = `xterm_version`
)
