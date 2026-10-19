// Package native talks to ScadaBench's native worker processes (studio_ied, studio_live,
// studio_goose, studio_control) over their private pipes, exactly as the Python engine's
// runtime/worker_process.py and runtime/worker_protocol.py do: a 4-byte little-endian length
// followed by UTF-8 JSON of at most 16 MiB, with int, float and bool kept distinct (pyjson).
package native

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"studio-engine/internal/pyjson"
)

// MaxFrame is the largest frame either side accepts (worker_protocol.py MAX_FRAME).
const MaxFrame = 16 * 1024 * 1024

// ErrPipeClosed is Python's EOFError text when a worker closes its pipe mid-frame or between frames.
var ErrPipeClosed = errors.New("Native worker pipe closed")

// FrameError is a ValueError raised while framing (Python maps it to HTTP 400 when it escapes).
type FrameError struct{ Message string }

func (e *FrameError) Error() string { return e.Message }

// ReadFrame reads one frame and decodes it like json.loads.
func ReadFrame(r io.Reader) (any, error) {
	var header [4]byte
	if err := readExact(r, header[:]); err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint32(header[:])
	if length == 0 || length > MaxFrame {
		return nil, &FrameError{fmt.Sprintf("Invalid native frame length %d", length)}
	}
	payload := make([]byte, length)
	if err := readExact(r, payload); err != nil {
		return nil, err
	}
	return pyjson.Decode(payload)
}

func readExact(r io.Reader, buffer []byte) error {
	read := 0
	for read < len(buffer) {
		n, err := r.Read(buffer[read:])
		read += n
		if read == len(buffer) {
			return nil
		}
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
				return ErrPipeClosed
			}
			return err
		}
		if n == 0 {
			return ErrPipeClosed
		}
	}
	return nil
}

// EncodeFrame returns the bytes json.dumps(message, allow_nan=False, separators=(",", ":"))
// writes, behind the length header.
func EncodeFrame(message any) ([]byte, error) {
	if bad := nonFinite(message); bad != "" {
		return nil, &FrameError{"Out of range float values are not JSON compliant: " + bad}
	}
	payload, err := pyjson.Compact(message)
	if err != nil {
		return nil, err
	}
	if len(payload) > MaxFrame {
		return nil, &FrameError{"Native request exceeds frame limit"}
	}
	frame := make([]byte, 4, 4+len(payload))
	binary.LittleEndian.PutUint32(frame, uint32(len(payload)))
	return append(frame, payload...), nil
}

// nonFinite returns Python's repr of the first float json.dumps(allow_nan=False) refuses.
func nonFinite(value any) string {
	switch typed := value.(type) {
	case float64:
		switch {
		case math.IsNaN(typed):
			return "nan"
		case math.IsInf(typed, 1):
			return "inf"
		case math.IsInf(typed, -1):
			return "-inf"
		}
	case []any:
		for _, item := range typed {
			if bad := nonFinite(item); bad != "" {
				return bad
			}
		}
	case *pyjson.Object:
		for _, key := range typed.Keys() {
			item, _ := typed.Get(key)
			if bad := nonFinite(item); bad != "" {
				return bad
			}
		}
	}
	return ""
}
