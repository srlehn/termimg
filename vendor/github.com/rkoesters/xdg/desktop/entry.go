package desktop

import (
	"errors"
	"github.com/rkoesters/xdg/keyfile"
	"io"
	"strings"
)

var (
	// ErrMissingType means that the desktop entry is missing the
	// Type key, which is always required.
	ErrMissingType = errors.New("missing entry type")

	// ErrMissingName means that the desktop entry is missing the
	// Name key, which is required by the types Application, Link,
	// and Directory.
	ErrMissingName = errors.New("missing entry name")

	// ErrMissingURL means that the desktop entry is missing the URL
	// key, which is required by the type Link.
	ErrMissingURL = errors.New("missing entry url")
)

const (
	groupDesktopEntry        = "Desktop Entry"
	groupDesktopActionPrefix = "Desktop Action "

	keyType            = "Type"
	keyVersion         = "Version"
	keyName            = "Name"
	keyGenericName     = "GenericName"
	keyNoDisplay       = "NoDisplay"
	keyComment         = "Comment"
	keyIcon            = "Icon"
	keyHidden          = "Hidden"
	keyOnlyShowIn      = "OnlyShowIn"
	keyNotShowIn       = "NotShowIn"
	keyDBusActivatable = "DBusActivatable"
	keyTryExec         = "TryExec"
	keyExec            = "Exec"
	keyPath            = "Path"
	keyTerminal        = "Terminal"
	keyActions         = "Actions"
	keyMimeType        = "MimeType"
	keyCategories      = "Categories"
	keyImplements      = "Implements"
	keyKeywords        = "Keywords"
	keyStartupNotify   = "StartupNotify"
	keyStartupWMClass  = "StartupWMClass"
	keyURL             = "URL"
)

// Entry represents a desktop entry file.
type Entry struct {
	// The type of desktop entry. It can be: Application, Link, or
	// Directory.
	Type Type
	// The version of spec that the file conforms to.
	Version string

	// The real name of the desktop entry.
	Name string
	// A generic name, for example: Text Editor or Web Browser.
	GenericName string
	// A short comment that describes the desktop entry.
	Comment string
	// The name of an icon that should be used for this desktop
	// entry.  If it is not an absolute path, it should be searched
	// for using the Icon Theme Specification.
	Icon string
	// The URL for a Link type entry.
	URL string

	// Whether or not to display the file in menus.
	NoDisplay bool
	// Whether the use has deleted the desktop entry.
	Hidden bool
	// A list of desktop environments that the desktop entry should
	// only be shown in.
	OnlyShowIn []string
	// A list of desktop environments that the desktop entry should
	// not be shown in.
	NotShowIn []string

	// Whether DBus Activation is supported by this application.
	DBusActivatable bool
	// The path to an executable to test if the program is
	// installed.
	TryExec string
	// Program to execute.
	Exec string
	// The path that should be the programs working directory.
	Path string
	// Whether the program should be run in a terminal window.
	Terminal bool

	// A slice of actions.
	Actions []*Action
	// A slice of mimetypes supported by this program.
	MimeType []string
	// A slice of categories that the desktop entry should be shown
	// in in a menu.
	Categories []string
	// A slice of interfaces that this application implements.
	Implements []string
	// A slice of keywords.
	Keywords []string

	// Whether the program will send a "remove" message when started
	// with the DESKTOP_STARTUP_ID env variable is set.
	StartupNotify bool
	// The string that the program will set as WM Class or WM name
	// hint.
	StartupWMClass string

	// Extended pairs. These are all of the key=value pairs in which
	// the key follows the format X-PRODUCT-KEY. For example,
	// accessing X-Unity-IconBackgroundColor can be done with:
	//
	//	entry.X["Unity"]["IconBackgroundColor"]
	//
	X map[string]map[string]string
}

// New reads a desktop file from r and returns an Entry that represents
// the desktop file using the default locale.
func New(r io.Reader) (*Entry, error) {
	return NewWithLocale(r, keyfile.DefaultLocale())
}

// NewWithLocale reads a desktop file from r and returns an Entry that
// represents the desktop file using the given locale l.
func NewWithLocale(r io.Reader, l keyfile.Locale) (*Entry, error) {
	kf, err := keyfile.New(r)
	if err != nil {
		return nil, err
	}

	// Create the entry.
	e := new(Entry)

	e.Type = ParseType(kf.Value(groupDesktopEntry, keyType))
	if kf.KeyExists(groupDesktopEntry, keyVersion) {
		e.Version, err = kf.String(groupDesktopEntry, keyVersion)
		if err != nil {
			return nil, err
		}
	}
	e.Name, err = kf.LocaleStringWithLocale(groupDesktopEntry, keyName, l)
	if err != nil {
		return nil, err
	}
	if kf.KeyExists(groupDesktopEntry, keyGenericName) {
		e.GenericName, err = kf.LocaleStringWithLocale(groupDesktopEntry, keyGenericName, l)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyNoDisplay) {
		e.NoDisplay, err = kf.Bool(groupDesktopEntry, keyNoDisplay)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyComment) {
		e.Comment, err = kf.LocaleStringWithLocale(groupDesktopEntry, keyComment, l)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyIcon) {
		e.Icon, err = kf.LocaleStringWithLocale(groupDesktopEntry, keyIcon, l)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyHidden) {
		e.Hidden, err = kf.Bool(groupDesktopEntry, keyHidden)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyOnlyShowIn) {
		e.OnlyShowIn, err = kf.StringList(groupDesktopEntry, keyOnlyShowIn)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyNotShowIn) {
		e.NotShowIn, err = kf.StringList(groupDesktopEntry, keyNotShowIn)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyDBusActivatable) {
		e.DBusActivatable, err = kf.Bool(groupDesktopEntry, keyDBusActivatable)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyTryExec) {
		e.TryExec, err = kf.String(groupDesktopEntry, keyTryExec)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyExec) {
		e.Exec, err = kf.String(groupDesktopEntry, keyExec)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyPath) {
		e.Path, err = kf.String(groupDesktopEntry, keyPath)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyTerminal) {
		e.Terminal, err = kf.Bool(groupDesktopEntry, keyTerminal)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyActions) {
		e.Actions, err = getActions(kf)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyMimeType) {
		e.MimeType, err = kf.StringList(groupDesktopEntry, keyMimeType)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyCategories) {
		e.Categories, err = kf.StringList(groupDesktopEntry, keyCategories)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyImplements) {
		e.Implements, err = kf.StringList(groupDesktopEntry, keyImplements)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyKeywords) {
		e.Keywords, err = kf.LocaleStringListWithLocale(groupDesktopEntry, keyKeywords, l)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyStartupNotify) {
		e.StartupNotify, err = kf.Bool(groupDesktopEntry, keyStartupNotify)
		if err != nil {
			return nil, err
		}
	}
	if kf.KeyExists(groupDesktopEntry, keyStartupWMClass) {
		e.StartupWMClass, err = kf.String(groupDesktopEntry, keyStartupWMClass)
		if err != nil {
			return nil, err
		}
	}
	if e.Type == Link {
		e.URL, err = kf.String(groupDesktopEntry, keyURL)
		if err != nil {
			return nil, err
		}
	}

	// Validate the entry.
	if e.Type == None {
		return nil, ErrMissingType
	}
	if e.Type > None && e.Type < Unknown && e.Name == "" {
		return nil, ErrMissingName
	}
	if e.Type == Link && e.URL == "" {
		return nil, ErrMissingURL
	}

	// Search for extended keys.
	e.X = make(map[string]map[string]string)
	for _, k := range kf.Keys(groupDesktopEntry) {
		a := strings.SplitN(k, "-", 3)
		if a[0] != "X" || len(a) < 3 {
			continue
		}
		if e.X[a[1]] == nil {
			e.X[a[1]] = make(map[string]string)
		}
		e.X[a[1]][a[2]] = kf.Value(groupDesktopEntry, k)
	}

	return e, nil
}

// Action is an Action group.
type Action struct {
	Name string
	Icon string
	Exec string
}

func getActions(kf *keyfile.KeyFile) ([]*Action, error) {
	var acts []*Action
	var act *Action
	var err error
	var list []string

	list, err = kf.StringList(groupDesktopEntry, keyActions)
	if err != nil {
		return nil, err
	}
	for _, a := range list {
		g := groupDesktopActionPrefix + a

		act = new(Action)

		act.Name, err = kf.String(g, keyName)
		if err != nil {
			return nil, err
		}
		act.Icon, err = kf.String(g, keyIcon)
		if err != nil {
			return nil, err
		}
		act.Exec, err = kf.String(g, keyExec)
		if err != nil {
			return nil, err
		}

		acts = append(acts, act)
	}
	return acts, nil
}
