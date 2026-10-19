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

// resolution is the routing outcome: a route, Flask's automatic OPTIONS answer (methods),
// a merge-slashes redirect, or a 404/405 error.
type resolution struct {
	route    *route
	options  []string
	redirect string
	err      error
}

// resolve finds the route for r and stores its path values on r. As Werkzeug, a path with
// "//" that matches nothing is retried with merged slashes and redirected when that matches.
func (rt *router) resolve(r *http.Request) resolution {
	method := r.Method
	if method == http.MethodHead || method == http.MethodOptions {
		method = http.MethodGet
	}
	allowed := map[string]bool{}
	best, values := rt.find(r.URL.Path, method, r.Method == http.MethodOptions, allowed)
	if best == nil {
		if merged := mergedPath(r.URL.Path); merged != "" {
			if found, _ := rt.find(merged, method, r.Method == http.MethodOptions, allowed); found != nil {
				return resolution{redirect: merged}
			}
		}
	}
	switch {
	case best != nil && r.Method == http.MethodOptions:
		return resolution{options: allowedMethods(allowed)}
	case best != nil:
		for name, value := range values {
			r.SetPathValue(name, value)
		}
		return resolution{route: best}
	case len(allowed) == 0:
		return resolution{err: ErrNotFound}
	}
	return resolution{err: ErrMethodNotAllowed}
}

// find returns the preferred route for path and method (any method when anyMethod, as
// every rule accepts Flask's automatic OPTIONS) and records the methods of matching rules.
func (rt *router) find(path, method string, anyMethod bool, allowed map[string]bool) (*route, map[string]string) {
	var best *route
	var bestValues map[string]string
	for index := range rt.routes {
		candidate := &rt.routes[index]
		values, ok := candidate.pattern.match(path)
		if !ok {
			continue
		}
		allowed[candidate.method] = true
		if (anyMethod || candidate.method == method) && (best == nil || candidate.pattern.before(best.pattern)) {
			best, bestValues = candidate, values
		}
	}
	return best, bestValues
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
