package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Folder names beside the portable exe (ADR-0083; ADR-0063 data folder name).
const (
	dataFolder = "ScadaBench-data"
	appFolder  = "app"
	keyFile    = "payload.key" // content key written by compile_go_release.py
)

// payloadFile is one embedded file: its slash path inside the app folder and its size.
type payloadFile struct {
	name string
	size int64
}

// listPayload returns every regular file of payload except the key file, in walk order.
func listPayload(payload fs.FS) ([]payloadFile, error) {
	var files []payloadFile
	err := fs.WalkDir(payload, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || name == keyFile {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, payloadFile{name: name, size: info.Size()})
		return nil
	})
	return files, err
}

// contentKey is the build's key from payload.key: 16 lowercase hex digits.
func contentKey(payload fs.FS) (string, error) {
	data, err := fs.ReadFile(payload, keyFile)
	key := strings.TrimSpace(string(data))
	if err != nil || len(key) != 16 || strings.Trim(key, "0123456789abcdef") != "" {
		return "", errors.New("this portable exe has no ScadaBench files; build it with studio/tools/compile_go_release.py")
	}
	return key, nil
}

// complete reports whether folder holds every payload file with its exact size.
func complete(folder string, files []payloadFile) bool {
	for _, file := range files {
		info, err := os.Stat(filepath.Join(folder, filepath.FromSlash(file.name)))
		if err != nil || !info.Mode().IsRegular() || info.Size() != file.size {
			return false
		}
	}
	return len(files) > 0
}

// stage copies the payload once to <data>/app/<key> and returns that folder. A complete
// copy is reused as it is; a new copy is written under a temporary name and renamed, so an
// interrupted start never leaves a half-written app folder in use (portable_runtime.py).
func stage(payload fs.FS, data string) (string, error) {
	key, err := contentKey(payload)
	if err != nil {
		return "", err
	}
	files, err := listPayload(payload)
	if err != nil {
		return "", err
	}
	target := filepath.Join(data, appFolder, key)
	if complete(target, files) {
		return target, nil
	}
	staging := target + ".partial-" + strconv.Itoa(os.Getpid())
	_ = os.RemoveAll(staging)
	if err := copyPayload(payload, files, staging); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	_ = os.RemoveAll(target)
	if err := renameWhenFree(staging, target); err != nil {
		_ = os.RemoveAll(staging)
		if !complete(target, files) { // another start may have finished the same copy first
			return "", fmt.Errorf("ScadaBench could not prepare %s: %w", target, err)
		}
	}
	return target, nil
}

// Antivirus programs scan the freshly written programs and hold them open meanwhile, and
// Windows refuses to rename a folder with open files ("Access is denied"; seen on the
// owner's laptop with the first 0.1.15 portable). The rename is retried until they let go.
const (
	renameTries = 120
	renamePause = 250 * time.Millisecond
)

// renameWhenFree renames staging to target, retrying while Windows refuses it.
func renameWhenFree(staging, target string) error {
	var err error
	for try := 0; try < renameTries; try++ {
		if err = os.Rename(staging, target); err == nil {
			return nil
		}
		time.Sleep(renamePause)
	}
	return err
}

func copyPayload(payload fs.FS, files []payloadFile, folder string) error {
	for _, file := range files {
		destination := filepath.Join(folder, filepath.FromSlash(file.name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		if err := copyOne(payload, file.name, destination); err != nil {
			return fmt.Errorf("copy %s: %w", path.Base(file.name), err)
		}
	}
	return nil
}

func copyOne(payload fs.FS, name, destination string) error {
	source, err := payload.Open(name)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		return err
	}
	return target.Close()
}
