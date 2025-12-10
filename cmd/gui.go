//go:build gui && cgo
// +build gui,cgo

package cmd

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/fdddf/xcstrings-translator/internal/server"
	"github.com/spf13/cobra"
	webview "github.com/webview/webview_go"
)

func init() {
	guiCmd.Flags().Int("width", 1280, "window width in pixels")
	guiCmd.Flags().Int("height", 800, "window height in pixels")
	guiCmd.Flags().Bool("debug", false, "enable the WebView debug console")
	rootCmd.AddCommand(guiCmd)
}

// guiCmd launches the desktop GUI by embedding the web UI inside a native webview.
var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Open the native desktop UI (Windows/macOS/Linux)",
	RunE: func(cmd *cobra.Command, args []string) error {
		width, err := cmd.Flags().GetInt("width")
		if err != nil {
			return err
		}
		height, err := cmd.Flags().GetInt("height")
		if err != nil {
			return err
		}
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return err
		}

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("failed to bind local port: %w", err)
		}

		app, errCh, err := server.ServeWithListener(ln)
		if err != nil {
			return err
		}

		w := webview.New(debug)
		if w == nil {
			return fmt.Errorf("failed to initialise webview window")
		}
		defer w.Destroy()

		w.SetTitle("XCStrings Translator")
		w.SetSize(width, height, webview.HintNone)
		w.Navigate("http://" + ln.Addr().String())

		// If the server dies unexpectedly, close the window so the user is not stuck.
		go func() {
			if err := <-errCh; err != nil {
				w.Dispatch(func() { w.Terminate() })
			}
		}()

		w.Run()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := app.ShutdownWithContext(ctx); err != nil {
			return fmt.Errorf("failed to shutdown embedded server: %w", err)
		}

		return nil
	},
}
