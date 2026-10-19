package licensing

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"
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

// Client talks to the license server (client.LicenseClient). Results are decoded JSON
// values: the certificate (a string or an object) and the verify answer.
type Client interface {
	StartTrial(hwid string) (any, error)
	Activate(licenseKey, hwid string) (any, error)
	Transfer(licenseKey, hwid string) (any, error)
	Verify(licenseKey any, hwid string) (any, error)
}

// ClientFactory builds a Client for a server URL, as LicenseService.client_factory does.
type ClientFactory func(serverURL string) Client

// allowedServer is the address rule shared by the client and set_server_url.
func allowedServer(url string) bool {
	return strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://127.0.0.1") ||
		strings.HasPrefix(url, "http://localhost")
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
