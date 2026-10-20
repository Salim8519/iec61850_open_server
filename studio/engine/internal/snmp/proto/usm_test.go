package proto

import (
	"bytes"
	"errors"
	"testing"

	"studio-engine/internal/snmp/model"
)

// RFC 3414 appendix A.3: "maplesyrup" localized to engine ID 00...02.
func TestRFC3414KeyVectors(t *testing.T) {
	engine := unhex(t, "000000000000000000000002")
	cases := []struct{ auth, ku, kul string }{
		{"md5", "9faf3283884e92834ebc9847d8edd963", "526f5eed9fcce26f8964c2930787d82b"},
		{"sha", "9fb5cc0381497b3793528939ff788d5d79145211", "6695febc9288e36282235fc7151f128497b38f3f"},
	}
	for _, c := range cases {
		ku := PasswordToKey(c.auth, []byte("maplesyrup"))
		if !bytes.Equal(ku, unhex(t, c.ku)) {
			t.Fatalf("%s Ku % x", c.auth, ku)
		}
		if kul := LocalizeKey(c.auth, ku, engine); !bytes.Equal(kul, unhex(t, c.kul)) {
			t.Fatalf("%s Kul % x", c.auth, kul)
		}
	}
}

func TestKeyExtensionLengthsAndVariants(t *testing.T) {
	cache := KeyCache{}
	engine := unhex(t, "80001f8880e9bd0c1d12667a5100000000")
	base := model.User{Name: "u", Level: model.LevelPrivacy, Auth: "sha", AuthPassword: "authpass1", PrivPassword: "privpass1"}
	lengths := map[string]int{"des": 16, "aes128": 16, "aes192": 24, "aes256": 32, "aes192c": 24, "aes256c": 32}
	keys := map[string][]byte{}
	for priv, length := range lengths {
		user := base
		user.Priv = priv
		k, err := cache.Localize(user, engine)
		if err != nil {
			t.Fatal(err)
		}
		if len(k.PrivKey) != length {
			t.Fatalf("%s key is %d bytes", priv, len(k.PrivKey))
		}
		keys[priv] = k.PrivKey
	}
	// Both variants start with the localized key and differ only in the extension.
	if !bytes.Equal(keys["aes256"][:20], keys["aes256c"][:20]) || bytes.Equal(keys["aes256"], keys["aes256c"]) {
		t.Fatal("the Blumenthal and Reeder extensions should share the first 20 bytes only")
	}
}

func TestV3RoundTripAtEveryLevel(t *testing.T) {
	cache := KeyCache{}
	engine := unhex(t, "80001f8880e9bd0c1d12667a5100000000")
	pdu := PDU{Type: GetRequest, RequestID: 99, VarBinds: []VarBind{{OID: MustOID("1.3.6.1.2.1.1.5.0"), Value: Null}}}
	users := []model.User{{Name: "none", Level: model.LevelNoAuth}}
	for _, auth := range model.AuthProtocols {
		users = append(users, model.User{Name: "a-" + auth, Level: model.LevelAuth, Auth: auth, AuthPassword: "authpassword"})
		for _, priv := range model.PrivProtocols {
			users = append(users, model.User{Name: auth + "-" + priv, Level: model.LevelPrivacy, Auth: auth,
				AuthPassword: "authpassword", Priv: priv, PrivPassword: "privpassword"})
		}
	}
	for index, user := range users {
		keys, err := cache.Localize(user, engine)
		if err != nil {
			t.Fatal(err)
		}
		header := Header{MsgID: int32(index), Flags: LevelFlags(user.Level) | FlagReportable, EngineID: engine,
			Boots: 7, Time: 1234, UserName: user.Name, ContextEngineID: engine}
		packet, err := EncodeV3(header, pdu, keys, uint64(index)*7919)
		if err != nil {
			t.Fatalf("%s: %v", user.Name, err)
		}
		m, err := Decode(packet)
		if err != nil {
			t.Fatalf("%s: %v", user.Name, err)
		}
		if err := Unprotect(m, keys); err != nil {
			t.Fatalf("%s: %v", user.Name, err)
		}
		if m.Header.UserName != user.Name || m.Header.Level() != user.Level || m.PDU.RequestID != 99 ||
			m.PDU.VarBinds[0].OID.String() != "1.3.6.1.2.1.1.5.0" || !bytes.Equal(m.Header.ContextEngineID, engine) {
			t.Fatalf("%s came back as %+v", user.Name, m)
		}
		if user.Level == model.LevelNoAuth {
			continue
		}
		wrong := user
		wrong.AuthPassword = "otherpassword"
		other, _ := cache.Localize(wrong, engine)
		again, _ := Decode(packet)
		if err := Unprotect(again, other); !errors.Is(err, ErrWrongDigest) {
			t.Fatalf("%s: a wrong auth password gave %v", user.Name, err)
		}
		if user.Level == model.LevelPrivacy {
			wrong = user
			wrong.PrivPassword = "otherpassword"
			other, _ = cache.Localize(wrong, engine)
			again, _ = Decode(packet)
			if err := Unprotect(again, other); err == nil && again.PDU.RequestID == 99 {
				t.Fatalf("%s: a wrong privacy password decrypted the message", user.Name)
			}
		}
	}
}

func TestTamperedMessageFailsDigest(t *testing.T) {
	cache := KeyCache{}
	engine := unhex(t, "8000000001020304")
	user := model.User{Name: "u", Level: model.LevelAuth, Auth: "sha256", AuthPassword: "authpassword"}
	keys, _ := cache.Localize(user, engine)
	packet, err := EncodeV3(Header{Flags: FlagAuth, EngineID: engine, UserName: "u"},
		PDU{Type: GetRequest, VarBinds: []VarBind{{OID: MustOID("1.3.6.1.2.1.1.1.0"), Value: Null}}}, keys, 0)
	if err != nil {
		t.Fatal(err)
	}
	packet[len(packet)-3] ^= 1
	m, err := Decode(packet)
	if err == nil {
		err = Unprotect(m, keys)
	}
	if err == nil {
		t.Fatal("a changed message passed the digest check")
	}
}

func TestCheckUserMessages(t *testing.T) {
	short := model.User{Name: "u", Level: model.LevelAuth, Auth: "sha", AuthPassword: "short"}
	if CheckUser(short) == "" {
		t.Fatal("a 5-character password was accepted")
	}
	if CheckUser(model.User{Name: "u", Level: model.LevelNoAuth}) != "" {
		t.Fatal("noAuthNoPriv needs no password")
	}
}
