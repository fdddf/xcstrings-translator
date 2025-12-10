//go:build windows && gui && cgo
// +build windows,gui,cgo

package main

import "github.com/fdddf/xcstrings-translator/cmd"

// Entry point for Windows GUI builds so double-click launches the native window.
func main() {
	cmd.ExecuteGUI()
}
