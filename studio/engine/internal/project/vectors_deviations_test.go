package project

import "strings"

// pythonRecorded maps a Go result to what the frozen Python engine recorded in the golden
// vectors, for the documented Go-only deviations only. Every other byte must still match.
//
// ADR-0090: the Go engine adds the MEASUREMENT_LIST kind, so pydantic's literal message for
// an unknown kind names one more choice than the Python vectors recorded.
func pythonRecorded(got string) string {
	return strings.ReplaceAll(got, "'JUNCTION', 'INFEED' or 'MEASUREMENT_LIST'", "'JUNCTION' or 'INFEED'")
}
