package events

import (
	"strconv"

	"studio-engine/internal/pyjson"
)

// encodePayload writes `, "key": value` for every payload key with json.dumps rules.
// plain is false when the payload overrides an envelope key (id, kind or timestamp): Python
// then keeps the envelope position with the payload value, handled by encodeOverridden.
func encodePayload(payload *pyjson.Object) ([]byte, bool, error) {
	var out []byte
	plain := true
	for _, key := range payload.Keys() {
		if key == "id" || key == "kind" || key == "timestamp" {
			plain = false
		}
		value, _ := payload.Get(key)
		encoded, err := pyjson.Dumps(value)
		if err != nil {
			return nil, false, err
		}
		out = append(out, ", "...)
		out = pyjson.AppendString(out, key)
		out = append(out, ": "...)
		out = append(out, encoded...)
	}
	return out, plain, nil
}

func envelope(id int64, kind string, timestamp float64, fragment []byte) []byte {
	out := make([]byte, 0, 64+len(kind)+len(fragment))
	out = append(out, `{"id": `...)
	out = strconv.AppendInt(out, id, 10)
	out = append(out, `, "kind": `...)
	out = pyjson.AppendString(out, kind)
	out = append(out, `, "timestamp": `...)
	out = pyjson.AppendFloat(out, timestamp)
	out = append(out, fragment...)
	return append(out, '}')
}

// encodeOverridden builds the event as Python's dict literal does and encodes it whole.
func encodeOverridden(event *Event) ([]byte, error) {
	object := pyjson.NewObject().Set("id", event.ID).Set("kind", event.Kind).Set("timestamp", event.Timestamp)
	for _, key := range event.Payload.Keys() {
		value, _ := event.Payload.Get(key)
		object.Set(key, value)
	}
	return pyjson.Dumps(object)
}
