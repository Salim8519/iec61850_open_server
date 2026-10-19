// Package svdom decides which Sampled Values streams to supervise and what a received
// native `sv_rx` report means (port of domain/sv_supervision.py). Pure: expectations
// come only from the SCD (SampledValueControl + SMV address) and the project's nominal
// frequency; nothing is guessed from the wire. Results are new pyjson objects.
package svdom

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// StreamLimit is STREAM_LIMIT: at most this many streams are supervised.
const StreamLimit = 16

// RateTolerance is RATE_TOLERANCE: the relative sample-rate deviation still accepted.
const RateTolerance = 0.02

// ProblemOrder returns PROBLEM_ORDER, the order problems are reported in.
func ProblemOrder() []string {
	return []string{"conf_rev_mismatch", "dataset_mismatch", "rate_mismatch", "not_synchronised",
		"sample_loss", "invalid_quality", "simulation"}
}

// ExpectedRate returns the samples per second the SCD promises, or nil when it cannot
// be stated. smp_mod 0 is samples per cycle (times nominalHz, int or float result as in
// Python), 1 is samples per second.
func ExpectedRate(stream *pyjson.Object, nominalHz any) any {
	smpMod, smpRate := pyv.Get(stream, "smp_mod"), pyv.Get(stream, "smp_rate")
	if pyv.Equal(smpMod, int64(0)) {
		return mul(smpRate, nominalHz)
	}
	if pyv.Equal(smpMod, int64(1)) {
		return smpRate
	}
	return nil
}

// SamplesPerCycle returns the samples in one nominal cycle, or 0 when not whole. A zero
// nominalHz (ZeroDivisionError in Python) also gives 0.
func SamplesPerCycle(stream *pyjson.Object, nominalHz any) any {
	smpMod, smpRate := pyv.Get(stream, "smp_mod"), pyv.Get(stream, "smp_rate")
	if pyv.Equal(smpMod, int64(0)) {
		return smpRate
	}
	if pyv.Equal(smpMod, int64(1)) {
		if rest, ok := mod(smpRate, nominalHz); ok && pyv.Equal(rest, int64(0)) {
			whole, _ := floorDiv(smpRate, nominalHz)
			return whole
		}
	}
	return int64(0)
}

// WatchList returns (supervised, notSupervised): every SCD stream of IEDs not set to
// Off as a new watch {**stream, key, publisher, expected_rate, per_cycle, wrap, state,
// problems}, the first StreamLimit supervised.
func WatchList(ieds []*pyjson.Object, nominalHz any) ([]*pyjson.Object, []*pyjson.Object) {
	streams := []*pyjson.Object{}
	for _, ied := range ieds {
		if pyv.Equal(pyv.Get(ied, "role"), "off") {
			continue
		}
		sv := pyv.OrObj(ied, "sv")
		for _, stream := range pyv.Objects(pyv.List(pyv.GetOr(sv, "publishes", []any{}))) {
			rate := ExpectedRate(stream, nominalHz)
			var wrap any = int64(0)
			if pyv.Truthy(rate) {
				wrap = rate
			}
			streams = append(streams, pyv.Merge(stream, pyv.O("key", pyv.Get(stream, "svcb_ref"),
				"publisher", pyv.Get(ied, "name"), "expected_rate", rate,
				"per_cycle", SamplesPerCycle(stream, nominalHz), "wrap", wrap, "state", "stopped",
				"problems", []any{})))
		}
	}
	if len(streams) <= StreamLimit {
		return streams, []*pyjson.Object{}
	}
	return streams[:StreamLimit], streams[StreamLimit:]
}

// NativeStream returns the stream description for the native worker; values are only
// interpreted when the layout is 9-2LE and whole cycles are defined.
func NativeStream(watch *pyjson.Object) *pyjson.Object {
	interpret := pyv.Equal(pyv.Get(watch, "layout"), "9-2LE") && greater(pyv.Get(watch, "per_cycle"), int64(0))
	layout := "raw"
	var channels, scale any = []any{}, []any{}
	if interpret {
		layout = "9-2LE"
		channels, scale = pyv.Get(watch, "channels"), pyv.Get(watch, "scale")
	}
	return pyv.O("key", pyv.Get(watch, "key"), "app_id", pyv.Get(watch, "app_id"), "mac", pyv.Get(watch, "mac"),
		"layout", layout, "per_cycle", pyv.Get(watch, "per_cycle"), "wrap", pyv.Get(watch, "wrap"),
		"channels", channels, "scale", scale)
}
