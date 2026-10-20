package manager

import (
	"encoding/hex"
	"testing"
	"time"

	"studio-engine/internal/snmp/agent"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/testutil"
)

const receiverIP = "127.66.20.2"

func TestTrapsAndInformsInEveryVersion(t *testing.T) {
	user := model.User{Name: "trapper", Level: model.LevelPrivacy, Auth: "sha256", AuthPassword: "authpassword",
		Priv: "aes128", PrivPassword: "privpassword"}
	events := make(chan map[string]any, 20)
	port := testutil.FreePort(t, receiverIP)
	engine, _ := hex.DecodeString("8000000005aabbccdd")
	receiver, err := Listen(receiverIP, port, []model.User{user}, engine, func(event map[string]any) { events <- event })
	if err != nil {
		t.Fatal(err)
	}
	defer receiver.Close()
	targets := []model.TrapTarget{
		{Host: receiverIP, Port: port, Version: "v1", Community: "public"},
		{Host: receiverIP, Port: port, Version: "v2c", Community: "public"},
		{Host: receiverIP, Port: port, Version: "v2c", Community: "public", Inform: true},
		{Host: receiverIP, Port: port, Version: "v3", User: "trapper"},
		{Host: receiverIP, Port: port, Version: "v3", User: "trapper", Inform: true},
	}
	cfg := model.Device{ID: "d1", Name: "GPS", IP: deviceIP, Port: testutil.FreePort(t, deviceIP), V2c: true, V3: true,
		ReadCommunity: "public", EngineID: "800000000511223344", Users: []model.User{user}, Traps: targets,
		Points: []model.Point{{OID: "1.3.6.1.2.1.1.2.0", Type: model.TypeOID, Value: "1.3.6.1.4.1.99999.2"}}}
	device, err := agent.Start(cfg, 9, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	results, err := device.SendTrap(agent.Trap{Notification: "1.3.6.1.6.3.1.1.5.3",
		VarBinds: []model.Point{{OID: "1.3.6.1.2.1.2.2.1.1.3", Type: model.TypeInteger, Value: "3"}}})
	if err != nil {
		t.Fatal(err)
	}
	for index, result := range results {
		if !result.OK {
			t.Fatalf("target %d (%s): %s", index, result.Target, result.Text)
		}
	}
	if results[2].Text != "acknowledged" || results[4].Text != "acknowledged" {
		t.Fatalf("informs were not acknowledged: %+v", results)
	}
	seen := map[string]int{}
	deadline := time.After(3 * time.Second)
	for len(seen) < 5 {
		select {
		case event := <-events:
			if problem, bad := event["problem"]; bad {
				t.Fatalf("receiver problem: %v", problem)
			}
			if event["notification"] != "1.3.6.1.6.3.1.1.5.3" {
				t.Fatalf("notification %v in %v", event["notification"], event)
			}
			rows := event["varbinds"].([]any)
			if len(rows) != 1 || rows[0].(map[string]any)["value"] != "3" {
				t.Fatalf("varbinds %v", rows)
			}
			seen[event["version"].(string)+" "+event["kind"].(string)]++
		case <-deadline:
			t.Fatalf("only received %v", seen)
		}
	}
	for _, want := range []string{"v1 trap", "v2c trap", "v2c inform", "v3 trap", "v3 inform"} {
		if seen[want] == 0 {
			t.Fatalf("missing %s in %v", want, seen)
		}
	}
}

func TestTrapFromUnknownUserIsReported(t *testing.T) {
	events := make(chan map[string]any, 5)
	port := testutil.FreePort(t, receiverIP)
	receiver, err := Listen(receiverIP, port, nil, []byte{0x80, 0, 0, 0, 5, 1}, func(e map[string]any) { events <- e })
	if err != nil {
		t.Fatal(err)
	}
	defer receiver.Close()
	user := model.User{Name: "nobody", Level: model.LevelAuth, Auth: "sha", AuthPassword: "authpassword"}
	cfg := model.Device{ID: "d2", Name: "X", IP: deviceIP, Port: testutil.FreePort(t, deviceIP), V3: true,
		EngineID: "800000000599", Users: []model.User{user},
		Traps: []model.TrapTarget{{Host: receiverIP, Port: port, Version: "v3", User: "nobody"}}}
	device, err := agent.Start(cfg, 1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	if _, err := device.SendTrap(agent.Trap{Notification: "1.3.6.1.6.3.1.1.5.1"}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event["problem"] == nil {
			t.Fatalf("a trap from an unknown user was accepted: %v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
	}
}
