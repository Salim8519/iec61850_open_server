package netcfg

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// pyMatch is re.match(pattern, text) for a ^...$ pattern: Python's $ also matches before one
// final "\n".
func pyMatch(pattern *regexp.Regexp, text string) bool {
	if pattern.MatchString(text) {
		return true
	}
	return strings.HasSuffix(text, "\n") && pattern.MatchString(text[:len(text)-1])
}

// resolvePath is Path(text).resolve() for comparison: absolute, cleaned, links of the
// existing part followed, and lower-case because Windows paths compare without case.
func resolvePath(text string) string {
	absolute, err := filepath.Abs(text)
	if err != nil {
		absolute = filepath.Clean(text)
	}
	existing, rest := absolute, ""
	for {
		if _, err := os.Lstat(existing); err == nil {
			break
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			existing, rest = absolute, ""
			break
		}
		rest = filepath.Join(filepath.Base(existing), rest)
		existing = parent
	}
	if resolved, err := filepath.EvalSymlinks(existing); err == nil {
		existing = resolved
	}
	return strings.ToLower(filepath.Join(existing, rest))
}

// samePath is Path(a).resolve() == Path(b).resolve().
func samePath(a, b string) bool { return resolvePath(a) == resolvePath(b) }

// gatewayWorker is Path(program).with_name("studio_gateway.exe").
func gatewayWorker(program string) string {
	return filepath.Join(filepath.Dir(program), "studio_gateway.exe")
}

// withSuffix is Path.with_suffix(suffix): the last ".ext" of the name is replaced.
func withSuffix(path, suffix string) string {
	name := filepath.Base(path)
	if dot := strings.LastIndex(name, "."); dot > 0 && dot < len(name)-1 {
		return path[:len(path)-len(name)+dot] + suffix
	}
	return path + suffix
}
