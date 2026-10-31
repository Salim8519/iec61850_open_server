package netcfg

import (
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
)

func writeRequest(t *testing.T, folder, name, fingerprint, ip string, ok bool) string {
	request := filepath.Join(folder, name+".json")
	item := pyjson.NewObject().Set("action", "add").Set("adapter_guid", "{ETH}").Set("adapter_index", 15).
		Set("adapter_name", "Ethernet").Set("for", "P545").Set("ip", ip).Set("kind", "alias").Set("owned", true).
		Set("prefix", 24)
	payload, _ := pyjson.Dumps(pyjson.NewObject().Set("action", "apply").Set("fingerprint", fingerprint).
		Set("items", []any{item}))
	key, _ := ItemKey(item)
	result, _ := pyjson.Dumps(pyjson.NewObject().Set("refused", false).Set("results",
		[]any{pyjson.NewObject().Set("key", key).Set("ok", ok).Set("message", "")}))
	os.WriteFile(request, payload, 0o666)
	os.WriteFile(filepath.Join(folder, name+".result.json"), result, 0o666)
	return request
}

func TestALateHelperResultIsJournaledOnce(t *testing.T) {
	directory := t.TempDir()
	folder := filepath.Join(directory, "network", "requests")
	os.MkdirAll(folder, 0o777)
	service := &NetworkService{Directory: directory, Journal: NewJournal(directory)}
	writeRequest(t, folder, "apply-late", "f-late", "192.168.230.21", true)
	writeRequest(t, folder, "apply-failed", "f-failed", "192.168.230.24", false)
	waiting := writeRequest(t, folder, "apply-waiting", "f-waiting", "192.168.230.29", true)
	os.WriteFile(filepath.Join(folder, "lab-install-x.result.json"), []byte(`{"ok":false}`), 0o666)
	waitFor(waiting, true)
	defer waitFor(waiting, false)
	for round := 0; round < 2; round++ {
		ours, err := service.OurAddresses()
		if err != nil {
			t.Fatal(err)
		}
		if len(ours) != 1 || ours["192.168.230.21"][0] != "{ETH}" {
			t.Fatalf("round %d: ours %v", round, ours)
		}
	}
	records, _ := service.Journal.Records()
	if len(records) != 2 || get(records[0], "event") != "failed" && get(records[1], "event") != "failed" {
		t.Fatalf("journal %v", records)
	}
}

func TestAJournaledRequestIsNotJournaledAgain(t *testing.T) {
	directory := t.TempDir()
	folder := filepath.Join(directory, "network", "requests")
	os.MkdirAll(folder, 0o777)
	service := &NetworkService{Directory: directory, Journal: NewJournal(directory)}
	writeRequest(t, folder, "apply-done", "f-done", "10.0.0.5", true)
	service.Journal.Append("applied", pyjson.NewObject().Set("kind", "alias").Set("owned", true).
		Set("adapter_guid", "{ETH}").Set("ip", "10.0.0.5"), "f-done", "")
	if err := service.catchUp(); err != nil {
		t.Fatal(err)
	}
	if records, _ := service.Journal.Records(); len(records) != 1 {
		t.Fatalf("journal %v", records)
	}
}
