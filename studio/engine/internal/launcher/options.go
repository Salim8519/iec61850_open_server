package launcher

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"studio-engine/internal/pycompat"
)

// Options are the server-mode arguments of launcher/entrypoint.py.
type Options struct {
	Port        int
	NoBrowser   bool
	DataDir     string
	SelfTest    bool
	Dev         bool
	DevToken    string
	FrontendURL string
	NativeExe   string
}

// UsageError is an argparse error: the caller prints Usage and the message, exit code 2.
type UsageError struct{ Message string }

func (e *UsageError) Error() string { return e.Message }

// Usage is the first line argparse prints on errors.
const Usage = "usage: studio_server [-h] [--port PORT] [--no-browser] [--data-dir DATA_DIR] [--self-test] " +
	"[--dev] [--dev-token DEV_TOKEN] [--frontend-url FRONTEND_URL] [--native-exe NATIVE_EXE]"

var valueFlags = []string{"--port", "--data-dir", "--dev-token", "--frontend-url", "--native-exe"}
var switchFlags = []string{"--no-browser", "--self-test", "--dev", "--help"}

// ParseOptions reads args as argparse did (also "--flag=value" and unambiguous
// abbreviations); defaults come from STUDIO_DEV_TOKEN and STUDIO_FRONTEND_URL.
func ParseOptions(args []string) (Options, error) {
	options := Options{Port: 8765, DevToken: envOr("STUDIO_DEV_TOKEN", "dev"),
		FrontendURL: envOr("STUDIO_FRONTEND_URL", "http://127.0.0.1:5173")}
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "-h" {
			argument = "--help"
		}
		if !strings.HasPrefix(argument, "--") {
			return options, &UsageError{"unrecognized arguments: " + strings.Join(args[index:], " ")}
		}
		name, value, inline := strings.Cut(argument, "=")
		flag, err := resolveFlag(name)
		if err != nil {
			return options, err
		}
		if contains(switchFlags, flag) {
			if inline {
				return options, &UsageError{fmt.Sprintf("argument %s: ignored explicit argument '%s'", flag, value)}
			}
			switch flag {
			case "--no-browser":
				options.NoBrowser = true
			case "--self-test":
				options.SelfTest = true
			case "--dev":
				options.Dev = true
			case "--help":
				return options, &UsageError{"help"}
			}
			continue
		}
		if !inline {
			if index+1 >= len(args) || strings.HasPrefix(args[index+1], "--") {
				return options, &UsageError{fmt.Sprintf("argument %s: expected one argument", flag)}
			}
			index++
			value = args[index]
		}
		if err := options.set(flag, value); err != nil {
			return options, err
		}
	}
	return options, nil
}

func (o *Options) set(flag, value string) error {
	switch flag {
	case "--port":
		number, err := pycompat.ParseInt(value)
		port, isSmall := number.(int64)
		if err != nil || !isSmall {
			return &UsageError{fmt.Sprintf("argument --port: invalid int value: %s", pycompat.Repr(value))}
		}
		o.Port = int(port)
	case "--data-dir":
		o.DataDir = value
	case "--dev-token":
		o.DevToken = value
	case "--frontend-url":
		o.FrontendURL = value
	case "--native-exe":
		o.NativeExe = value
	}
	return nil
}

// resolveFlag accepts a full flag name or a unique prefix of one.
func resolveFlag(name string) (string, error) {
	all := append(append([]string(nil), valueFlags...), switchFlags...)
	if contains(all, name) {
		return name, nil
	}
	var matches []string
	for _, flag := range all {
		if strings.HasPrefix(flag, name) {
			matches = append(matches, flag)
		}
	}
	sort.Strings(matches)
	switch len(matches) {
	case 0:
		return "", &UsageError{"unrecognized arguments: " + name}
	case 1:
		return matches[0], nil
	}
	return "", &UsageError{fmt.Sprintf("ambiguous option: %s could match %s", name, strings.Join(matches, ", "))}
}

func contains(list []string, item string) bool {
	for _, entry := range list {
		if entry == item {
			return true
		}
	}
	return false
}

func envOr(name, fallback string) string {
	if value, found := os.LookupEnv(name); found {
		return value
	}
	return fallback
}
