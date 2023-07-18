//go:build windows

package wndws

func RunsOnWine() bool { return wineVerProc.Find() == nil }

// https://www.winehq.org/pipermail/wine-devel/2008-September/069387.html

// TODO save version
