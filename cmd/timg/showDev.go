//go:build dev

package main

func init() {
	showCmd.Flags().BoolVarP(&showCoords, `coords`, `n`, false, `show cell coordinates`)
	showCmd.Flags().BoolVarP(&showGrid, `grid`, `g`, false, `show cell grid (chess pattern)`)
}
