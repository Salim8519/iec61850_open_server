package toolsapi

import (
	"net/http"
	"strings"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pycsv"
)

// items is the items() check: a list of objects whose ref and fc are text.
func items(data *pyjson.Object) ([]*pyjson.Object, error) {
	refused := apperr.ValueError("Send items as a list of {ref, fc}")
	entries, ok := value(data, "items", nil).([]any)
	if !ok {
		return nil, refused
	}
	result := make([]*pyjson.Object, len(entries))
	for index, raw := range entries {
		item, isObject := raw.(*pyjson.Object)
		if !isObject {
			return nil, refused
		}
		_, refText := value(item, "ref", nil).(string)
		_, fcText := value(item, "fc", nil).(string)
		if !refText || !fcText {
			return nil, refused
		}
		result[index] = item
	}
	return result, nil
}

func (a *Area) explorerRoutes() []api.Route {
	service := a.deps.Explorer
	return []api.Route{
		route("GET", "/tools/explorer", true, func(*http.Request) (any, error) { return service.Warm(), nil }),
		route("POST", "/tools/explorer/connect", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			return service.Connect(value(data, "host", ""), value(data, "port", int64(102)))
		}),
		route("POST", "/tools/explorer/disconnect", false, func(*http.Request) (any, error) {
			return service.Disconnect(), nil
		}),
		route("GET", "/tools/explorer/devices", false, func(*http.Request) (any, error) {
			names, err := service.Devices()
			return named(names, err)
		}),
		route("GET", "/tools/explorer/nodes", false, func(r *http.Request) (any, error) {
			device, _ := query(r, "device")
			names, err := service.Nodes(device)
			return named(names, err)
		}),
		route("GET", "/tools/explorer/node", false, func(r *http.Request) (any, error) {
			reference, _ := query(r, "ref")
			return service.Node(reference)
		}),
		route("POST", "/tools/explorer/read", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			list, err := items(data)
			if err != nil {
				return nil, err
			}
			values, err := service.Read(list)
			if err != nil {
				return nil, err
			}
			return pyjson.NewObject().Set("values", values), nil
		}),
		route("PUT", "/tools/explorer/watch", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			list, err := items(data)
			if err != nil {
				return nil, err
			}
			return service.SetWatch(list, value(data, "interval_ms", nil))
		}),
		route("POST", "/tools/explorer/polling", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			if truthy(value(data, "on", nil)) {
				return service.StartPolling()
			}
			return service.StopPolling(), nil
		}),
		route("GET", "/tools/explorer/watch.csv", false, func(*http.Request) (any, error) {
			return watchCSV(service.Status()), nil
		}),
		route("POST", "/tools/explorer/operate", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			reference, isText := value(data, "ref", nil).(string)
			if !isText {
				return nil, apperr.ValueError("Send the control reference, for example LD0/XCBR1.Pos")
			}
			return service.Operate(reference, value(data, "value", nil), value(data, "options", nil),
				value(data, "confirm", nil) == true)
		}),
	}
}

func named(names any, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("names", names), nil
}

// watchCSV is the watch list as reference,fc,value,status,read_at_unix (CRLF).
func watchCSV(status *pyjson.Object) *download {
	writer := pycsv.NewWriter()
	writer.Row("reference", "fc", "value", "status", "read_at_unix")
	watch, _ := value(status, "watch", nil).([]any)
	for _, raw := range watch {
		item := raw.(*pyjson.Object)
		at := value(item, "at", nil)
		if !truthy(at) {
			at = ""
		}
		writer.Row(value(item, "ref", nil), value(item, "fc", nil), value(item, "text", ""), value(item, "status", ""), at)
	}
	host, _ := value(status, "host", "").(string)
	if host == "" {
		host = "device"
	}
	return &download{body: writer.String(), contentType: "text/csv; charset=utf-8",
		disposition: "attachment; filename=watch-" + strings.ReplaceAll(host, ".", "-") + ".csv"}
}
