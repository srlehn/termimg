//go:build !windows

package wndws

func RunsOnWine() bool { return false }
