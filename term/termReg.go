package term

import (
	"slices"
	"strings"
)

// ResetTerminalCheckerList
func ResetTerminalCheckerList() {
	termCheckerListIsInit = false
	initTermCheckerList()
}

// AllTerminals returns all enabled registered dummy terminals
func AllTerminalCheckers() []TermChecker {
	initTermCheckerList()
	return terminalCheckers
}

func AllRegisteredTermCheckers() []TermChecker {
	l := len(terminalCheckersRegistered)
	terminalCheckersRegistered = terminalCheckersRegistered[:l:l]
	return terminalCheckersRegistered
}

var mapTermCheckers map[string]TermChecker

func RegisteredTermChecker(name string) TermChecker {
	if mapTermCheckers == nil {
		mapTermCheckers = make(map[string]TermChecker)
		for _, tm := range terminalCheckersRegistered {
			if tm == nil {
				continue
			}
			mapTermCheckers[tm.Name()] = tm
		}
	}
	tm, ok := mapTermCheckers[name]
	if ok && tm != nil {
		return tm
	}
	return nil
}

func initTermCheckerList() {
	if !termCheckerListIsInit {
		terminalCheckers = make([]TermChecker, len(terminalCheckersRegistered))
		copy(terminalCheckers, terminalCheckersRegistered)
		termCheckerListIsInit = true
	}
}

var (
	termCheckerListIsInit      bool
	terminalCheckers           []TermChecker
	terminalCheckersRegistered []TermChecker
)

// RegisterTerminal ...
func RegisterTermChecker(t TermChecker) {
	if initer, ok := t.(interface{ Init(TermChecker) }); ok {
		initer.Init(t)
	}
	terminalCheckersRegistered = append(terminalCheckersRegistered, t)
	termCheckerListIsInit = false
}

func DisableTerminal(name string) {
	initTermCheckerList()
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return
	}
	idxLocal := slices.IndexFunc(terminalCheckers, func(t TermChecker) bool { return t != nil && t.Name() == name })
	if idxLocal < 0 {
		return
	}
	terminalCheckers = slices.Delete(terminalCheckers, idxLocal, idxLocal+1)
}
