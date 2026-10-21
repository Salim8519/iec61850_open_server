package licensing

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
)

// Client constants of licensing/client.py.
const (
	ClientTimeoutSeconds = 20
	MaxDifficulty        = 26
	maxAnswerBytes       = 1 << 20
	maxErrorBytes        = 1 << 16
)

// Server error texts, byte-identical to client.py.
const (
	NoServerText   = "No license server address is set. Open 'Server & offline options' and enter the address from your purchase email."
	OfflineText    = "Cannot reach the license server. Check the internet connection, or use offline activation."
	BadPuzzleText  = "The license server asked for an unexpected puzzle. Try again later."
	defaultErrCode = "server_error"
)

// ServerError is client.LicenseServerError: a readable failure from, or while reaching,
// the license server. Code is "server_error" unless the server or the client names one.
type ServerError struct {
	Message string
	Code    any // a string, or whatever JSON value the server sent as "code"
}

func (e *ServerError) Error() string { return e.Message }

// Client talks to the license server (client.LicenseClient). Results are the server's
// whole decoded JSON answers: trial and activate carry the certificate plus the
// signed server time ("server_time") that trials anchor their clock to. parts are the
// trial part hashes; email and code are sent only when email is truthy.
type Client interface {
	StartTrial(hwid string, parts *pyjson.Object, email, code any) (any, error)
	RequestTrialCode(hwid, email string) (any, error)
	Activate(licenseKey, hwid string) (any, error)
	// Verify asks whether the license is still valid; the answer counts only when it comes
	// back signed for nonce (VerifiedStatus).
	Verify(licenseKey any, hwid, nonce string) (any, error)
}

// ClientFactory builds a Client for a server URL, as LicenseService.client_factory does.
type ClientFactory func(serverURL string) Client

// allowedServer is the address rule of the client (client.server_allowed): a packaged build
// talks only to the address baked into it (ADR-0097). testServer is the explicit test-only
// seam: https anywhere, or http on this PC. The release composition never sets it.
func allowedServer(url string, testServer bool) bool {
	if testServer {
		return strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://127.0.0.1") ||
			strings.HasPrefix(url, "http://localhost")
	}
	return url == DefaultServerURL
}

// Solve finds the counter where SHA256(nonce + ":" + counter) starts with difficulty zero
// bits (client.solve).
func Solve(nonce string, difficulty int64) (int64, error) {
	if difficulty <= 0 || difficulty > MaxDifficulty {
		return 0, &ServerError{Message: BadPuzzleText, Code: defaultErrCode}
	}
	prefix := []byte(nonce + ":")
	shift := 32 - uint(difficulty)
	buffer := make([]byte, 0, len(prefix)+20)
	for counter := int64(0); ; counter++ {
		buffer = strconv.AppendInt(append(buffer[:0], prefix...), counter, 10)
		digest := sha256.Sum256(buffer)
		if binary.BigEndian.Uint32(digest[:4])>>shift == 0 {
			return counter, nil
		}
	}
}
