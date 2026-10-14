"""Which Sampled Values streams to supervise, and what a received report means.

Pure. Expectations come only from the SCD (SampledValueControl + SMV address) and
the project's nominal frequency; nothing is guessed from the wire.
"""
STREAM_LIMIT = 16
RATE_TOLERANCE = 0.02
PROBLEM_ORDER = ("conf_rev_mismatch", "dataset_mismatch", "rate_mismatch", "not_synchronised",
                 "sample_loss", "invalid_quality", "simulation")


def expected_rate(stream, nominal_hz):
    """Samples per second the SCD promises, or None when it cannot be stated."""
    if stream["smp_mod"] == 0:
        return stream["smp_rate"] * nominal_hz
    if stream["smp_mod"] == 1:
        return stream["smp_rate"]
    return None


def samples_per_cycle(stream, nominal_hz):
    if stream["smp_mod"] == 0:
        return stream["smp_rate"]
    if stream["smp_mod"] == 1 and stream["smp_rate"] % nominal_hz == 0:
        return stream["smp_rate"] // nominal_hz
    return 0


def watch_list(ieds, nominal_hz):
    """(supervised, not_supervised): every SCD stream of IEDs not set to Off, up to the limit."""
    streams = []
    for ied in ieds:
        if ied.get("role") == "off":
            continue
        for stream in (ied.get("sv") or {}).get("publishes", []):
            rate = expected_rate(stream, nominal_hz)
            streams.append({**stream, "key": stream["svcb_ref"], "publisher": ied["name"],
                            "expected_rate": rate, "per_cycle": samples_per_cycle(stream, nominal_hz),
                            "wrap": rate or 0, "state": "stopped", "problems": []})
    return streams[:STREAM_LIMIT], streams[STREAM_LIMIT:]


def native_stream(watch):
    """Only interpret values when the layout is recognised and whole cycles are defined."""
    interpret = watch["layout"] == "9-2LE" and watch["per_cycle"] > 0
    return {"key": watch["key"], "app_id": watch["app_id"], "mac": watch["mac"],
            "layout": "9-2LE" if interpret else "raw", "per_cycle": watch["per_cycle"],
            "wrap": watch["wrap"], "channels": watch["channels"] if interpret else [],
            "scale": watch["scale"] if interpret else []}


def assess(watch, event, previous=None):
    """State and problems of one stream from a native `sv_rx` report."""
    previous = previous or {}
    state = event.get("state", "waiting")
    problems = []
    if event.get("asdus"):
        if event.get("conf_rev") != watch["conf_rev"]:
            problems.append("conf_rev_mismatch")
        if watch["layout"] == "9-2LE" and event.get("data_size") not in (None, 64):
            problems.append("dataset_mismatch")
        rate, expected = event.get("rate") or 0, watch.get("expected_rate")
        if state == "receiving" and expected and rate and abs(rate - expected) / expected > RATE_TOLERANCE:
            problems.append("rate_mismatch")
        if event.get("smp_synch") == 0:
            problems.append("not_synchronised")
        if (event.get("lost") or 0) > (previous.get("lost") or 0):
            problems.append("sample_loss")
        if (event.get("invalid_quality") or 0) > (previous.get("invalid_quality") or 0):
            problems.append("invalid_quality")
        if event.get("simulate"):
            problems.append("simulation")
    problems.sort(key=PROBLEM_ORDER.index)
    if state == "receiving" and problems:
        state = "problem"
    return {"state": state, "problems": problems}
