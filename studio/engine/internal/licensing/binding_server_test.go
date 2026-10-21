package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/licensing/tpm"
	"studio-engine/internal/pyjson"
)

// bindServer is a small stand-in for the license server's ADR-0098 behaviour (the contract in
// docs/architecture/license-binding-contract.md): chip challenge, lease tokens, signed verify
// answers. The real server is exercised by the end-to-end test of the orchestrator.
type bindServer struct {
	t          *testing.T
	private    ed25519.PrivateKey
	public     ed25519.PublicKey
	clock      *float64
	acceptChip bool // false: the server cannot use a chip and answers a plain certificate
	offline    bool
	licenses   map[string]*bindRecord
	sessions   map[string]bindSession
	tokens     int
	calls      []string
}

type bindRecord struct {
	hwid, ek, ak, token string
	revoked             bool
}

type bindSession struct {
	key, hwid, ek, ak, credential string
}

const bindKey = "SUBS-2222-3333-4444-5555"

func newBindServer(t *testing.T, clock *float64) *bindServer {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return &bindServer{t: t, private: private, public: public, clock: clock, acceptChip: true,
		licenses: map[string]*bindRecord{bindKey: {}}, sessions: map[string]bindSession{}}
}

func (s *bindServer) keys() map[string][]byte {
	return map[string][]byte{KeyID(s.public): s.public}
}

func (s *bindServer) sign(payload map[string]any) map[string]any {
	payload["kid"] = KeyID(s.public)
	text, err := json.Marshal(payload) // keys come out sorted, as the real server writes them
	if err != nil {
		s.t.Fatal(err)
	}
	signature := ed25519.Sign(s.private, text)
	return map[string]any{"payload": string(text), "signature": base64.RawURLEncoding.EncodeToString(signature)}
}

func (s *bindServer) now() int64 { return int64(*s.clock) }

func (s *bindServer) newToken() string {
	s.tokens++
	return fmt.Sprintf("%032x", 0x1000+s.tokens)
}

func (s *bindServer) certificate(key string, record *bindRecord) map[string]any {
	payload := map[string]any{"email": "buyer@example.com", "expires_at": 0, "hwid": record.hwid,
		"issued_at": s.now(), "license_id": key, "max_machines": 1, "type": "lifetime", "version": 1}
	if record.ak != "" {
		payload["ak"], payload["ek"] = record.ak, record.ek
	} else {
		payload["lease_days"] = 30
	}
	return s.sign(payload)
}

func (s *bindServer) stamp() map[string]any {
	return s.sign(map[string]any{"purpose": "server-time", "time": s.now(), "version": 1})
}

func (s *bindServer) decode(value map[string]any) any {
	data, err := json.Marshal(value)
	if err != nil {
		s.t.Fatal(err)
	}
	return mustDecode(s.t, string(data))
}

func (s *bindServer) refuse(code string) error {
	return &ServerError{Message: "refused: " + code, Code: code}
}

func (s *bindServer) StartTrial(string, *pyjson.Object, any, any) (any, error) {
	return nil, offline()
}
func (s *bindServer) RequestTrialCode(string, string) (any, error) { return nil, offline() }

func (s *bindServer) Activate(key, hwid string) (any, error) {
	return s.ActivateWithChip(key, hwid, nil)
}

func (s *bindServer) ActivateWithChip(key, hwid string, offer *pyjson.Object) (any, error) {
	s.calls = append(s.calls, "activate")
	if s.offline {
		return nil, offline()
	}
	record := s.licenses[key]
	if record == nil {
		return nil, s.refuse("unknown_key")
	}
	ek := ""
	if offer != nil {
		ek, _ = lookupString(offer, "ek_public")
	}
	// A chip license is re-activated only by its own chip; a lease license by the same computer.
	if record.hwid != "" && !(record.ek != "" && ek == record.ek) && !(record.ek == "" && record.hwid == hwid) {
		return nil, s.refuse("bound_elsewhere")
	}
	if offer != nil && s.acceptChip {
		return s.challenge(key, hwid, record, offer)
	}
	record.hwid, record.ak, record.ek, record.token = hwid, "", "", s.newToken()
	return s.decode(map[string]any{"certificate": s.certificate(key, record), "server_time": s.stamp(),
		"lease_token": record.token}), nil
}

func (s *bindServer) challenge(key, hwid string, record *bindRecord, offer *pyjson.Object) (any, error) {
	text := func(object *pyjson.Object, name string) string {
		value, _ := lookupString(object, name)
		return value
	}
	ek := text(offer, "ek_public")
	inner, _ := offer.Get("ak")
	akObject, _ := inner.(*pyjson.Object)
	session := fmt.Sprintf("%032x", len(s.sessions)+1)
	credential := base64.StdEncoding.EncodeToString([]byte("credential-" + session))
	s.sessions[session] = bindSession{key: key, hwid: hwid, ek: ek, ak: text(akObject, "public"), credential: credential}
	return s.decode(map[string]any{"challenge": map[string]any{"session": session, "credential": credential,
		"secret": base64.StdEncoding.EncodeToString([]byte("secret"))}}), nil
}

func (s *bindServer) ActivateChip(session, secret string) (any, error) {
	s.calls = append(s.calls, "activate-chip")
	pending, found := s.sessions[session]
	if !found || secret != pending.credential {
		return nil, s.refuse("bad_request")
	}
	delete(s.sessions, session)
	record := s.licenses[pending.key]
	record.hwid, record.ek, record.ak, record.token = pending.hwid, pending.ek, pending.ak, ""
	return s.decode(map[string]any{"certificate": s.certificate(pending.key, record), "server_time": s.stamp()}), nil
}

func (s *bindServer) Verify(key any, hwid, nonce string) (any, error) {
	return s.VerifyLease(key, hwid, nonce, "")
}

func (s *bindServer) VerifyLease(key any, hwid, nonce, token string) (any, error) {
	s.calls = append(s.calls, "verify:"+token)
	if s.offline {
		return nil, offline()
	}
	id, _ := key.(string)
	record := s.licenses[id]
	status, next := "unknown", ""
	switch {
	case record == nil:
	case record.revoked:
		status = "revoked"
	case record.hwid != hwid:
		status = "moved"
	case record.ak != "":
		status = "valid"
	case record.token == "" || record.token == token:
		status, next = "valid", s.newToken()
		record.token = next
	default:
		status = "moved" // a stale token: this is a copy
	}
	answer := map[string]any{"status": status, "server_time": s.stamp(), "answer": s.sign(map[string]any{
		"hwid": hwid, "lease_token": next, "license_id": id, "nonce": nonce, "purpose": VerifyAnswerPurpose,
		"status": status, "time": s.now(), "version": 1})}
	if status == "valid" {
		answer["certificate"] = s.certificate(id, record)
	}
	return s.decode(answer), nil
}

// bindEnv is one computer with the license service on it.
type bindEnv struct {
	t        *testing.T
	server   *bindServer
	clock    *float64
	hwid     string
	home     string
	registry memoryRegistry
	chip     tpm.Chip
	service  *Service
}

func newBindEnv(t *testing.T, chip tpm.Chip) *bindEnv {
	t.Setenv("STUDIO_LICENSE_CERT_PATH", "")
	clock := 1_760_000_000.0
	server := newBindServer(t, &clock)
	env := &bindEnv{t: t, server: server, clock: &clock, hwid: strings.Repeat("ab", 32), chip: chip}
	env.registry = memoryRegistry{}
	env.home = scenarioDir(t)
	env.service = env.newService()
	return env
}

// newService builds the service over env.home, env.registry and env.chip.
func (e *bindEnv) newService() *Service {
	return New(Options{Home: e.home, DataDir: filepath.Join(e.home, "data"), Enforce: true, Keys: e.server.keys(),
		HWID: func() (string, error) { return e.hwid, nil }, Clock: func() float64 { return *e.clock },
		Debugger: func() bool { return false }, Logf: e.t.Logf, Registry: e.registry, Chip: e.chip,
		Nonce:         func() string { return RandomNonce() },
		ClientFactory: func(string) Client { return e.server }})
}

// copyTo is a copy of this computer's license files (and registry values) on another
// computer, as a cloned disk or a copied profile would hold them.
func (e *bindEnv) copyTo(chip tpm.Chip) *bindEnv {
	clone := &bindEnv{t: e.t, server: e.server, clock: e.clock, hwid: e.hwid, chip: chip,
		home: scenarioDir(e.t), registry: memoryRegistry{}}
	entries, _ := os.ReadDir(e.home)
	for _, entry := range entries {
		if data, err := os.ReadFile(filepath.Join(e.home, entry.Name())); err == nil && !entry.IsDir() {
			os.WriteFile(filepath.Join(clone.home, entry.Name()), data, 0o666)
		}
	}
	for name, value := range e.registry {
		clone.registry[name] = value
	}
	clone.service = clone.newService()
	return clone
}

func (e *bindEnv) advance(days float64) { *e.clock += days * 86400 }

func (e *bindEnv) status() Status {
	status, err := e.service.Current(true)
	if err != nil {
		e.t.Fatal(err)
	}
	return status
}

func (e *bindEnv) activate() {
	if _, err := e.service.Activate(bindKey); err != nil {
		e.t.Fatalf("activate: %v", err)
	}
}

func (e *bindEnv) certificatePayload() *pyjson.Object {
	data, err := os.ReadFile(filepath.Join(e.home, "license.cert"))
	if err != nil {
		e.t.Fatal(err)
	}
	bundle, err := ParseBundle(string(data))
	if err != nil {
		e.t.Fatal(err)
	}
	payload, err := VerifyBundle(bundle, e.server.keys())
	if err != nil {
		e.t.Fatal(err)
	}
	return payload
}
