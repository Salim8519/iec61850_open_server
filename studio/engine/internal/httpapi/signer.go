package httpapi

import (
	"bytes"
	"compress/zlib"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"io"
	"math/big"
	"strings"
	"time"

	"studio-engine/internal/pyjson"
)

// Flask's SecureCookieSessionInterface signs the session with itsdangerous'
// URLSafeTimedSerializer: salt "cookie-session", HMAC key derivation, SHA-1, the
// compact JSON payload (zlib-compressed with a leading "." when that is shorter),
// URL-safe base64 without padding, then ".<timestamp>.<signature>".
const (
	sessionSalt   = "cookie-session"
	sessionMaxAge = 31 * 24 * time.Hour // PERMANENT_SESSION_LIFETIME, checked on load
)

var b64 = base64.RawURLEncoding

type signer struct{ key []byte }

func newSigner(secret string) signer {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(sessionSalt))
	return signer{key: mac.Sum(nil)}
}

func (s signer) signature(value string) string {
	mac := hmac.New(sha1.New, s.key)
	mac.Write([]byte(value))
	return b64.EncodeToString(mac.Sum(nil))
}

// dumps serializes and signs a session object at time now.
func (s signer) dumps(session *pyjson.Object, now time.Time) (string, error) {
	payload, err := pyjson.Compact(session)
	if err != nil {
		return "", err
	}
	var packed bytes.Buffer
	writer, _ := zlib.NewWriterLevel(&packed, zlib.BestCompression)
	_, _ = writer.Write(payload)
	_ = writer.Close()
	value := b64.EncodeToString(payload)
	if packed.Len() < len(payload)-1 {
		value = "." + b64.EncodeToString(packed.Bytes())
	}
	stamp := b64.EncodeToString(big.NewInt(now.Unix()).Bytes())
	value += "." + stamp
	return value + "." + s.signature(value), nil
}

// loads verifies and decodes a signed session; any failure gives nil (an empty session).
func (s signer) loads(cookie string, now time.Time) *pyjson.Object {
	signed, signature, found := cut(cookie)
	if !found || !hmac.Equal([]byte(s.signature(signed)), []byte(signature)) {
		return nil
	}
	value, stampText, found := cut(signed)
	if !found {
		return nil
	}
	stamp, err := b64.DecodeString(stampText)
	if err != nil || len(stamp) > 8 {
		return nil
	}
	age := now.Unix() - new(big.Int).SetBytes(stamp).Int64()
	if age < 0 || age > int64(sessionMaxAge/time.Second) {
		return nil
	}
	payload, ok := unpack(value)
	if !ok {
		return nil
	}
	decoded, err := pyjson.Decode(payload)
	if err != nil {
		return nil
	}
	session, _ := decoded.(*pyjson.Object)
	return session
}

func cut(text string) (string, string, bool) {
	index := strings.LastIndexByte(text, '.')
	if index < 0 {
		return "", "", false
	}
	return text[:index], text[index+1:], true
}

func unpack(value string) ([]byte, bool) {
	compressed := strings.HasPrefix(value, ".")
	data, err := b64.DecodeString(strings.TrimPrefix(value, "."))
	if err != nil {
		return nil, false
	}
	if !compressed {
		return data, true
	}
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	defer reader.Close()
	data, err = io.ReadAll(io.LimitReader(reader, 1<<20))
	return data, err == nil
}
