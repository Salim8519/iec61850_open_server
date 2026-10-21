//go:build release

package main

import (
	"embed"
	"io/fs"
)

// The release build (compile_go_release.py) copies the finished ScadaBench folder and its
// payload.key into ./payload before building with -tags release.
//
//go:embed all:payload
var embedded embed.FS

var payload fs.FS = mustSub(embedded, "payload")

func mustSub(files embed.FS, folder string) fs.FS {
	sub, err := fs.Sub(files, folder)
	if err != nil {
		panic(err)
	}
	return sub
}
