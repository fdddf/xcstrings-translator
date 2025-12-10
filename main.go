//go:build !windows || !gui || !cgo
// +build !windows !gui !cgo

package main

import "github.com/fdddf/xcstrings-translator/cmd"

func main() {
	cmd.Execute()
}
