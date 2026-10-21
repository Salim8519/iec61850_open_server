package licensing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
)

// trialVectors is generated/trial.json (make_trial.py).
type trialVectors struct {
	Parts []struct {
		GUID   *string           `json:"guid"`
		Serial *uint32           `json:"serial"`
		UUID   string            `json:"uuid"`
		Junk   bool              `json:"junk"`
		Swap   string            `json:"swap"`
		Parts  map[string]string `json:"parts"`
	} `json:"parts"`
	Stamps []struct {
		outcome
		Stamp  string `json:"stamp"`
		NoKeys bool   `json:"no_keys"`
	} `json:"stamps"`
	Policy []struct {
		Status, Applied, Host      string
		Confirmed, Suspicious, Now int64
		Due, Missing               bool
		Latest                     int64
	} `json:"policy"`
	Clock []struct {
		Name     string            `json:"name"`
		Files    map[string]string `json:"files"`
		Registry map[string]string `json:"registry"`
		State    *string           `json:"state"`
		ReadAll  []string          `json:"read_all"`
		ReadInts []string          `json:"read_ints"`
		Places   int               `json:"places"`
		Written  struct {
			Files    map[string]string `json:"files"`
			Registry map[string]string `json:"registry"`
		} `json:"written"`
	} `json:"clock"`
	SupportURL string            `json:"support_url"`
	Texts      map[string]string `json:"texts"`
}

// memoryRegistry is license_fakes.FakeRegistry: HKCU\Software\ScadaBench in memory.
type memoryRegistry map[string]string

func (r memoryRegistry) Get(name string) (string, bool) { value, ok := r[name]; return value, ok }
func (r memoryRegistry) Set(name, text string)          { r[name] = text }

func loadTrial(t *testing.T) trialVectors {
	var vectors trialVectors
	readVectors(t, "trial.json", &vectors)
	return vectors
}

func TestTrialPartsMatchPython(t *testing.T) {
	for _, c := range loadTrial(t).Parts {
		parts := ComposeTrialParts(RawTrialParts{MachineGUID: c.GUID, VolumeSerial: c.Serial, FirmwareUUID: c.UUID})
		if IsJunkFirmware(c.UUID) != c.Junk || SwapUUIDFields(c.UUID) != c.Swap {
			t.Errorf("%q: junk %v swap %q, want %v %q", c.UUID, IsJunkFirmware(c.UUID), SwapUUIDFields(c.UUID), c.Junk, c.Swap)
		}
		if keys := fmt.Sprint(parts.Keys()); keys != "[guid volume firmware]" {
			t.Errorf("part order %s", keys)
		}
		for kind, want := range c.Parts {
			if got, _ := parts.Get(kind); got != want {
				t.Errorf("%v %v %q: %s = %v, want %s", c.GUID, c.Serial, c.UUID, kind, got, want)
			}
		}
	}
}

func TestServerTimeStampsMatchPython(t *testing.T) {
	for index, c := range loadTrial(t).Stamps {
		keys := testKeys(t)
		if c.NoKeys {
			keys = map[string][]byte{}
		}
		moment, ok, err := VerifyServerTime(mustDecode(t, c.Stamp), keys)
		var want *int64
		json.Unmarshal(c.OK, &want)
		switch {
		case err != nil:
			t.Errorf("case %d: error %v", index, err)
		case (want == nil) == ok || (want != nil && *want != moment):
			t.Errorf("case %d %s: got %d %v, want %v", index, c.Stamp, moment, ok, c.OK)
		}
	}
}

func TestTrialPolicyMatchesPython(t *testing.T) {
	for index, c := range loadTrial(t).Policy {
		status := statusFrom(t, c.Status)
		applied := ApplyTrial(status, c.Confirmed, c.Suspicious, c.Now, c.Host)
		if got := compactText(t, applied.Object()); got != c.Applied {
			t.Errorf("case %d: applied %s\nwant %s", index, got, c.Applied)
		}
		if CheckDue(c.Confirmed, c.Suspicious, c.Now) != c.Due || LatestSeen(c.Now-86400, c.Confirmed) != c.Latest {
			t.Errorf("case %d: due/latest differ", index)
		}
		copies := 0
		if index/10%2 == 1 {
			copies = 1
		}
		if AnchorMissing(copies, c.Confirmed) != c.Missing {
			t.Errorf("case %d: anchor_missing differs", index)
		}
	}
}

// statusFrom rebuilds a Status from LicenseStatus.as_dict() JSON.
func statusFrom(t *testing.T, text string) Status {
	object := mustDecode(t, text).(*pyjson.Object)
	field := func(key string) any { value, _ := object.Get(key); return value }
	return Status{Licensed: field("licensed").(bool), Mode: field("mode").(string), Plan: field("plan"),
		DaysRemaining: field("days_remaining"), ExpiresAt: field("expires_at"), Reason: field("reason"),
		LicenseID: field("license_id"), Email: field("email"),
		OnlineCheckRequired: field("online_check_required").(bool)}
}

func TestClockValuesMatchPython(t *testing.T) {
	for index, c := range loadTrial(t).Clock {
		home := t.TempDir()
		for name, data := range c.Files {
			os.WriteFile(filepath.Join(home, name), fromB64(t, data), 0o666)
		}
		writeOrRemove(t, filepath.Join(home, "license_state.json"), c.State)
		var registry Registry
		memory := memoryRegistry{}
		if c.Registry != nil {
			for name, value := range c.Registry {
				memory[name] = value
			}
			registry = memory
		}
		value := NewClockValue(c.Name, home, registry)
		ints := []string{}
		for _, number := range value.ReadInts() {
			ints = append(ints, number.String())
		}
		if fmt.Sprint(value.ReadAll()) != fmt.Sprint(c.ReadAll) || fmt.Sprint(ints) != fmt.Sprint(c.ReadInts) ||
			value.Places() != c.Places {
			t.Errorf("case %d: read %q %v %d, want %q %v %d", index, value.ReadAll(), ints, value.Places(),
				c.ReadAll, c.ReadInts, c.Places)
		}
		value.Write("1760000500")
		if problem := sameFiles(t, snapshotFiles(t, home), c.Written.Files); problem != "" {
			t.Errorf("case %d: %s", index, problem)
		}
		if c.Registry != nil && fmt.Sprint(map[string]string(memory)) != fmt.Sprint(c.Written.Registry) {
			t.Errorf("case %d: registry %v, want %v", index, memory, c.Written.Registry)
		}
	}
}

func TestTrialTextsMatchPython(t *testing.T) {
	v := loadTrial(t)
	for code, want := range v.Texts {
		got := ServerErrors[code]
		switch code {
		case "needs_online":
			got = NeedsOnlineText("h.example")
		case "clock_rollback":
			got = ClockRollback
		}
		if got != want {
			t.Errorf("%s = %q\nwant %q", code, got, want)
		}
	}
	if SupportURL != v.SupportURL || len(ServerErrors) != len(v.Texts)-2 {
		t.Errorf("support URL %s or texts differ", SupportURL)
	}
}
