package proto

import "fmt"

// usmStats counters reported to a manager (RFC 3414 section 5), by failure.
var (
	StatsUnsupportedSecLevels = MustOID("1.3.6.1.6.3.15.1.1.1.0")
	StatsNotInTimeWindows     = MustOID("1.3.6.1.6.3.15.1.1.2.0")
	StatsUnknownUserNames     = MustOID("1.3.6.1.6.3.15.1.1.3.0")
	StatsUnknownEngineIDs     = MustOID("1.3.6.1.6.3.15.1.1.4.0")
	StatsWrongDigests         = MustOID("1.3.6.1.6.3.15.1.1.5.0")
	StatsDecryptionErrors     = MustOID("1.3.6.1.6.3.15.1.1.6.0")
)

// ReportText is what a received usmStats report means, in plain words.
func ReportText(oid OID) string {
	texts := map[string]string{
		StatsUnsupportedSecLevels.String(): "the device does not allow this security level for the user",
		StatsNotInTimeWindows.String():     "the device clock did not match (retrying fixes this)",
		StatsUnknownUserNames.String():     "the device does not know this user name",
		StatsUnknownEngineIDs.String():     "the engine ID was not known yet",
		StatsWrongDigests.String():         "wrong authentication password or protocol",
		StatsDecryptionErrors.String():     "wrong privacy password or protocol",
	}
	if text, found := texts[oid.String()]; found {
		return text
	}
	return "the device reported " + oid.String()
}

// TimeWindow is how far, in seconds, a message's engine time may be from the engine's.
const TimeWindow = 150

// EncodeV3 builds a v3 message for h's engine, user, flags and context. keys are needed when
// the flags ask for authentication; salt is the sender's privacy salt counter.
func EncodeV3(h Header, p PDU, keys *Keys, salt uint64) ([]byte, error) {
	if h.Flags&FlagAuth != 0 && (keys == nil || keys.Auth == "") {
		return nil, fmt.Errorf("authentication needs the user's keys")
	}
	if h.Flags&FlagPriv != 0 && (keys == nil || keys.Priv == "") {
		return nil, fmt.Errorf("privacy needs the user's keys")
	}
	body, err := EncodePDU(p)
	if err != nil {
		return nil, err
	}
	scoped := sequence(TagSequence, octets(h.ContextEngineID), octets([]byte(h.ContextName)), body)
	data, privParams := scoped, []byte{}
	if h.Flags&FlagPriv != 0 {
		var encrypted []byte
		privParams, encrypted, err = encrypt(keys, h.Boots, h.Time, scoped, salt)
		if err != nil {
			return nil, err
		}
		data = octets(encrypted)
	}
	authParams := []byte{}
	if h.Flags&FlagAuth != 0 {
		authParams = make([]byte, authSpecs[keys.Auth].macLen)
	}
	maxSize := h.MaxSize
	if maxSize == 0 {
		maxSize = 65507
	}
	usm := sequence(TagSequence, octets(h.EngineID), intElement(int64(h.Boots)), intElement(int64(h.Time)),
		octets([]byte(h.UserName)), octets(authParams), octets(privParams))
	global := sequence(TagSequence, intElement(int64(h.MsgID)), intElement(int64(maxSize)),
		octets([]byte{h.Flags}), intElement(SecurityModelUSM))
	packet := sequence(TagSequence, intElement(Version3), global, octets(usm), data)
	if h.Flags&FlagAuth == 0 {
		return packet, nil
	}
	parsed, err := Decode(packet)
	if err != nil {
		return nil, err
	}
	copy(packet[offsetIn(packet, parsed.Header.AuthParams):], mac(keys, packet))
	return packet, nil
}

// Unprotect checks the digest and decrypts a received v3 message with the user's keys,
// returning ErrWrongDigest, ErrDecryption or a malformed-packet error.
func Unprotect(m *Message, keys *Keys) error {
	h := m.Header
	if h.Flags&FlagAuth != 0 {
		if keys == nil || keys.Auth == "" {
			return ErrWrongDigest
		}
		if err := verify(keys, h); err != nil {
			return err
		}
	}
	if h.Flags&FlagPriv == 0 {
		return nil
	}
	if keys.Priv == "" {
		return ErrDecryption
	}
	plain, err := decrypt(keys, h)
	if err != nil {
		return err
	}
	if err := decodeScoped(&reader{data: plain}, m); err != nil {
		return ErrDecryption
	}
	h.Encrypted = nil
	return nil
}

// LevelFlags are the msgFlags of a security level.
func LevelFlags(level string) byte {
	switch level {
	case "authPriv":
		return FlagAuth | FlagPriv
	case "authNoPriv":
		return FlagAuth
	}
	return 0
}
