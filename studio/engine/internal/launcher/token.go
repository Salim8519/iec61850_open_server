package launcher

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
)

// Fixed development values (entrypoint.py): a dev reload keeps the tab signed in.
const DevSecret = "studio-dev-secret"

// LaunchToken is secrets.token_urlsafe(32): 32 random bytes, URL-safe base64, no padding.
func LaunchToken() string {
	return base64.RawURLEncoding.EncodeToString(randomBytes(32))
}

// SecretKey is secrets.token_hex(32), Flask's per-launch SECRET_KEY in the packaged app.
func SecretKey() string { return hex.EncodeToString(randomBytes(32)) }

func randomBytes(n int) []byte {
	data := make([]byte, n)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	return data
}
