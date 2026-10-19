package events

import (
	"math/big"
	"net/http"
	"strconv"
	"time"

	"studio-engine/internal/api"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/pycompat"
)

// Heartbeat is how long a stream waits for events before writing ": heartbeat".
const Heartbeat = 10 * time.Second

// Area serves GET /api/v1/events from a bus.
type Area struct {
	Bus       *Bus
	Heartbeat time.Duration
}

// Routes implements api.Area.
func (a Area) Routes() []api.Route {
	return []api.Route{{Method: http.MethodGet, Pattern: "/api/v1/events", Handler: httpapi.Handle(a.stream)}}
}

// stream is api/events.py stream_events: the cursor is int(Last-Event-ID header, else
// ?after=, else 0), at least 0; then "retry: 1000", and for ever the events after the
// cursor as "id: N\ndata: <json>\n\n" or ": heartbeat\n\n" after a quiet wait. The
// handler goroutine is this client's only writer; the bus lock is never held while writing.
func (a Area) stream(w http.ResponseWriter, r *http.Request) error {
	cursor, err := startCursor(r)
	if err != nil {
		return err
	}
	header := w.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return nil
	}
	flusher, _ := w.(http.Flusher)
	wait := a.Heartbeat
	if wait <= 0 {
		wait = Heartbeat
	}
	buffer := []byte("retry: 1000\n\n")
	done := r.Context().Done()
	for {
		if _, err := w.Write(buffer); err != nil {
			return nil
		}
		if flusher != nil {
			flusher.Flush()
		}
		items := a.Bus.After(cursor, wait, done)
		select {
		case <-done:
			return nil
		default:
		}
		buffer = buffer[:0]
		if len(items) == 0 {
			buffer = append(buffer, ": heartbeat\n\n"...)
		}
		for _, event := range items {
			cursor = event.ID
			buffer = append(buffer, "id: "...)
			buffer = strconv.AppendInt(buffer, event.ID, 10)
			buffer = append(buffer, "\ndata: "...)
			buffer = append(buffer, event.Data...)
			buffer = append(buffer, "\n\n"...)
		}
	}
}

// startCursor parses the reconnect cursor with Python's int() rules; a bad value is the
// ValueError Flask answered with 400.
func startCursor(r *http.Request) (int64, error) {
	text, found := httpapi.Header(r, "Last-Event-ID")
	if !found {
		if text, found = httpapi.Arg(r, "after"); !found {
			return 0, nil
		}
	}
	value, err := pycompat.ParseInt(text)
	if err != nil {
		return 0, err
	}
	switch number := value.(type) {
	case int64:
		return max(number, 0), nil
	case *big.Int:
		if number.Sign() > 0 {
			return 1<<63 - 1, nil
		}
	}
	return 0, nil
}
