package push

import (
	"context"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/pycompat"
	"studio-engine/internal/pyjson"
)

// stream is GET /api/v1/push?client=<tab id>&after=<event id>: the bus events after the
// cursor with the exact lines of /api/v1/events ("id: N\ndata: <json>\n\n", replay,
// resync, ": heartbeat\n\n" after a quiet Heartbeat) interleaved with topic frames
// ("event: push\ndata: <frame>\n\n", no id line, so EventSource's Last-Event-ID stays the
// bus cursor). The first frame is {"t":"hello","epoch":..,"client":..}; then the full frame
// of every subscribed topic.
func (a *Area) stream(w http.ResponseWriter, r *http.Request) error {
	id, err := clientID(r.URL.Query().Get("client"))
	if err != nil {
		return err
	}
	cursor, err := startCursor(r)
	if err != nil {
		return err
	}
	c, gen, wake, err := a.attach(id)
	if err != nil {
		return err
	}
	defer a.detach(c, gen)
	ctx := r.Context()
	stopWaking := context.AfterFunc(ctx, func() {
		select {
		case wake <- struct{}{}:
		default:
		}
	})
	defer stopWaking()
	header := w.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return nil
	}
	flusher, _ := w.(http.Flusher)
	buffer := append(make([]byte, 0, 4096), "retry: 1000\n\n"...)
	buffer = a.hello(buffer, id)
	gather := time.NewTimer(time.Hour)
	gather.Stop()
	defer gather.Stop()
	var frames [][]byte
	lastWrite, eventsAt := time.Now(), time.Time{}
	for {
		if len(buffer) > 0 {
			if _, err := w.Write(buffer); err != nil {
				return nil
			}
			if flusher != nil {
				flusher.Flush()
			}
			lastWrite, buffer = time.Now(), buffer[:0]
		}
		if !eventsAt.IsZero() {
			if rest := a.gather - time.Since(eventsAt); rest > 0 { // a burst: collect it
				gather.Reset(rest)
				select {
				case <-gather.C:
				case <-ctx.Done():
					return nil
				}
			}
		}
		items := a.bus.After(cursor, a.beat-time.Since(lastWrite), wake)
		if ctx.Err() != nil {
			return nil
		}
		var current bool
		if frames, current = c.drain(gen, frames[:0]); !current {
			return nil // a newer stream of this tab took over
		}
		eventsAt = time.Time{}
		if len(items) > 0 {
			eventsAt = time.Now()
		}
		for _, event := range items {
			cursor = event.ID
			buffer = append(buffer, "id: "...)
			buffer = strconv.AppendInt(buffer, event.ID, 10)
			buffer = append(buffer, "\ndata: "...)
			buffer = append(buffer, event.Data...)
			buffer = append(buffer, "\n\n"...)
		}
		for index, frame := range frames {
			buffer = append(append(append(buffer, "event: push\ndata: "...), frame...), "\n\n"...)
			frames[index] = nil
		}
		if len(buffer) == 0 && time.Since(lastWrite) >= a.beat {
			buffer = append(buffer, ": heartbeat\n\n"...)
		}
	}
}

// hello is the stream's first frame: the boot id lets a tab notice a server restart.
func (a *Area) hello(out []byte, id string) []byte {
	out = append(out, `event: push`+"\n"+`data: {"t":"hello","epoch":`...)
	out = pyjson.AppendString(out, a.epoch)
	out = pyjson.AppendString(append(out, `,"client":`...), id)
	return append(out, "}\n\n"...)
}

// clientID accepts 1 to 64 letters, digits, "-", "_" or ".".
func clientID(id string) (string, error) {
	if id == "" || len(id) > 64 {
		return "", apperr.ValueError("Send the tab id as client")
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return "", apperr.ValueError("Send the tab id as client")
		}
	}
	return id, nil
}

// startCursor is the /api/v1/events cursor rule: int(Last-Event-ID header, else ?after=,
// else 0) with Python's int() parsing, at least 0.
func startCursor(r *http.Request) (int64, error) {
	after, hasAfter, err := httpapi.Arg(r, "after")
	if err != nil {
		return 0, err
	}
	text, found := httpapi.Header(r, "Last-Event-ID")
	if !found {
		if text, found = after, hasAfter; !found {
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
