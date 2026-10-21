package httpapi

import (
	"crypto/subtle"
	"net/http"

	"studio-engine/internal/pyjson"
)

// SessionCookie is Flask's SESSION_COOKIE_NAME for ScadaBench.
const SessionCookie = "studio_session"

// sessionAuthenticated is session.get("authenticated") truthiness. Reading the session
// makes Flask add "Vary: Cookie" to the response, so this does too.
func (s *Server) sessionAuthenticated(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Add("Vary", "Cookie")
	cookie := cookieValue(r, SessionCookie)
	if cookie == "" {
		return false
	}
	session := s.signer.loads(cookie, s.now())
	if session == nil {
		return false
	}
	value, _ := session.Get("authenticated")
	return truthy(value)
}

// cookieValue returns the first cookie named name, as Werkzeug's request.cookies does.
func cookieValue(r *http.Request, name string) string {
	for _, cookie := range r.Cookies() {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

// establishSession is POST /api/v1/session (api/security.py establish_session).
func (s *Server) establishSession(w http.ResponseWriter, r *http.Request) {
	body, err := ReadJSON(r, true)
	if err != nil {
		WriteError(w, err)
		return
	}
	supplied, err := pyGet(orEmpty(body), "token", "")
	if err != nil {
		WriteError(w, err)
		return
	}
	text, isText := supplied.(string)
	if isText {
		if equal, err := compareDigest(text, s.config.Token); err != nil {
			WriteError(w, err)
			return
		} else if equal {
			s.grantSession(w)
			return
		}
	}
	WriteJSON(w, http.StatusUnauthorized, errorBody("Invalid launch session. Reopen the app from its launcher."))
}

func (s *Server) grantSession(w http.ResponseWriter) {
	value, err := s.signer.dumps(pyjson.NewObject().Set("authenticated", true), s.now())
	if err != nil {
		WriteError(w, err)
		return
	}
	header := w.Header()
	header.Add("Vary", "Cookie")
	header.Add("Set-Cookie", SessionCookie+"="+value+"; HttpOnly; Path=/; SameSite=Strict")
	WriteJSON(w, http.StatusOK, pyjson.NewObject().Set("ok", true))
}

// compareDigest is hmac.compare_digest for two str: constant time, and a TypeError (500)
// when either holds a non-ASCII character.
func compareDigest(a, b string) (bool, error) {
	if !isASCII(a) || !isASCII(b) {
		return false, &typeError{"comparing strings with non-ASCII characters is not supported"}
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1, nil
}

type typeError struct{ message string }

func (e *typeError) Error() string { return e.message }

func isASCII(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] >= 0x80 {
			return false
		}
	}
	return true
}
