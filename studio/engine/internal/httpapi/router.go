package httpapi

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"studio-engine/internal/api"
)

type route struct {
	method  string
	pattern pattern
	handler http.Handler
	polled  bool
}

// router matches like Werkzeug's URL map with Flask's automatic HEAD and OPTIONS: an
// unknown path is 404, a known path with another method 405 (Flask's handler answer has
// no Allow header), OPTIONS answers 200 with Allow, HEAD runs the GET handler.
type router struct {
	routes []route
}

func (rt *router) add(r api.Route) error {
	parsed, err := parsePattern(r.Pattern)
	if err != nil {
		return err
	}
	method := strings.ToUpper(r.Method)
	for _, existing := range rt.routes {
		if existing.method == method && existing.pattern.text == parsed.text {
			return fmt.Errorf("route %s %s is registered twice", method, r.Pattern)
		}
	}
	rt.routes = append(rt.routes, route{method: method, pattern: parsed, handler: r.Handler, polled: r.Polled})
	return nil
}

// resolve finds the route for r and stores its path values on r.
func (rt *router) resolve(r *http.Request) (*route, []string, error) {
	method := r.Method
	if method == http.MethodHead {
		method = http.MethodGet
	}
	var best *route
	var bestValues map[string]string
	allowed := map[string]bool{}
	for index := range rt.routes {
		candidate := &rt.routes[index]
		values, ok := candidate.pattern.match(r.URL.Path)
		if !ok {
			continue
		}
		allowed[candidate.method] = true
		if candidate.method == method && (best == nil || candidate.pattern.before(best.pattern)) {
			best, bestValues = candidate, values
		}
	}
	if best != nil {
		for name, value := range bestValues {
			r.SetPathValue(name, value)
		}
		return best, nil, nil
	}
	if len(allowed) == 0 {
		return nil, nil, ErrNotFound
	}
	methods := allowedMethods(allowed)
	if r.Method == http.MethodOptions {
		return nil, methods, nil
	}
	return nil, nil, ErrMethodNotAllowed
}

// allowedMethods lists the methods of the matching rules plus Flask's automatic HEAD
// (for GET rules) and OPTIONS, sorted.
func allowedMethods(allowed map[string]bool) []string {
	allowed[http.MethodOptions] = true
	if allowed[http.MethodGet] {
		allowed[http.MethodHead] = true
	}
	methods := make([]string, 0, len(allowed))
	for method := range allowed {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	return methods
}

// writeOptions is Flask's default OPTIONS answer: 200, empty HTML body, Allow header.
func writeOptions(w http.ResponseWriter, methods []string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	WriteBody(w, http.StatusOK, "text/html; charset=utf-8", nil)
}
