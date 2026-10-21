package proto

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"

	"studio-engine/internal/snmp/model"
)

// authSpec is an authentication protocol: its hash and the digest length on the wire.
type authSpec struct {
	hash   func() hash.Hash
	macLen int
}

var authSpecs = map[string]authSpec{
	"md5":    {md5.New, 12},
	"sha":    {sha1.New, 12},
	"sha224": {sha256.New224, 16},
	"sha256": {sha256.New, 24},
	"sha384": {sha512.New384, 32},
	"sha512": {sha512.New, 48},
}

// privKeyLen is how many key bytes each privacy protocol needs (DES: key and pre-IV).
var privKeyLen = map[string]int{"des": 16, "aes128": 16, "aes192": 24, "aes256": 32, "aes192c": 24, "aes256c": 32}

// MinPassword is the shortest v3 password (RFC 3414 section 11.2; Net-SNMP refuses shorter).
const MinPassword = 8

// Keys are a user's keys localized to one engine.
type Keys struct {
	Level   string
	Auth    string
	AuthKey []byte
	Priv    string
	PrivKey []byte
}

// PasswordToKey is RFC 3414 A.2: the hash of the password repeated to 1 MB.
func PasswordToKey(auth string, password []byte) []byte {
	h := authSpecs[auth].hash()
	const total = 1 << 20
	block := make([]byte, 64)
	at := 0
	for written := 0; written < total; written += len(block) {
		for index := range block {
			block[index] = password[at%len(password)]
			at++
		}
		h.Write(block)
	}
	return h.Sum(nil)
}

// LocalizeKey is H(Ku || engineID || Ku).
func LocalizeKey(auth string, ku, engineID []byte) []byte {
	h := authSpecs[auth].hash()
	h.Write(ku)
	h.Write(engineID)
	h.Write(ku)
	return h.Sum(nil)
}

// CheckUser returns the reason a user cannot be used, or "".
func CheckUser(u model.User) string {
	if u.Name == "" || len(u.Name) > 32 {
		return "A v3 user needs a name of 1 to 32 characters"
	}
	switch u.Level {
	case model.LevelNoAuth:
		return ""
	case model.LevelAuth, model.LevelPrivacy:
	default:
		return fmt.Sprintf("User %s: choose a security level", u.Name)
	}
	if _, known := authSpecs[u.Auth]; !known {
		return fmt.Sprintf("User %s: choose an authentication protocol", u.Name)
	}
	if len(u.AuthPassword) < MinPassword {
		return fmt.Sprintf("User %s: the authentication password needs at least %d characters", u.Name, MinPassword)
	}
	if u.Level == model.LevelAuth {
		return ""
	}
	if _, known := privKeyLen[u.Priv]; !known {
		return fmt.Sprintf("User %s: choose a privacy protocol", u.Name)
	}
	if len(u.PrivPassword) < MinPassword {
		return fmt.Sprintf("User %s: the privacy password needs at least %d characters", u.Name, MinPassword)
	}
	return ""
}

// KeyCache keeps the slow password-to-key results (1 MB of hashing each) by protocol and password.
type KeyCache map[string][]byte

func (c KeyCache) ku(auth, password string) []byte {
	name := auth + "\x00" + password
	if key, found := c[name]; found {
		return key
	}
	key := PasswordToKey(auth, []byte(password))
	c[name] = key
	return key
}

// Localize derives a user's keys for engineID; the user must pass CheckUser.
func (c KeyCache) Localize(u model.User, engineID []byte) (*Keys, error) {
	if problem := CheckUser(u); problem != "" {
		return nil, fmt.Errorf("%s", problem)
	}
	keys := &Keys{Level: u.Level}
	if u.Level == model.LevelNoAuth {
		return keys, nil
	}
	keys.Auth = u.Auth
	keys.AuthKey = LocalizeKey(u.Auth, c.ku(u.Auth, u.AuthPassword), engineID)
	if u.Level == model.LevelAuth {
		return keys, nil
	}
	keys.Priv = u.Priv
	key := LocalizeKey(u.Auth, c.ku(u.Auth, u.PrivPassword), engineID)
	keys.PrivKey = extendKey(u.Auth, u.Priv, key, engineID)
	return keys, nil
}

// extendKey makes a localized privacy key long enough. Blumenthal (aes192, aes256) appends
// H(key) until long enough; Reeder (aes192c, aes256c, as Cisco and Net-SNMP) appends the
// localized key of the key itself used as a password.
func extendKey(auth, priv string, key, engineID []byte) []byte {
	need := privKeyLen[priv]
	for len(key) < need {
		var more []byte
		if priv == "aes192c" || priv == "aes256c" {
			more = LocalizeKey(auth, PasswordToKey(auth, key), engineID)
		} else {
			h := authSpecs[auth].hash()
			h.Write(key)
			more = h.Sum(nil)
		}
		key = append(append([]byte{}, key...), more...)
	}
	return key[:need]
}

// MacLen is the digest length of an authentication protocol (0 when unknown).
func MacLen(auth string) int { return authSpecs[auth].macLen }
