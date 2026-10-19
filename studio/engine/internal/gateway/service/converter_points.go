package service

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// mapped is map_points over the master's watched rows and the user's command routes.
func (c *Converter) mapped(protocol string) ([]any, []string, []any, error) {
	rows, err := c.Master.Points()
	if err != nil {
		return nil, nil, nil, err
	}
	points, notes, err := gateway.MapPoints(rows, protocol, c.routes())
	return points, notes, rows, err
}

// Points are the upstream points (and notes), refused until the device is connected.
func (c *Converter) Points(protocol string) ([]any, []string, error) {
	if !c.Master.LinkOpen() {
		return nil, nil, apperr.ValueError("Connect to the device first: the converter serves the points ScadaBench reads from it")
	}
	points, notes, _, err := c.mapped(protocol)
	if err != nil {
		return nil, nil, err
	}
	if len(points) == 0 {
		return nil, nil, apperr.ValueError("No watched points from the device yet. Ask for all values, or tick the points to serve.")
	}
	return points, notes, nil
}

// Sample is the upstream value of one remote point.
func (c *Converter) Sample(point *pyjson.Object) (Sample, error) {
	m := c.Master
	device := text(m.Config(), "protocol")
	m.tableMu.Lock()
	if m.device != nil {
		device = pytext.Str(m.device[0])
	}
	var row *pyjson.Object
	if found, ok := get(m.table, pytext.Str(get(point, "remote"))).(*pyjson.Object); ok && found.Len() > 0 {
		row = copyObject(found)
	}
	m.tableMu.Unlock()
	good := false
	if row != nil {
		quality, found := row.Get("quality")
		if !found {
			quality = int64(0)
		}
		group, _ := get(row, "group").(string)
		good = len(gateway.QualityWords(device, quality, group)) == 0
	}
	object, err := gateway.RemoteSample(point, row, good, m.LinkOpen())
	if err != nil {
		return Sample{}, err
	}
	ok, _ := get(object, "source_ok").(bool)
	return Sample{ID: text(object, "id"), Value: get(object, "value"), Quality: get(object, "quality"),
		TimeMS: get(object, "time_ms"), SourceOK: ok}, nil
}

// Table is the mapping as gateway point rows with what each point carries now.
func (c *Converter) Table(protocol string) ([]any, error) {
	if !c.Master.LinkOpen() {
		return []any{}, nil
	}
	points, _, watched, err := c.mapped(protocol)
	if err != nil {
		return nil, err
	}
	values := map[string]*pyjson.Object{}
	for _, raw := range watched {
		row := raw.(*pyjson.Object)
		values[pytext.Str(get(row, "key"))] = row
	}
	rows := []any{}
	for _, raw := range points {
		point := raw.(*pyjson.Object)
		row := values[pytext.Str(get(point, "remote"))]
		said := "relayed to the device"
		online := false
		if row != nil {
			said = pytext.Str(get(row, "value_text")) + " \xc2\xb7 " + pytext.Str(get(row, "quality_text"))
			online = pytext.Truthy(get(row, "good")) && !pytext.Truthy(get(row, "stale"))
		}
		rows = append(rows, copyObject(point).Set("ied_name", "Device").Set("ied_role", "real").
			Set("monitor_only", false).Set("value_text", said).Set("online", online))
	}
	return rows, nil
}
