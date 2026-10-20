package manager

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/snmp/agent"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
	"studio-engine/internal/testutil"
)

const deviceIP = "127.66.20.1"

func testDevice(t *testing.T, users ...model.User) (model.Device, *agent.Device) {
	t.Helper()
	cfg := model.Device{ID: "d1", Name: "Test switch", IP: deviceIP, Port: testutil.FreePort(t, deviceIP),
		V1: true, V2c: true, V3: true, ReadCommunity: "public", WriteCommunity: "private", Users: users,
		EngineID: "800000000501020304050607",
		Points: []model.Point{
			{OID: "1.3.6.1.2.1.1.1.0", Type: model.TypeOctets, Value: "ScadaBench test switch"},
			{OID: "1.3.6.1.2.1.1.2.0", Type: model.TypeOID, Value: "1.3.6.1.4.1.99999.1"},
			{OID: "1.3.6.1.2.1.1.3.0", Type: model.TypeTimeTicks, Value: "0", Behaviour: model.BehaviourUptime},
			{OID: "1.3.6.1.2.1.1.5.0", Type: model.TypeOctets, Value: "sw1", Writable: true},
			{OID: "1.3.6.1.2.1.2.2.1.8.1", Type: model.TypeInteger, Value: "1"},
			{OID: "1.3.6.1.2.1.2.2.1.8.2", Type: model.TypeInteger, Value: "2", Writable: true},
			{OID: "1.3.6.1.2.1.31.1.1.1.6.1", Type: model.TypeCounter64, Value: "18446744073709551000"},
		}}
	device, err := agent.Start(cfg, 5, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(device.Close)
	return cfg, device
}

func open(t *testing.T, target model.Target) *Session {
	t.Helper()
	target.TimeoutMS = 400
	session, err := Open(target)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(session.Close)
	return session
}

func TestV2cReadWalkAndSet(t *testing.T) {
	cfg, _ := testDevice(t)
	session := open(t, model.Target{Host: deviceIP, Port: cfg.Port, Version: "v2c", Community: "private"})
	values, err := session.Get([]proto.OID{proto.MustOID("1.3.6.1.2.1.1.1.0"), proto.MustOID("1.3.6.1.2.1.1.9.0")})
	if err != nil {
		t.Fatal(err)
	}
	if _, text, _ := proto.ValueText(values[0].Value); text != "ScadaBench test switch" || values[1].Value.Type != proto.TagNoSuchObj {
		t.Fatalf("get returned %+v", values)
	}
	walked, err := session.Walk(proto.MustOID("1.3.6.1"), nil)
	if err != nil || len(walked) != len(cfg.Points) {
		t.Fatalf("walk found %d values, %v", len(walked), err)
	}
	if _, err := session.Set([]proto.VarBind{{OID: proto.MustOID("1.3.6.1.2.1.1.5.0"),
		Value: proto.Value{Type: proto.TagOctetString, Bytes: []byte("renamed")}}}); err != nil {
		t.Fatal(err)
	}
	values, _ = session.Get([]proto.OID{proto.MustOID("1.3.6.1.2.1.1.5.0")})
	if string(values[0].Value.Bytes) != "renamed" {
		t.Fatalf("set did not stick: %+v", values)
	}
	_, err = session.Set([]proto.VarBind{{OID: proto.MustOID("1.3.6.1.2.1.1.1.0"),
		Value: proto.Value{Type: proto.TagOctetString, Bytes: []byte("x")}}})
	if err == nil || !strings.Contains(err.Error(), "not writable") {
		t.Fatalf("a read-only point was written: %v", err)
	}
	reader := open(t, model.Target{Host: deviceIP, Port: cfg.Port, Version: "v2c", Community: "public"})
	_, err = reader.Set([]proto.VarBind{{OID: proto.MustOID("1.3.6.1.2.1.1.5.0"),
		Value: proto.Value{Type: proto.TagOctetString, Bytes: []byte("x")}}})
	if err == nil || !strings.Contains(err.Error(), "no access") {
		t.Fatalf("the read community wrote: %v", err)
	}
	wrong := open(t, model.Target{Host: deviceIP, Port: cfg.Port, Version: "v2c", Community: "nope"})
	if _, err := wrong.Get([]proto.OID{proto.MustOID("1.3.6.1.2.1.1.1.0")}); err == nil ||
		!strings.Contains(err.Error(), "community") {
		t.Fatalf("a wrong community was answered: %v", err)
	}
}

func TestV1WalkSkipsCounter64(t *testing.T) {
	cfg, _ := testDevice(t)
	session := open(t, model.Target{Host: deviceIP, Port: cfg.Port, Version: "v1", Community: "public"})
	walked, err := session.Walk(proto.MustOID("1.3.6.1"), nil)
	if err != nil || len(walked) != len(cfg.Points)-1 {
		t.Fatalf("v1 walk found %d values, %v", len(walked), err)
	}
	if _, err := session.Get([]proto.OID{proto.MustOID("1.3.6.1.2.1.1.9.0")}); err == nil {
		t.Fatal("v1 get of a missing OID did not fail")
	}
	many, err := session.GetMany([]proto.OID{proto.MustOID("1.3.6.1.2.1.1.1.0"), proto.MustOID("1.3.6.1.2.1.1.9.0"),
		proto.MustOID("1.3.6.1.2.1.1.5.0")}, 20)
	if err != nil || len(many) != 3 || many[1].Value.Type != proto.TagNoSuchObj || string(many[2].Value.Bytes) != "sw1" {
		t.Fatalf("v1 GetMany returned %+v, %v", many, err)
	}
}

func TestV3EveryLevelAndRefusals(t *testing.T) {
	users := []model.User{
		{Name: "plain", Level: model.LevelNoAuth},
		{Name: "md5des", Level: model.LevelPrivacy, Auth: "md5", AuthPassword: "authpassword", Priv: "des", PrivPassword: "privpassword"},
		{Name: "shaaes", Level: model.LevelPrivacy, Auth: "sha", AuthPassword: "authpassword", Priv: "aes128", PrivPassword: "privpassword"},
		{Name: "sha256aes256", Level: model.LevelPrivacy, Auth: "sha256", AuthPassword: "authpassword", Priv: "aes256", PrivPassword: "privpassword"},
		{Name: "sha512aes256c", Level: model.LevelPrivacy, Auth: "sha512", AuthPassword: "authpassword", Priv: "aes256c", PrivPassword: "privpassword"},
		{Name: "sha224", Level: model.LevelAuth, Auth: "sha224", AuthPassword: "authpassword"},
	}
	cfg, _ := testDevice(t, users...)
	for _, user := range users {
		session := open(t, model.Target{Host: deviceIP, Port: cfg.Port, Version: "v3", User: user})
		walked, err := session.Walk(proto.MustOID("1.3.6.1"), nil)
		if err != nil || len(walked) != len(cfg.Points) {
			t.Fatalf("%s: walk found %d values, %v", user.Name, len(walked), err)
		}
	}
	wrong := users[2]
	wrong.AuthPassword = "otherpassword"
	session := open(t, model.Target{Host: deviceIP, Port: cfg.Port, Version: "v3", User: wrong})
	if _, err := session.Get([]proto.OID{proto.MustOID("1.3.6.1.2.1.1.1.0")}); err == nil ||
		!strings.Contains(err.Error(), "wrong authentication password") {
		t.Fatalf("a wrong password gave %v", err)
	}
	stranger := model.User{Name: "stranger", Level: model.LevelNoAuth}
	session = open(t, model.Target{Host: deviceIP, Port: cfg.Port, Version: "v3", User: stranger})
	if _, err := session.Get([]proto.OID{proto.MustOID("1.3.6.1.2.1.1.1.0")}); err == nil ||
		!strings.Contains(err.Error(), "does not know this user") {
		t.Fatalf("an unknown user gave %v", err)
	}
	lower := users[1]
	lower.Level = model.LevelAuth
	session = open(t, model.Target{Host: deviceIP, Port: cfg.Port, Version: "v3", User: lower})
	if _, err := session.Get([]proto.OID{proto.MustOID("1.3.6.1.2.1.1.1.0")}); err == nil ||
		!strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("a lower security level gave %v", err)
	}
}

func TestPollerReportsChanges(t *testing.T) {
	cfg, device := testDevice(t)
	session := open(t, model.Target{Host: deviceIP, Port: cfg.Port, Version: "v2c", Community: "public"})
	rounds := make(chan []any, 10)
	poller := StartPoller(session, []proto.OID{proto.MustOID("1.3.6.1.2.1.2.2.1.8.2")}, 50*time.Millisecond,
		func(values []any, problem string) { rounds <- values })
	defer poller.Stop()
	if first := <-rounds; len(first) != 1 {
		t.Fatalf("first round: %v", first)
	}
	if _, err := device.SetValue("1.3.6.1.2.1.2.2.1.8.2", "1", false); err != nil {
		t.Fatal(err)
	}
	select {
	case change := <-rounds:
		if change[0].(map[string]any)["value"] != "1" {
			t.Fatalf("change: %v", change)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the change was not reported")
	}
}

func TestPortClashIsExplained(t *testing.T) {
	cfg, _ := testDevice(t)
	_, err := agent.Start(cfg, 6, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("a second device on the same port gave %v", err)
	}
}
