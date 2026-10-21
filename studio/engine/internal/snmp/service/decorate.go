package service

import (
	"fmt"
	"strconv"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/tools/pytext"
)

// Decorate adds to each value row its MIB name, what the value means ("up" for 1, uptime as
// days and hours) and whether the MIB lets a manager write it. Rows are changed in place.
func (l *Library) Decorate(rows []any) []any {
	library := l.Lib()
	out := make([]any, 0, len(rows))
	for _, item := range rows {
		row, isObject := item.(*pyjson.Object)
		if !isObject {
			continue
		}
		oid := pytext.Str(field(row, "oid"))
		kind := pytext.Str(field(row, "type"))
		value := pytext.Str(field(row, "value"))
		row.Set("name", library.Name(oid))
		meaning := ""
		node, _ := library.Lookup(oid)
		switch kind {
		case model.TypeInteger:
			if number, err := strconv.ParseInt(value, 10, 64); err == nil {
				meaning = library.EnumText(oid, number)
			}
		case model.TypeTimeTicks:
			if ticks, err := strconv.ParseUint(value, 10, 64); err == nil {
				meaning = Duration(ticks)
			}
		}
		if node != nil && node.Units != "" && meaning == "" {
			meaning = node.Units
		}
		row.Set("meaning", meaning)
		writable := false
		if node != nil {
			writable = node.Access == "read-write" || node.Access == "read-create"
		}
		if _, has := row.Get("writable"); !has {
			row.Set("writable", writable)
		}
		out = append(out, row)
	}
	return out
}

// Duration words timeticks (hundredths of a second) like Net-SNMP: "3 days, 4:05:06.07".
func Duration(ticks uint64) string {
	hundredths := ticks % 100
	seconds := ticks / 100
	days := seconds / 86400
	clock := fmt.Sprintf("%d:%02d:%02d.%02d", seconds/3600%24, seconds/60%60, seconds%60, hundredths)
	switch days {
	case 0:
		return clock
	case 1:
		return "1 day, " + clock
	}
	return fmt.Sprintf("%d days, %s", days, clock)
}
