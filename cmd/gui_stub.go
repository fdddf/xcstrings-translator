//go:build !gui || !cgo
// +build !gui !cgo

package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// guiCmd stub used when built without WebView/cgo support.
var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Open the native desktop UI (requires cgo and -tags gui)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("this binary was built without GUI support; rebuild with cgo enabled and -tags gui to enable the desktop window")
	},
}

func init() {
	rootCmd.AddCommand(guiCmd)
}
