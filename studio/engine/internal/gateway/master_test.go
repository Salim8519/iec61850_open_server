package gateway

import (
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pyoracle"
)

// TestMasterRulesAgainstPython replays master_rules with random settings, ranges, probe
// requests, command routes and stored master sections.
func TestMasterRulesAgainstPython(t *testing.T) {
	replay(t, "master_rules_oracle.py", "2000", func(item pyoracle.Case, _ map[string][]*pyjson.Object) {
		args := item.Args
		switch item.Fn {
		case "protocol_arg":
			got, err := ProtocolArg(args[0])
			expect(t, item, got, err)
		case "validate_host":
			got, err := ValidateHost(args[0])
			expect(t, item, got, err)
		case "validate_settings":
			got, err := ValidateMasterSettings(args[0], args[1])
			expect(t, item, got, err)
		case "validate_ranges":
			got, err := ValidateRanges(args[0])
			expect(t, item, got, err)
		case "validate_probe":
			table, start, end, err := ValidateProbe(args[0], args[1], args[2])
			expect(t, item, []any{table, start, end}, err)
		case "ranges_from_probe":
			got, err := RangesFromProbe(list(args[0]), args[1])
			expect(t, item, got, err)
		case "validate_routes":
			got, err := ValidateRoutes(args[0])
			expect(t, item, got, err)
		case "normalize_master":
			got, err := NormalizeMaster(args[0])
			expect(t, item, got, err)
		case "validate_role":
			got, err := ValidateRole(args[0])
			expect(t, item, got, err)
		case "empty_master":
			expect(t, item, EmptyMaster(), nil)
		default:
			t.Errorf("unknown oracle function %s", item.Fn)
		}
	})
}

// TestMasterPointsAgainstPython replays master_points: merge sequences on one table (the
// table is compared after every batch, also after an error), display rows, CSV export,
// quality words, value text and label imports.
func TestMasterPointsAgainstPython(t *testing.T) {
	replay(t, "master_points_oracle.py", "1500", func(item pyoracle.Case, _ map[string][]*pyjson.Object) {
		args := item.Args
		switch item.Fn {
		case "merge_sequence":
			table, steps := pyjson.NewObject(), []any{}
			for _, batch := range list(args[0]) {
				added, err := Merge(table, list(batch))
				if err != nil {
					steps = append(steps, pyjson.NewObject().Set("error", errText(err)))
				} else {
					steps = append(steps, pyjson.NewObject().Set("added", strs(added)))
				}
				snapshot, _ := pyjson.Dumps(table)
				copied, _ := pyjson.Decode(snapshot)
				steps = append(steps, pyjson.NewObject().Set("table", copied))
			}
			expect(t, item, steps, nil)
		case "rows":
			got, err := Rows(object(args[0]), args[1].(string), stringList(args[2]), object(args[3]), args[4].(bool))
			expect(t, item, got, err)
		case "export_csv":
			expect(t, item, ExportMasterCSV(list(args[0])), nil)
		case "quality_words":
			group, _ := args[2].(string)
			expect(t, item, strs(QualityWords(args[0].(string), args[1], group)), nil)
		case "value_text":
			got, err := ValueText(object(args[0]))
			expect(t, item, got, err)
		case "key":
			expect(t, item, Key(object(args[0])), nil)
		case "group_name":
			expect(t, item, GroupName(args[0].(string), args[1].(string)), nil)
		case "import_labels":
			labels, problems, err := ImportLabels(args[0].(string))
			expect(t, item, []any{labels, strs(problems)}, err)
		default:
			t.Errorf("unknown oracle function %s", item.Fn)
		}
	})
}
