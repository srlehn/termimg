//go:build dev

package main

func init() {
	showCmd.PersistentFlags().BoolVarP(&showCoords, `coords`, `n`, false, `show cell coordinates`)
	showCmd.PersistentFlags().BoolVarP(&showGrid, `grid`, `g`, false, `show cell grid (chess pattern)`)
}
