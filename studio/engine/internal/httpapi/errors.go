package httpapi

import (
	"errors"
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Werkzeug descriptions the UI can show (app_factory.http_error answers {"error": text}).
const (
	TextBadRequest      = "The browser (or proxy) sent a request that this server could not understand."
	TextNotFound        = "The requested URL was not found on the server. If you entered the URL manually please check your spelling and try again."
	TextMethodNotAllow  = "The method is not allowed for the requested URL."
	TextTooLarge        = "The data value transmitted exceeds the capacity limit."
	TextUnsupportedJSON = "Did not attempt to load JSON data because the request Content-Type was not 'application/json'."
	TextInternal        = "The server encountered an internal error and was unable to complete your request. Either the server is overloaded or there is an error in the application."
)

// HTTPError is a Werkzeug HTTPException: Flask answers {"error": Description} with Code.
type HTTPError struct {
	Code        int
	Description string
}

func (e *HTTPError) Error() string { return e.Description }

// Werkzeug exceptions handlers raise.
var (
	ErrBadRequest       = &HTTPError{http.StatusBadRequest, TextBadRequest}
	ErrNotFound         = &HTTPError{http.StatusNotFound, TextNotFound}
	ErrMethodNotAllowed = &HTTPError{http.StatusMethodNotAllowed, TextMethodNotAllow}
	ErrTooLarge         = &HTTPError{http.StatusRequestEntityTooLarge, TextTooLarge}
	ErrUnsupportedJSON  = &HTTPError{http.StatusUnsupportedMediaType, TextUnsupportedJSON}
)

// WriteError answers err exactly as the Python app's error handlers did:
//   - StudioError: 409 for code "conflict", else 400, {"code", "details", "error"}
//     (falsy details become []);
//   - ValueError and KeyError: 400 {"error"}; RuntimeError: 503 {"error"};
//   - HTTPError: its code with {"error": description};
//   - anything else: 500 with Werkzeug's InternalServerError text.
func WriteError(w http.ResponseWriter, err error) {
	var studio *apperr.Error
	var exception *HTTPError
	switch {
	case errors.As(err, &studio):
		writeAppError(w, studio)
	case errors.As(err, &exception):
		writeMessage(w, exception.Code, exception.Description)
	default:
		writeInternalError(w)
	}
}

func writeAppError(w http.ResponseWriter, e *apperr.Error) {
	switch e.Kind {
	case apperr.Studio:
		status := http.StatusBadRequest
		if e.Code == "conflict" {
			status = http.StatusConflict
		}
		details := e.Details
		if texts, isTexts := details.([]string); isTexts {
			list := make([]any, len(texts))
			for index, text := range texts {
				list[index] = text
			}
			details = list
		}
		if !truthy(details) {
			details = []any{}
		}
		WriteJSON(w, status, pyjson.NewObject().Set("error", e.Message).Set("code", e.Code).Set("details", details))
	case apperr.Value, apperr.Key:
		writeMessage(w, http.StatusBadRequest, e.Message)
	case apperr.Runtime:
		writeMessage(w, http.StatusServiceUnavailable, e.Message)
	default:
		writeInternalError(w)
	}
}

func writeInternalError(w http.ResponseWriter) {
	body, _ := pyjson.Flask(errorBody(TextInternal))
	WriteBody(w, http.StatusInternalServerError, "application/json", body)
}

// truthy is Python's bool(value) for pyjson values.
func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case []string:
		return len(v) > 0
	case *pyjson.Object:
		return v != nil && v.Len() > 0
	}
	return true
}
