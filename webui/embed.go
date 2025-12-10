package webui

import "embed"

// EmbeddedFS ships the built UI assets.
//
//go:embed dist/*
var EmbeddedFS embed.FS
