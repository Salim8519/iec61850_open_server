"""GOOSE checks for the station health view: on the device and on the wire."""
from .health_reads import gocb_object

PROBLEM_TEXT = {
    "conf_rev_mismatch": "arrives with a different configuration revision than the SCD",
    "dataset_mismatch": "arrives with a different dataset than the SCD",
    "go_id_mismatch": "arrives with a different GOOSE ID than the SCD",
    "invalid_frame": "arrives but cannot be decoded",
    "needs_commission": "says its publisher still needs commissioning",
    "timeout": "stopped arriving within its allowed time",
}


def result(area, item, verdict, message, observed_at=None, source="scd", detail=None):
    return {"area": area, "item": item, "result": verdict, "message": message,
            "observed_at": observed_at, "source": source, "detail": detail or {}}


def sentence(parts):
    """Join findings into one sentence without changing the case of references."""
    text = "; ".join(parts)
    return text[:1].upper() + text[1:] + "."


def _same_reference(a, b):
    return str(a or "").replace(".", "$") == str(b or "").replace(".", "$")


def device_gocb_checks(ied, reads):
    """Compare GoID, DatSet and ConfRev read from the device with each SCD GOOSE block."""
    rows = []
    for control in (ied.get("goose") or {}).get("publishes", []):
        base = gocb_object(control["gocb_ref"])
        values = reads["values"]
        got = {name: values.get(f"{base}.{name}") for name in ("GoEna", "GoID", "DatSet", "ConfRev")}
        when, item = reads["checked_at"], control["gocb_ref"]
        if all(value is None for value in got.values()):
            rows.append(result("goose_device", item, "warning", "The device did not return this GOOSE block. "
                               "It may run a different configuration than the SCD.", when, "mms_read"))
            continue
        differences = []
        if got["GoID"] is not None and got["GoID"] != control["go_id"]:
            differences.append(f"GOOSE ID is {got['GoID']} on the device but {control['go_id']} in the SCD")
        if got["DatSet"] is not None and not _same_reference(got["DatSet"], control["dataset_ref"]):
            differences.append(f"dataset is {got['DatSet']} on the device but {control['dataset_ref']} in the SCD")
        if got["ConfRev"] is not None and got["ConfRev"] != control["conf_rev"]:
            differences.append(f"confRev is {got['ConfRev']} on the device but {control['conf_rev']} in the SCD")
        if differences:
            rows.append(result("goose_device", item, "problem", sentence(differences),
                               when, "mms_read", got))
        elif got["GoEna"] is False:
            rows.append(result("goose_device", item, "warning", "This GOOSE block is switched off on the device.",
                               when, "mms_read", got))
        else:
            rows.append(result("goose_device", item, "ok", "Matches the SCD.", when, "mms_read", got))
    return rows


def wire_check(watch, listener):
    """Verdict for one GOOSE block from what the passive listener saw."""
    item = watch["gocb_ref"]
    if listener.get("state") != "listening":
        return result("goose_wire", item, "not_checked",
                      listener.get("message") or "The GOOSE listener is not running.")
    when, state, problem = watch.get("observed_at"), watch.get("state"), watch.get("problem")
    if state == "receiving":
        verdict = "warning" if watch.get("received_test") else "ok"
        message = ("Arriving, marked as test/simulation traffic." if verdict == "warning"
                   else f"Arriving as engineered ({watch.get('frames', 0)} messages).")
        return result("goose_wire", item, verdict, message, when, "wire")
    if state in {"problem", "timeout"}:
        detail = {key: watch.get(f"received_{key}") for key in ("conf_rev", "go_id", "dataset")}
        expected = {"conf_rev": watch.get("conf_rev"), "go_id": watch.get("go_id"), "dataset": watch.get("dataset_ref")}
        text = PROBLEM_TEXT.get(problem or state, "arrives with a problem")
        named = {"conf_rev_mismatch": "conf_rev", "dataset_mismatch": "dataset", "go_id_mismatch": "go_id"}.get(problem)
        if named:
            text += f" (wire {detail[named]}, SCD {expected[named]})"
        return result("goose_wire", item, "problem", f"This GOOSE {text}.", when, "wire", detail)
    return result("goose_wire", item, "warning", "Not seen on the network yet.", when, "wire")


def unknown_publishers(status):
    """GOOSE on the wire that the project does not describe, plus the listing limit if reached."""
    rows = [dict(item) for item in status.get("unknown", [])]
    limit = status.get("unknown_limit")
    note = (f"More than {limit} unknown GOOSE blocks were seen; only the first {limit} are listed."
            if limit else "")
    return rows, note
