// Package model holds the SNMP data that the server, the studio_snmp worker, the importers
// and the project share (ADR-0092). It is plain data with JSON tags: no networking, no files.
package model

// Value types of a point, as the project, the worker frames, CSV files and the page name them.
const (
	TypeInteger   = "integer"   // INTEGER / Integer32
	TypeOctets    = "octets"    // OCTET STRING (text, or hex when Hex is set)
	TypeOID       = "oid"       // OBJECT IDENTIFIER, dotted numbers
	TypeIPAddress = "ipaddress" // IpAddress, a.b.c.d
	TypeCounter32 = "counter32"
	TypeGauge32   = "gauge32" // Gauge32 / Unsigned32
	TypeTimeTicks = "timeticks"
	TypeCounter64 = "counter64"
	TypeOpaque    = "opaque" // hex
	TypeNull      = "null"
)

// Types lists every point type in the order the page offers them.
var Types = []string{TypeInteger, TypeOctets, TypeOID, TypeIPAddress, TypeCounter32, TypeGauge32,
	TypeTimeTicks, TypeCounter64, TypeOpaque, TypeNull}

// Behaviours of a simulated point.
const (
	BehaviourFixed   = "fixed"   // the value stays until someone changes it
	BehaviourUptime  = "uptime"  // timeticks since the device started (hundredths of a second)
	BehaviourCounter = "counter" // grows by Rate per second from Value, wrapping like the type
)

// Point is one simulated OID, or one value read from a device. Value is always text: decimal
// for numbers (Counter64 does not fit a JSON number), dotted for OID and IpAddress, and the
// string itself for octets unless Hex is set ("00 1a ff"). Label is the user's own name for
// the value, shown instead of the MIB name; it never goes on the wire.
type Point struct {
	OID       string  `json:"oid"`
	Label     string  `json:"label,omitempty"`
	Type      string  `json:"type"`
	Value     string  `json:"value"`
	Hex       bool    `json:"hex,omitempty"`
	Behaviour string  `json:"behaviour,omitempty"`
	Rate      float64 `json:"rate,omitempty"`
	Writable  bool    `json:"writable,omitempty"`
}

// Security levels, authentication and privacy protocols of an SNMPv3 user.
const (
	LevelNoAuth  = "noAuthNoPriv"
	LevelAuth    = "authNoPriv"
	LevelPrivacy = "authPriv"
)

// AuthProtocols and PrivProtocols are the names the page and project use. "aes192c" and
// "aes256c" extend the key the way Cisco and Net-SNMP do (Reeder draft); "aes192"/"aes256"
// follow the Blumenthal draft.
var (
	AuthProtocols = []string{"md5", "sha", "sha224", "sha256", "sha384", "sha512"}
	PrivProtocols = []string{"des", "aes128", "aes192", "aes256", "aes192c", "aes256c"}
)

// User is one SNMPv3 user (USM). Passwords are stored in the project (owner decision,
// ADR-0092) and never written to logs.
type User struct {
	Name         string `json:"name"`
	Level        string `json:"level"`
	Auth         string `json:"auth,omitempty"`
	AuthPassword string `json:"auth_password,omitempty"`
	Priv         string `json:"priv,omitempty"`
	PrivPassword string `json:"priv_password,omitempty"`
}

// TrapTarget is where a simulated device sends its traps or informs.
type TrapTarget struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Version   string `json:"version"` // v1, v2c or v3
	Community string `json:"community,omitempty"`
	User      string `json:"user,omitempty"` // a user of the device, for v3
	Inform    bool   `json:"inform,omitempty"`
}

// Device is one simulated SNMP agent.
type Device struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Profile        string       `json:"profile"`
	IP             string       `json:"ip"`
	Port           int          `json:"port"`
	V1             bool         `json:"v1"`
	V2c            bool         `json:"v2c"`
	V3             bool         `json:"v3"`
	ReadCommunity  string       `json:"read_community"`
	WriteCommunity string       `json:"write_community"`
	Users          []User       `json:"users"`
	EngineID       string       `json:"engine_id"` // hex
	Traps          []TrapTarget `json:"traps"`
	Points         []Point      `json:"points"`
	LabIP          string       `json:"lab_ip,omitempty"` // its address on the lab network (Network setup)
	// LocalIP is set only in the worker request while the device runs on its lab address: the
	// device also answers on this 127.x address, so this PC reaches it and gets its traps.
	LocalIP string `json:"local_ip,omitempty"`
}

// Target is the device the reader talks to.
type Target struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Version   string `json:"version"` // v1, v2c or v3
	Community string `json:"community,omitempty"`
	User      User   `json:"user"`
	Context   string `json:"context,omitempty"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
	Retries   int    `json:"retries,omitempty"`
}
