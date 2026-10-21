package push

import (
	"bytes"
	"context"
	"log"
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/httpapi"
)

// RouteIndex holds the GET handlers of the composed areas by pattern, so a topic's full
// frame is produced by the very handler that answers its polled GET: the bytes are equal
// by construction, including error answers.
type RouteIndex map[string]http.Handler

// IndexRoutes indexes the GET routes of every area (later areas do not replace earlier ones;
// httpapi refuses duplicates anyway).
func IndexRoutes(areas ...[]api.Route) RouteIndex {
	index := RouteIndex{}
	for _, routes := range areas {
		for _, route := range routes {
			if route.Method == http.MethodGet {
				if _, found := index[route.Pattern]; !found {
					index[route.Pattern] = route.Handler
				}
			}
		}
	}
	return index
}

// Getter returns the in-process GET of pattern (for example "/api/v1/health"), or false.
func (index RouteIndex) Getter(pattern string) (Getter, bool) {
	handler, found := index[pattern]
	if !found {
		return nil, false
	}
	return HandlerGetter(pattern, handler), true
}

// HandlerGetter calls handler with a GET of path?query and returns status and body. A
// panic answers like Flask's 500 (the server's own recovery is not on this path).
func HandlerGetter(path string, handler http.Handler) Getter {
	return func(query string) (status int, body []byte) {
		target := path
		if query != "" {
			target += "?" + query
		}
		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
		if err != nil {
			return internalError()
		}
		request.RemoteAddr = "127.0.0.1:0"
		recorder := &recorder{header: http.Header{}}
		defer func() {
			if failure := recover(); failure != nil {
				log.Printf("push: GET %s panicked: %v", path, failure)
				status, body = internalError()
			}
		}()
		handler.ServeHTTP(recorder, request)
		if recorder.status == 0 {
			recorder.status = http.StatusOK
		}
		return recorder.status, recorder.body.Bytes()
	}
}

func internalError() (int, []byte) {
	recorder := &recorder{header: http.Header{}}
	httpapi.WriteError(recorder, errInternal{})
	return recorder.status, recorder.body.Bytes()
}

type errInternal struct{}

func (errInternal) Error() string { return "internal" }

// recorder is the smallest http.ResponseWriter: status and body only.
type recorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *recorder) Header() http.Header { return r.header }

func (r *recorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}

func (r *recorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}
