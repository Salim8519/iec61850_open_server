package labnet

import (
	"bytes"
	"encoding/json"
	"errors"
)

// Names the installer, the service and the app share.
const (
	ServiceName    = "ScadaBenchLabNet"
	ServiceDisplay = "ScadaBench lab network"
	PipeName       = `\\.\pipe\ScadaBenchLabNet`
	ProgramName    = "scadabench_labnet.exe"
	// ParametersKey is the service's own registry key (only administrators can write it): the
	// card GUID the installer created.
	ParametersKey = `SYSTEM\CurrentControlSet\Services\` + ServiceName + `\Parameters`
	CardValue     = "CardGUID"
)

// Operations the service accepts; anything else is refused.
const (
	OpStatus = "status"
	OpAdd    = "add"
	OpRemove = "remove"
)

// Request is one line the app sends on the pipe.
type Request struct {
	Op string `json:"op"`
	IP string `json:"ip,omitempty"`
}

// Reply is the service's one-line answer. Addresses are the card's IPv4 addresses after the
// request; Message is plain English when OK is false.
type Reply struct {
	OK        bool     `json:"ok"`
	Message   string   `json:"message,omitempty"`
	Card      string   `json:"card,omitempty"`
	Addresses []string `json:"addresses"`
	Version   string   `json:"version,omitempty"`
}

// maxRequest is the longest request line the service reads.
const maxRequest = 512

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
	switch request.Op {
	case OpStatus:
		if request.IP != "" {
			return request, errors.New("status takes no address")
		}
	case OpAdd, OpRemove:
	default:
		return request, errors.New("only status, add and remove are allowed")
	}
	return request, nil
}

// Pinged is one echo: whether it answered, the round trip in milliseconds and the TTL.
type Pinged struct {
	OK  bool   `json:"ok"`
	MS  uint32 `json:"ms"`
	TTL uint8  `json:"ttl"`
}

// ErrNoCard is the answer while the lab card is missing.
var ErrNoCard = errors.New("the ScadaBench Lab card is missing; use Repair on the lab network page")
