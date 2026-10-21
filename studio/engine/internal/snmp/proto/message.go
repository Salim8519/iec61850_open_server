package proto

import "fmt"

// Message versions on the wire.
const (
	Version1  = 0
	Version2c = 1
	Version3  = 3
)

// msgFlags bits (RFC 3412).
const (
	FlagAuth       byte = 0x01
	FlagPriv       byte = 0x02
	FlagReportable byte = 0x04
)

// SecurityModelUSM is the only v3 security model ScadaBench speaks.
const SecurityModelUSM = 3

// Message is one SNMP message. Community is used by v1/v2c, Header by v3.
type Message struct {
	Version   int
	Community string
	PDU       PDU
	Header    *Header
}

// Header is the v3 part of a message: header data, USM parameters and the scoped PDU's
// context. Encrypted holds msgData while it is still encrypted.
type Header struct {
	MsgID           int32
	MaxSize         int32
	Flags           byte
	SecurityModel   int
	EngineID        []byte
	Boots           int32
	Time            int32
	UserName        string
	AuthParams      []byte
	PrivParams      []byte
	ContextEngineID []byte
	ContextName     string
	Encrypted       []byte

	raw []byte // the whole packet, for checking the digest
}

// Level is the security level the flags ask for.
func (h *Header) Level() string {
	switch {
	case h.Flags&FlagPriv != 0:
		return "authPriv"
	case h.Flags&FlagAuth != 0:
		return "authNoPriv"
	}
	return "noAuthNoPriv"
}

// EncodeCommunity builds a v1 or v2c message.
func EncodeCommunity(version int, community string, p PDU) ([]byte, error) {
	if version != Version1 && version != Version2c {
		return nil, fmt.Errorf("community messages are v1 or v2c")
	}
	body, err := EncodePDU(p)
	if err != nil {
		return nil, err
	}
	return sequence(TagSequence, intElement(int64(version)), octets([]byte(community)), body), nil
}

// Decode reads any SNMP message. A v3 message comes back with its header; when it is
// encrypted, PDU is empty until Unprotect decrypts it.
func Decode(packet []byte) (*Message, error) {
	outer := &reader{data: packet}
	content, err := outer.expect(TagSequence, "message")
	if err != nil {
		return nil, err
	}
	r := &reader{data: content}
	version, err := r.integer("version")
	if err != nil {
		return nil, err
	}
	m := &Message{Version: int(version)}
	switch version {
	case Version1, Version2c:
		community, err := r.expect(TagOctetString, "community")
		if err != nil {
			return nil, err
		}
		m.Community = string(community)
		m.PDU, err = decodePDU(r)
		return m, err
	case Version3:
		m.Header = &Header{raw: packet}
		return m, decodeV3(r, m)
	}
	return nil, malformed(fmt.Sprintf("unsupported version %d", version))
}

func decodeV3(r *reader, m *Message) error {
	h := m.Header
	global, err := r.expect(TagSequence, "msgGlobalData")
	if err != nil {
		return err
	}
	g := &reader{data: global}
	id, err := g.integer("msgID")
	if err != nil {
		return err
	}
	size, err := g.integer("msgMaxSize")
	if err != nil {
		return err
	}
	flags, err := g.expect(TagOctetString, "msgFlags")
	if err != nil || len(flags) != 1 {
		return malformed("msgFlags")
	}
	model, err := g.integer("msgSecurityModel")
	if err != nil {
		return err
	}
	h.MsgID, h.MaxSize, h.Flags, h.SecurityModel = int32(id), int32(size), flags[0], int(model)
	if h.Flags&FlagPriv != 0 && h.Flags&FlagAuth == 0 {
		return malformed("privacy without authentication")
	}
	security, err := r.expect(TagOctetString, "msgSecurityParameters")
	if err != nil {
		return err
	}
	if err := decodeUSM(security, h); err != nil {
		return err
	}
	if h.Flags&FlagPriv != 0 {
		h.Encrypted, err = r.expect(TagOctetString, "encryptedPDU")
		return err
	}
	return decodeScoped(r, m)
}

func decodeUSM(security []byte, h *Header) error {
	outer := &reader{data: security}
	content, err := outer.expect(TagSequence, "UsmSecurityParameters")
	if err != nil {
		return err
	}
	u := &reader{data: content}
	if h.EngineID, err = u.expect(TagOctetString, "msgAuthoritativeEngineID"); err != nil {
		return err
	}
	boots, err := u.integer("msgAuthoritativeEngineBoots")
	if err != nil {
		return err
	}
	clock, err := u.integer("msgAuthoritativeEngineTime")
	if err != nil {
		return err
	}
	user, err := u.expect(TagOctetString, "msgUserName")
	if err != nil {
		return err
	}
	if h.AuthParams, err = u.expect(TagOctetString, "msgAuthenticationParameters"); err != nil {
		return err
	}
	if h.PrivParams, err = u.expect(TagOctetString, "msgPrivacyParameters"); err != nil {
		return err
	}
	h.Boots, h.Time, h.UserName = int32(boots), int32(clock), string(user)
	return nil
}

// decodeScoped reads a plaintext ScopedPDU.
func decodeScoped(r *reader, m *Message) error {
	content, err := r.expect(TagSequence, "ScopedPDU")
	if err != nil {
		return err
	}
	s := &reader{data: content}
	if m.Header.ContextEngineID, err = s.expect(TagOctetString, "contextEngineID"); err != nil {
		return err
	}
	name, err := s.expect(TagOctetString, "contextName")
	if err != nil {
		return err
	}
	m.Header.ContextName = string(name)
	m.PDU, err = decodePDU(s)
	return err
}
