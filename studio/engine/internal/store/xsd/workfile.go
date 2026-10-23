package xsd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// workFiles holds the SCL bytes of the validation in progress under
// <data>/cache/xsd/work, so the helper can read documents larger than one frame.
// Each file is removed as soon as its answer arrived; leftovers of a crashed engine
// older than staleAge are removed when the folder is first used.
type workFiles struct {
	dir     string
	cleaned bool
}

const staleAge = time.Hour

func (w *workFiles) write(source []byte) (string, error) {
	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return "", errWorkFile(err)
	}
	if !w.cleaned {
		w.cleaned = true
		w.removeStale()
	}
	file, err := os.CreateTemp(w.dir, "source-*.scl")
	if err != nil {
		return "", errWorkFile(err)
	}
	_, err = file.Write(source)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(file.Name())
		return "", errWorkFile(err)
	}
	path, err := filepath.Abs(file.Name())
	if err != nil {
		_ = os.Remove(file.Name())
		return "", errWorkFile(err)
	}
	return extendedPath(path), nil
}

// extendedPath gives the helper a Windows extended-length path (`\\?\C:\...` or
// `\\?\UNC\server\share\...`) so a data folder deeper than MAX_PATH (260) still opens: the
// helper opens the file with _wfopen, which refuses longer plain paths (change 0185). The
// path is absolute and clean (filepath.Abs). Other systems get the path unchanged.
func extendedPath(path string) string {
	if runtime.GOOS != "windows" || strings.HasPrefix(path, `\\?\`) {
		return path
	}
	if strings.HasPrefix(path, `\\`) {
		return `\\?\UNC\` + path[2:]
	}
	return `\\?\` + path
}

func (w *workFiles) remove(path string) {
	_ = os.Remove(path)
}

func (w *workFiles) removeStale() {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "source-") || !strings.HasSuffix(name, ".scl") {
			continue
		}
		if info, err := entry.Info(); err == nil && time.Since(info.ModTime()) > staleAge {
			_ = os.Remove(filepath.Join(w.dir, name))
		}
	}
}
