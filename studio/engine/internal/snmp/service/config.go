package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/netip"
	"strconv"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/profiles"
)

// Roles of the tool's header switch.
const (
	RoleSimulate = "simulate"
	RoleRead     = "read"
)

// Config is project.snmp.
type Config struct {
	Role      string         `json:"role"`
	AutoStart bool           `json:"auto_start"`
	Devices   []model.Device `json:"devices"`
	Reader    Reader         `json:"reader"`
}

// Reader is the Read devices settings.
type Reader struct {
	Target    model.Target `json:"target"`
	EveryMS   int          `json:"every_ms"`
	Watched   []string     `json:"watched"`
	TrapIP    string       `json:"trap_ip"`
	TrapPort  int          `json:"trap_port"`
	TrapUsers []model.User `json:"trap_users"`
	TrapsOn   bool         `json:"traps_on"`
	EngineID  string       `json:"engine_id"`
}

// Defaults that make a new device and the reader work together at once (owner: smart defaults).
const (
	DefaultPollMS  = 1000
	firstAddress   = "127.60.0.1"
	defaultUser    = "scadabench"
	defaultAuthKey = "scadabench-auth"
	defaultPrivKey = "scadabench-priv"
)

// DefaultUser is the v3 user new devices and the reader start with.
func DefaultUser() model.User {
	return model.User{Name: defaultUser, Level: model.LevelPrivacy, Auth: "sha256", AuthPassword: defaultAuthKey,
		Priv: "aes128", PrivPassword: defaultPrivKey}
}

// DefaultConfig is the section of a project that never used SNMP.
func DefaultConfig() Config {
	return Config{Role: RoleSimulate, Devices: []model.Device{}, Reader: Reader{
		Target:  model.Target{Host: firstAddress, Port: 161, Version: "v2c", Community: "public", User: DefaultUser()},
		EveryMS: DefaultPollMS, Watched: []string{}, TrapIP: "127.0.0.1", TrapPort: 162,
		TrapUsers: []model.User{DefaultUser()}, EngineID: NewEngineID()}}
}

// Load reads the stored section; anything missing takes its default, so an old or partly
// written section still opens. Points and devices are checked when they are used.
func Load(section *pyjson.Object) Config {
	config := DefaultConfig()
	if section == nil {
		return config
	}
	var stored Config
	if err := FromPy(section, &stored); err != nil {
		return config
	}
	if stored.Role == RoleRead {
		config.Role = RoleRead
	}
	config.AutoStart = stored.AutoStart
	if stored.Devices != nil {
		config.Devices = stored.Devices
	}
	for index := range config.Devices {
		device := &config.Devices[index]
		if device.Users == nil {
			device.Users = []model.User{}
		}
		if device.Traps == nil {
			device.Traps = []model.TrapTarget{}
		}
		if device.Points == nil {
			device.Points = []model.Point{}
		}
	}
	reader := stored.Reader
	if reader.Target.Host != "" {
		config.Reader.Target = reader.Target
	}
	if reader.EveryMS > 0 {
		config.Reader.EveryMS = reader.EveryMS
	}
	if reader.Watched != nil {
		config.Reader.Watched = reader.Watched
	}
	if reader.TrapIP != "" {
		config.Reader.TrapIP, config.Reader.TrapPort = reader.TrapIP, reader.TrapPort
	}
	if reader.TrapUsers != nil {
		config.Reader.TrapUsers = reader.TrapUsers
	}
	config.Reader.TrapsOn = reader.TrapsOn
	if len(reader.EngineID) >= 10 {
		config.Reader.EngineID = reader.EngineID
	}
	return config
}

// Section is the config as stored in the project.
func (c Config) Section() (*pyjson.Object, error) { return ToObject(c) }

// NewEngineID is a random RFC 3411 engine ID in the octets format (0x80 ... 05 + 8 bytes).
func NewEngineID() string {
	random := make([]byte, 8)
	_, _ = rand.Read(random)
	return "8000000005" + hex.EncodeToString(random)
}

// NewDevice is a device of profile with a free id, name and address.
func NewDevice(profile string, existing []model.Device) model.Device {
	ids, names, addresses := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, device := range existing {
		ids[device.ID], names[device.Name] = true, true
		addresses[device.IP+":"+strconv.Itoa(device.Port)] = true
	}
	id := ""
	for number := 1; id == "" || ids[id]; number++ {
		id = "snmp" + strconv.Itoa(number)
	}
	base := profiles.Names[profile]
	if base == "" {
		base = profiles.Names[profiles.Empty]
	}
	name := ""
	for number := 1; name == "" || names[name]; number++ {
		name = base + " " + strconv.Itoa(number)
	}
	address := netip.MustParseAddr(firstAddress)
	for addresses[address.String()+":161"] {
		address = address.Next()
	}
	points := profiles.Points(profile, name)
	if points == nil {
		points = []model.Point{}
	}
	return model.Device{ID: id, Name: name, Profile: profile, IP: address.String(), Port: 161,
		V1: true, V2c: true, V3: true, ReadCommunity: "public", WriteCommunity: "private",
		Users: []model.User{DefaultUser()}, EngineID: NewEngineID(),
		Traps:  []model.TrapTarget{{Host: "127.0.0.1", Port: 162, Version: "v2c", Community: "public"}},
		Points: points}
}

// findDevice returns the index of the device with id.
func (c Config) findDevice(id string) (int, error) {
	for index, device := range c.Devices {
		if device.ID == id {
			return index, nil
		}
	}
	return -1, fmt.Errorf("there is no SNMP device %q", id)
}
