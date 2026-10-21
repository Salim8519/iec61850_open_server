package pytext

import (
	"errors"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// Path returns str(Path(base, *parts)) as pathlib prints it on Windows: "/" becomes "\",
// repeated separators and "." parts disappear, ".." stays, no trailing separator. A
// relative base stays relative, exactly as Python keeps it.
func Path(base string, parts ...string) string {
	joined := strings.ReplaceAll(strings.Join(append([]string{base}, parts...), "\\"), "/", "\\")
	prefix := ""
	switch {
	case strings.HasPrefix(joined, "\\\\"): // UNC \\server\share
		rest := strings.TrimLeft(joined, "\\")
		pieces := strings.SplitN(rest, "\\", 3)
		if len(pieces) >= 2 {
			prefix = "\\\\" + pieces[0] + "\\" + pieces[1] + "\\"
			joined = ""
			if len(pieces) == 3 {
				joined = pieces[2]
			}
		}
	case len(joined) >= 2 && joined[1] == ':':
		prefix, joined = joined[:2], joined[2:]
		if strings.HasPrefix(joined, "\\") {
			prefix += "\\"
		}
	case strings.HasPrefix(joined, "\\"):
		prefix = "\\"
	}
	var kept []string
	for _, piece := range strings.Split(joined, "\\") {
		if piece != "" && piece != "." {
			kept = append(kept, piece)
		}
	}
	if prefix == "" && len(kept) == 0 {
		return "."
	}
	return prefix + strings.Join(kept, "\\")
}

// OSError returns str(error) for an OSError Python would raise for the same failure.
// errnoStyle selects the C runtime form used by open()/read_bytes()/shutil.copyfile
// ("[Errno 2] No such file or directory: 'C:\\x'"); otherwise the Win32 form used by
// os.mkdir/os.replace ("[WinError 5] Access is denied: 'C:\\x'"). Other errors keep their
// Go text.
func OSError(err error, errnoStyle bool) string {
	var pathErr *fs.PathError
	var linkErr *os.LinkError
	var code syscall.Errno
	if !errors.As(err, &code) {
		return err.Error()
	}
	names := ""
	switch {
	case errors.As(err, &pathErr):
		names = ": " + StrRepr(Path(pathErr.Path))
	case errors.As(err, &linkErr):
		names = ": " + StrRepr(Path(linkErr.Old)) + " -> " + StrRepr(Path(linkErr.New))
	}
	if errnoStyle {
		if number, text, ok := crtErrno(uintptr(code)); ok {
			return "[Errno " + strconv.Itoa(number) + "] " + text + names
		}
	}
	message := strings.TrimRight(code.Error(), " .\r\n\t")
	return "[WinError " + strconv.Itoa(int(code)) + "] " + message + names
}

// crtErrno maps Win32 error codes as the C runtime's _dosmaperr does for the common cases.
func crtErrno(code uintptr) (int, string, bool) {
	switch code {
	case 2, 3, 15, 18, 53, 161: // file/path/drive not found
		return 2, "No such file or directory", true
	case 5, 19, 32, 33, 65, 82, 108: // access denied, sharing and lock violations
		return 13, "Permission denied", true
	case 80, 183: // file exists
		return 17, "File exists", true
	case 112: // disk full
		return 28, "No space left on device", true
	}
	return 0, "", false
}
