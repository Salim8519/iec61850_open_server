package proto

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"studio-engine/internal/snmp/model"
)

func unhex(t *testing.T, text string) []byte {
	t.Helper()
	data, err := hex.DecodeString(strings.ReplaceAll(text, " ", ""))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// A v2c GET of sysDescr.0 as Net-SNMP sends it.
const getSysDescr = "30 26 02 01 01 04 06 70 75 62 6c 69 63 a0 19 02 01 01 02 01 00 02 01 00 " +
	"30 0e 30 0c 06 08 2b 06 01 02 01 01 01 00 05 00"

func TestCommunityMessageBytes(t *testing.T) {
	pdu := PDU{Type: GetRequest, RequestID: 1, VarBinds: []VarBind{{OID: MustOID("1.3.6.1.2.1.1.1.0"), Value: Null}}}
	packet, err := EncodeCommunity(Version2c, "public", pdu)
	if err != nil {
		t.Fatal(err)
	}
	if want := unhex(t, getSysDescr); !bytes.Equal(packet, want) {
		t.Fatalf("got % x\nwant % x", packet, want)
	}
	m, err := Decode(packet)
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != Version2c || m.Community != "public" || m.PDU.Type != GetRequest || m.PDU.RequestID != 1 ||
		m.PDU.VarBinds[0].OID.String() != "1.3.6.1.2.1.1.1.0" || m.PDU.VarBinds[0].Value.Type != TagNull {
		t.Fatalf("decoded %+v", m)
	}
}

func TestValuesRoundTrip(t *testing.T) {
	cases := []struct{ kind, text string }{
		{model.TypeInteger, "-2147483648"}, {model.TypeInteger, "2147483647"}, {model.TypeInteger, "0"},
		{model.TypeInteger, "-129"}, {model.TypeInteger, "128"},
		{model.TypeCounter32, "4294967295"}, {model.TypeGauge32, "128"}, {model.TypeTimeTicks, "0"},
		{model.TypeCounter64, "18446744073709551615"}, {model.TypeOctets, "Line 1\nLine 2 ü"},
		{model.TypeOID, "1.3.6.1.4.1.4294967295.0"}, {model.TypeIPAddress, "192.168.1.200"},
		{model.TypeOID, "2.999.1"}, {model.TypeNull, ""},
	}
	for _, c := range cases {
		value, err := ParseValue(c.kind, c.text, false)
		if err != nil {
			t.Fatalf("%s %q: %v", c.kind, c.text, err)
		}
		pdu := PDU{Type: Response, RequestID: -5, VarBinds: []VarBind{{OID: MustOID("1.3.6.1.4.1.99999.1.0"), Value: value}}}
		packet, err := EncodeCommunity(Version1, "c", pdu)
		if err != nil {
			t.Fatal(err)
		}
		m, err := Decode(packet)
		if err != nil {
			t.Fatalf("%s: %v", c.kind, err)
		}
		kind, text, _ := ValueText(m.PDU.VarBinds[0].Value)
		if kind != c.kind || text != c.text || m.PDU.RequestID != -5 {
			t.Fatalf("%s %q came back as %s %q", c.kind, c.text, kind, text)
		}
	}
	binary, _ := ParseValue(model.TypeOctets, "00 1a ff", true)
	if _, text, isHex := ValueText(binary); text != "00 1a ff" || !isHex {
		t.Fatalf("hex octets came back as %q %v", text, isHex)
	}
	for _, bad := range [][2]string{{model.TypeInteger, "2147483648"}, {model.TypeCounter32, "-1"},
		{model.TypeIPAddress, "300.1.1.1"}, {model.TypeOID, "1.40"}, {model.TypeOID, "abc"}} {
		if _, err := ParseValue(bad[0], bad[1], false); err == nil {
			t.Fatalf("%s %q was accepted", bad[0], bad[1])
		}
	}
}

func TestTrapV1AndBulkRoundTrip(t *testing.T) {
	trap := PDU{Type: TrapV1, Enterprise: MustOID("1.3.6.1.4.1.99999"), AgentAddr: [4]byte{127, 60, 0, 1},
		GenericTrap: 2, SpecificTrap: 0, Timestamp: 4242,
		VarBinds: []VarBind{{OID: MustOID("1.3.6.1.2.1.2.2.1.1.3"), Value: Value{Type: TagInteger, Int: 3}}}}
	packet, err := EncodeCommunity(Version1, "public", trap)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Decode(packet)
	if err != nil {
		t.Fatal(err)
	}
	got := m.PDU
	if got.Enterprise.String() != "1.3.6.1.4.1.99999" || got.AgentAddr != trap.AgentAddr || got.GenericTrap != 2 ||
		got.Timestamp != 4242 || got.VarBinds[0].Value.Int != 3 {
		t.Fatalf("trap came back as %+v", got)
	}
	bulk := PDU{Type: GetBulkRequest, RequestID: 77, ErrorStatus: 1, ErrorIndex: 25,
		VarBinds: []VarBind{{OID: MustOID("1.3.6.1.2.1.1"), Value: Null}}}
	packet, _ = EncodeCommunity(Version2c, "public", bulk)
	if m, err = Decode(packet); err != nil || m.PDU.ErrorStatus != 1 || m.PDU.ErrorIndex != 25 {
		t.Fatalf("bulk came back as %+v, %v", m, err)
	}
}

func TestMalformedPacketsAreRefused(t *testing.T) {
	valid := unhex(t, getSysDescr)
	for cut := 0; cut < len(valid); cut++ {
		if _, err := Decode(valid[:cut]); err == nil {
			t.Fatalf("a packet cut at %d bytes was accepted", cut)
		}
	}
	for _, text := range []string{"30 03 02 01 07", "30 84 ff ff ff ff", "04 00", "30 05 02 01 03 30 00"} {
		if _, err := Decode(unhex(t, text)); err == nil {
			t.Fatalf("%s was accepted", text)
		}
	}
}

func TestOIDOrder(t *testing.T) {
	a, b, c := MustOID("1.3.6.1.2.1.1"), MustOID("1.3.6.1.2.1.1.0"), MustOID("1.3.6.1.2.1.2")
	if a.Compare(b) != -1 || b.Compare(c) != -1 || c.Compare(a) != 1 || a.Compare(a) != 0 {
		t.Fatal("OID order is wrong")
	}
	if !b.HasPrefix(a) || c.HasPrefix(a) {
		t.Fatal("OID prefix is wrong")
	}
}
