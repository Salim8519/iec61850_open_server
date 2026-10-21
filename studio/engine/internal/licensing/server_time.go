package licensing

import "errors"

// ServerTimePurpose marks a license-server time stamp (licensing/server_time.py). A stamp
// payload is {"kid", "purpose": "server-time", "time", "version": 1}, signed with the
// same Ed25519 keys as certificates. It has no "type" or "hwid", so it can never pass as
// a license, and a license never passes as a stamp. Native twin: license_clock.cpp.
const ServerTimePurpose = "server-time"

// VerifyServerTime returns the time of a genuine stamp (decoded JSON or its text); ok is
// false for missing, foreign or altered input. An error stands for the OverflowError
// Python does not catch (an infinite "version" or "time" in a genuinely signed stamp).
func VerifyServerTime(stamp any, keys map[string][]byte) (moment int64, ok bool, err error) {
	if !truthy(stamp) {
		return 0, false, nil
	}
	bundle, err := ParseBundle(stamp)
	if err != nil {
		return 0, false, nil
	}
	payload, err := VerifyBundle(bundle, keys)
	if err != nil {
		return 0, false, nil // CertificateError, or the caught TypeError of a list "kid"
	}
	if purpose, _ := payload.Get("purpose"); purpose != any(ServerTimePurpose) {
		return 0, false, nil
	}
	version, found := payload.Get("version")
	if !found {
		version = int64(0)
	}
	number, err := pyInt(version)
	if err != nil || number != 1 {
		return 0, false, overflowOnly(err)
	}
	raw, found := payload.Get("time")
	if !found {
		return 0, false, nil
	}
	if moment, err = pyInt(raw); err != nil {
		return 0, false, overflowOnly(err)
	}
	return moment, moment > 0, nil
}

// overflowOnly keeps only the error Python does not catch here.
func overflowOnly(err error) error {
	if errors.Is(err, errIntOverflow) {
		return uncaught(err.Error())
	}
	return nil
}
