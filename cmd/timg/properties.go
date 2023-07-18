package main

import (
	"github.com/spf13/cobra"

	_ "github.com/srlehn/termimg/drawers"
	"github.com/srlehn/termimg/internal/testutil/routines"
	_ "github.com/srlehn/termimg/terminals"
)

func init() { rootCmd.AddCommand(listCmd) }

var listCmd = &cobra.Command{
	Use:   "properties",
	Short: "list terminal properties",
	Long:  "list terminal properties",
	Run: func(cmd *cobra.Command, args []string) {
		run(routines.ListTermProps)
	},
}
