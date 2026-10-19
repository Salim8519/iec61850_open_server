package platformapi

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/licensing"
	"studio-engine/internal/pycompat"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/updates"
)

// answer is a handler's own error answer: `return jsonify(...), status`.
type answer struct {
	status int
	body   *pyjson.Object
}

func (a *answer) Error() string { return fmt.Sprintf("HTTP %d", a.status) }

func writeJSON(w http.ResponseWriter, status int, value any) { httpapi.WriteJSON(w, status, value) }

// writeError answers an *answer as given and anything else as app_factory.py's handlers
// did (httpapi.WriteError: StudioError 400/409, ValueError/KeyError 400, RuntimeError 503,
// Werkzeug errors, else 500).
func writeError(w http.ResponseWriter, err error) {
	var own *answer
	if errors.As(err, &own) {
		writeJSON(w, own.status, own.body)
		return
	}
	httpapi.WriteError(w, err)
}

// run calls h; a panic is the unexpected exception Flask answered with 500.
func run(h handler, r *http.Request) (value any, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			value, err = nil, fmt.Errorf("panic: %v", failure)
		}
	}()
	return h(r)
}

// licenseRun is api/license._run: license-server, action and certificate errors answer
// {"error", "code"} with 503/409/400; anything else reaches the app's handlers.
func (a *Area) licenseRun(value *pyjson.Object, err error) (any, error) {
	if err == nil {
		return value, nil
	}
	if status, body, ok := licensing.ErrorResponse(err); ok {
		return nil, &answer{status, body}
	}
	return nil, a.uncaught(err)
}

// configFailure is api/updates.configure's `except ValueError`: 400 {"error", "code"}.
func configFailure(err error) error {
	var config *updates.ConfigError
	if errors.As(err, &config) {
		return &answer{http.StatusBadRequest, config.Body()}
	}
	return uncaughtKeyError(err)
}

// keyErrorText is str(KeyError(name)) for the plain names the ported services report.
var keyErrorText = regexp.MustCompile(`^'[A-Za-z_][A-Za-z0-9_]*'$`)

// uncaughtKeyError turns a licensing.UncaughtError that stands for a Python KeyError into
// the 400 {"error": "'key'"} answer of app_factory's KeyError handler (the licensing and
// updates generators ran without that handler, so their error type has no kind; see the
// stream log). Other uncaught errors stay 500.
func uncaughtKeyError(err error) error {
	var uncaught *licensing.UncaughtError
	if errors.As(err, &uncaught) && keyErrorText.MatchString(uncaught.Reason) {
		return apperr.ValueError(uncaught.Reason)
	}
	return err
}

// uncaught maps license errors no route catches: a KeyError as above, and a license.cert
// that is not UTF-8, which Python's read_text raised as UnicodeDecodeError (a ValueError:
// 400 with CPython's exact text, rebuilt from the file).
func (a *Area) uncaught(err error) error {
	var uncaught *licensing.UncaughtError
	if errors.As(err, &uncaught) && uncaught.Reason == "license.cert is not UTF-8" {
		if data, readErr := os.ReadFile(a.deps.License.CertFile()); readErr == nil {
			if _, decodeErr := pycompat.DecodeUTF8(data); decodeErr != nil {
				return decodeErr // an apperr ValueError with CPython's text
			}
		}
	}
	return uncaughtKeyError(err)
}
