package labnet

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// Names the installer, the service and the app share.
const (
	ServiceName    = "ScadaBenchLabNet"
	ServiceDisplay = "ScadaBench lab network"
	PipeName       = `\\.\pipe\ScadaBenchLabNet`
	ProgramName    = "scadabench_labnet.exe"
	// ServiceProgram is the separate service program the installer copies to Program Files. The
	// installer never copies itself: Kaspersky stopped "install" as PDM:Trojan.Win32.Generic when
	// it copied its own file there and made it a service (ADR-0131).
	ServiceProgram = "scadabench_labsvc.exe"
	// oldServiceProgram is the copy installations before ADR-0131 registered as the service.
	oldServiceProgram = ProgramName
	// ParametersKey is the service's own registry key (only administrators can write it): the
	// card GUID the installer created.
	ParametersKey = `SYSTEM\CurrentControlSet\Services\` + ServiceName + `\Parameters`
	CardValue     = "CardGUID"
	// ApprovedKey holds the real-card addresses an administrator approved through the network
	// helper (ADR-0132): one DWORD value per address, named "{card GUID} address", holding the
	// prefix length. Only administrators can write it; the service changes nothing else on a
	// real card.
	ApprovedKey = ParametersKey + `\Approved`
)

// Operations the service accepts; anything else is refused.
const (
	OpStatus = "status"
	OpAdd    = "add"
	OpRemove = "remove"
)

// Request is one line the app sends on the pipe. Add and remove name one address (IP) or, from
// protocol 2 on, several (IPs, ADR-0128).
type Request struct {
	Op  string   `json:"op"`
	IP  string   `json:"ip,omitempty"`
	IPs []string `json:"ips,omitempty"`
	// Card names a real card (its GUID) instead of the lab card, from protocol 3 on (ADR-0132):
	// only addresses approved for that card may be added or removed there.
	Card string `json:"card,omitempty"`
}

// Wanted is the addresses the request names, in order.
func (r Request) Wanted() []string {
	if len(r.IPs) > 0 {
		return r.IPs
	}
	return []string{r.IP}
}

// Reply is the service's one-line answer. Addresses are the card's IPv4 addresses after the
// request; Message is plain English when OK is false. Failed gives the reason for each address
// of the request that was not added or removed.
type Reply struct {
	OK        bool              `json:"ok"`
	Message   string            `json:"message,omitempty"`
	Card      string            `json:"card,omitempty"`
	Addresses []string          `json:"addresses"`
	Failed    map[string]string `json:"failed,omitempty"`
	Version   string            `json:"version,omitempty"`
}

// ApprovalName is the registry value name of one approved address under ApprovedKey; the network
// helper writes it, the service reads it (ADR-0132).
func ApprovalName(cardGUID, ip string) string { return strings.ToUpper(cardGUID) + " " + ip }

// RealCards says whether a service that answered with this version puts approved addresses on
// real cards (protocol 3, ADR-0132).
func RealCards(version string) bool {
	number, err := strconv.Atoi(version)
	return err == nil && number >= 3
}

// Batched says whether a service that answered with this version takes several addresses in one
// request (protocol 2, ADR-0128). Protocol 1 takes one.
func Batched(version string) bool {
	number, err := strconv.Atoi(version)
	return err == nil && number >= 2
}

const (
	// maxRequest is the longest request line the service reads.
	maxRequest = 8 * 1024
	// MaxAddresses is the most addresses one request may name.
	MaxAddresses = 256
)

// DecodeRequest reads one request; unknown operations and fields are refused.
func DecodeRequest(line []byte) (Request, error) {
	var request Request
	if len(line) == 0 || len(line) > maxRequest {
		return request, errors.New("the request is empty or too long")
	}
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, errors.New("the request is not valid")
	}
	if request.Card != "" && !isGUID(request.Card) {
		return request, errors.New("the card must be a GUID")
	}
	switch request.Op {
	case OpStatus:
		if request.IP != "" || request.IPs != nil {
			return request, errors.New("status takes no address")
		}
	case OpAdd, OpRemove:
		if request.IPs != nil && (request.IP != "" || len(request.IPs) == 0 || len(request.IPs) > MaxAddresses) {
			return request, errors.New("name one address, or 1 to 256 addresses")
		}
	default:
		return request, errors.New("only status, add and remove are allowed")
	}
	return request, nil
}

// isGUID accepts "{XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}" (hex digits, either case).
func isGUID(text string) bool {
	if len(text) != 38 || text[0] != '{' || text[37] != '}' {
		return false
	}
	for index, char := range text[1:37] {
		switch {
		case index == 8 || index == 13 || index == 18 || index == 23:
			if char != '-' {
				return false
			}
		case !strings.ContainsRune("0123456789abcdefABCDEF", char):
			return false
		}
	}
	return true
}

// Pinged is one echo: whether it answered, the round trip in milliseconds and the TTL.
type Pinged struct {
	OK  bool   `json:"ok"`
	MS  uint32 `json:"ms"`
	TTL uint8  `json:"ttl"`
}

// ErrNoCard is the answer while the lab card is missing.
var ErrNoCard = errors.New("the ScadaBench Lab card is missing; use Repair on the lab network page")
