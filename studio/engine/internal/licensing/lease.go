package licensing

import (
	"regexp"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
)

// LeaseStore keeps what a lease license needs between runs (ADR-0098), each value in the
// three places of the trial clock (dot-file, HKCU\Software\ScadaBench, license_state.json):
//
//   - the lease token, as "<server time>:<32 hex>". The server hands out a new token with
//     every valid answer; the newest copy is the one to send. A stale token means this
//     is a copy and the server answers "moved".
//   - the last valid signed verify answer ({payload, signature}). Only the license server can
//     make it, so its time proves when the license was last confirmed; editing a file cannot
//     extend a lease.
type LeaseStore struct {
	Token  ClockValue
	Answer ClockValue
	Keys   map[string][]byte
}

// NewLeaseStore returns the lease record kept under home.
func NewLeaseStore(home string, registry Registry, keys map[string][]byte) LeaseStore {
	return LeaseStore{Token: NewClockValue("lease_token", home, registry),
		Answer: NewClockValue("lease_answer", home, registry), Keys: keys}
}

var leaseTokenPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// ValidLeaseToken reports whether text is a lease token: 32 lowercase hex characters.
func ValidLeaseToken(text string) bool { return leaseTokenPattern.MatchString(text) }

// CurrentToken is the newest stored lease token, "" when there is none.
func (l LeaseStore) CurrentToken() string {
	var token string
	var newest int64 = -1
	for _, copy := range l.Token.ReadAll() {
		stamp, text, found := strings.Cut(copy, ":")
		at, err := strconv.ParseInt(stamp, 10, 64)
		if !found || err != nil || !ValidLeaseToken(text) {
			continue
		}
		if at > newest {
			newest, token = at, text
		}
	}
	return token
}

// SetToken stores token everywhere; at is the server time it came with.
func (l LeaseStore) SetToken(at int64, token string) {
	if ValidLeaseToken(token) {
		l.Token.Write(strconv.FormatInt(at, 10) + ":" + token)
	}
}

// receipt returns the newest genuine "valid" verify answer stored for this license and
// computer: signed by the license server and naming this license ID and machine ID.
func (l LeaseStore) receipt(licenseID any, hwid string) (moment int64, bundle Bundle, found bool) {
	id, isText := licenseID.(string)
	if !isText {
		return 0, Bundle{}, false
	}
	for _, copy := range l.Answer.ReadAll() {
		parsed, err := ParseBundle(copy)
		if err != nil {
			continue
		}
		payload, err := VerifyBundle(parsed, l.Keys)
		if err != nil || !validAnswerFor(payload, id, hwid) {
			continue
		}
		raw, _ := payload.Get("time")
		if at, err := pyInt(raw); err == nil && at > moment {
			moment, bundle, found = at, parsed, true
		}
	}
	return moment, bundle, found
}

// validAnswerFor reports whether a verified payload is a "valid" verify answer for this
// license and machine (its nonce belonged to an earlier request, so it is not compared).
func validAnswerFor(payload *pyjson.Object, licenseID, hwid string) bool {
	if purpose, _ := payload.Get("purpose"); purpose != any(VerifyAnswerPurpose) {
		return false
	}
	if version, _ := payload.Get("version"); version != any(int64(1)) {
		return false
	}
	if status, _ := payload.Get("status"); status != any("valid") {
		return false
	}
	if got, _ := payload.Get("license_id"); got != any(licenseID) {
		return false
	}
	got, _ := payload.Get("hwid")
	return got == any(hwid)
}

// ReceiptTime is the signed time of the newest stored valid answer for the license in
// payload, 0 when there is none.
func (l LeaseStore) ReceiptTime(payload *pyjson.Object, hwid string) int64 {
	licenseID, _ := payload.Get("license_id")
	moment, _, _ := l.receipt(licenseID, hwid)
	return moment
}

// SetReceipt stores a valid answer everywhere (replacing older ones).
func (l LeaseStore) SetReceipt(bundle Bundle) {
	if data, err := pyjson.Dumps(bundle.Object()); err == nil {
		l.Answer.Write(string(data))
	}
}

// ClearReceipt forgets the stored answer (a new activation starts a new lease).
func (l LeaseStore) ClearReceipt() { l.Answer.Write("") }
