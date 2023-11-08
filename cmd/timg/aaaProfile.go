//go:build dev

package main

import (
	"os"
	"runtime/pprof"
)

func init() {
	rootCmd.Flags().StringVarP(&cpuProfileFlag, `cpuprofile`, `p`, ``, `write cpu profile to file`)
	cpuProfilefunc = profileFunc
}

func profileFunc(profileFile string) func() {
	f, err := os.Create(profileFile)
	if err != nil {
		// TODO
		return nil
	}
	pprof.StartCPUProfile(f)
	cleanUp := func() {
		pprof.StopCPUProfile()
		_ = f.Close()
	}
	return cleanUp
}
