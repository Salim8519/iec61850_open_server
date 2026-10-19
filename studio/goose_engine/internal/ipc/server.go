package ipc

import (
	"encoding/json"
	"fmt"
	"io"
)

// Handler answers one operation. Params is the full request frame.
type Handler func(params json.RawMessage) (any, error)

// Serve answers requests until "stop" or end of input, like the C++ workers.
// onStop runs before the stop reply so resources are released first.
func Serve(in io.Reader, out *Writer, handlers map[string]Handler, onStop func()) error {
	for {
		request, err := ReadRequest(in)
		if err == io.EOF {
			onStop()
			return nil
		}
		if err != nil {
			onStop()
			return err
		}
		if request.Op == "stop" {
			onStop()
			return out.Write(map[string]any{"id": request.ID, "ok": true})
		}
		result, err := dispatch(request, handlers)
		if err != nil {
			if writeErr := out.Write(map[string]any{"id": request.ID, "ok": false, "error": err.Error()}); writeErr != nil {
				return writeErr
			}
			continue
		}
		if err := out.Write(map[string]any{"id": request.ID, "ok": true, "result": result}); err != nil {
			return err
		}
	}
}

func dispatch(request Request, handlers map[string]Handler) (result any, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("internal error in %s: %v", request.Op, failure)
		}
	}()
	if request.Version != 1 {
		return nil, fmt.Errorf("protocol version must be 1")
	}
	handler, found := handlers[request.Op]
	if !found {
		return nil, fmt.Errorf("unknown operation %s", request.Op)
	}
	return handler(request.Params)
}
