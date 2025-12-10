package cmd

import (
	"fmt"

	"github.com/fdddf/xcstrings-translator/internal/server"
	"github.com/spf13/cobra"
)

// serveCmd launches the Fiber web UI server.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI for visual localisation",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		fmt.Printf("Serving web UI on %s\n", addr)
		return server.Serve(addr)
	},
}

func init() {
	serveCmd.Flags().String("addr", ":8080", "listen address for the web UI")
	rootCmd.AddCommand(serveCmd)
}
