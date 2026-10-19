package explorer

import (
	"errors"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// endpointKey is the watch_by_host key "<host>:<port>".
func endpointKey(host string, port any) string { return host + ":" + pytext.Str(port) }

// buildWatch opens only the key logical nodes and takes their key points; a node that
// fails with a native error (not a timeout) is skipped, as Python's except RuntimeError.
func (s *Service) buildWatch() ([]any, error) {
	devices, err := s.Devices()
	if err != nil {
		return nil, err
	}
	var points []any
	for _, device := range asList(devices) {
		name := pytext.Str(device)
		nodes, err := s.Nodes(name)
		if err != nil {
			return nil, err
		}
		var names []string
		for _, node := range asList(nodes) {
			names = append(names, pytext.Str(node))
		}
		for _, node := range KeyNodes(name, names) {
			view, err := s.Node(node)
			var native *apperr.Error
			if err != nil && !isTimeout(err) && errors.As(err, &native) && native.Kind == apperr.Runtime {
				continue
			}
			if err != nil {
				return nil, err
			}
			points = append(points, KeyPoints(view)...)
		}
	}
	return MergeWatch(points), nil
}

func asList(value any) []any {
	items, _ := value.([]any)
	return items
}

// startWatching restores the watch list remembered for this device, else builds the
// key points, then polls.
func (s *Service) startWatching() (*pyjson.Object, error) {
	host, port, _ := s.Endpoint()
	section, _ := s.memory.Get().Get("explorer")
	remembered := section.(*pyjson.Object)
	hosts, _ := remembered.Get("watch_by_host")
	items := asList(get(hosts.(*pyjson.Object), endpointKey(host, port)))
	if len(items) == 0 {
		built, err := s.buildWatch()
		if err != nil {
			return nil, err
		}
		items = built
	}
	if len(items) == 0 {
		return s.Status(), nil
	}
	watch := make([]*pyjson.Object, len(items))
	for index, item := range items {
		watch[index] = item.(*pyjson.Object)
	}
	interval, _ := remembered.Get("interval_ms")
	if _, err := s.SetWatch(watch, interval); err != nil {
		return nil, err
	}
	return s.StartPolling()
}

// remember saves the endpoint, interval and watch list after every change.
func (s *Service) remember() error {
	host, port, ok := s.Endpoint()
	if !ok {
		return nil
	}
	section, _ := s.memory.Get().Get("explorer")
	current := section.(*pyjson.Object)
	watches := copyObject(get(current, "watch_by_host").(*pyjson.Object))
	s.mu.Lock()
	items := make([]any, len(s.watch))
	for index, item := range s.watch {
		items[index] = pyjson.NewObject().Set("ref", get(item, "ref")).Set("fc", get(item, "fc")).
			Set("type", get(item, "type"))
	}
	interval := s.interval
	s.mu.Unlock()
	watches.Set(endpointKey(host, port), items)
	_, err := s.memory.Update("explorer", pyjson.NewObject().Set("host", host).Set("port", port).
		Set("interval_ms", interval).Set("watch_by_host", watches))
	return err
}
