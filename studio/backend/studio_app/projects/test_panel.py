"""Validate saved signal-test controls and report provenance."""
from studio_app.contracts.errors import StudioError
from studio_app.domain.signal import validate_value
from studio_app.scl.report_catalogue import report_members
import re


def validate_mapping(ied, reference, report_ref):
    if report_ref is not None and not isinstance(report_ref, str):
        raise StudioError("Report reference must be text or null")
    if report_ref and reference not in report_members(ied, report_ref):
        raise StudioError("Selected signal is not a member of the chosen report")


def validate_panel(entries, ieds):
    if not isinstance(entries, list) or len(entries) > 500:
        raise StudioError("Signal Test supports at most 500 saved controls")
    models = {ied["id"]: ied for ied in ieds}
    result, ids = [], set()
    for entry in entries:
        if not isinstance(entry, dict):
            raise StudioError("Invalid saved signal control")
        key, label = entry.get("id"), entry.get("label")
        if not isinstance(key, str) or not re.fullmatch(r"[A-Za-z0-9_-]{1,64}", key) or key in ids:
            raise StudioError("Saved control IDs must be valid and unique")
        if not isinstance(label, str) or not 1 <= len(label.strip()) <= 100:
            raise StudioError("Saved control label must contain 1 to 100 characters")
        if not isinstance(entry.get("ied_id"), str) or not isinstance(entry.get("ref"), str):
            raise StudioError("Saved control needs an IED and exact signal reference")
        ied = models.get(entry.get("ied_id"))
        signal = next((s for s in ied["signals"] if s["ref"] == entry.get("ref")), None) if ied else None
        if signal is None or not signal["writable"] or signal["fc"] == "CF":
            raise StudioError("Choose a supported runtime-editable signal")
        validate_value(signal, signal["value"])
        report_ref = entry.get("report_ref")
        validate_mapping(ied, signal["ref"], report_ref)
        duration = entry.get("duration_ms", 1000)
        active = entry.get("active_value", True)
        if isinstance(duration, bool) or not isinstance(duration, int) or not 100 <= duration <= 60000:
            raise StudioError("Pulse duration must be 100 to 60000 milliseconds")
        if not isinstance(active, bool):
            raise StudioError("Pulse active value must be Boolean")
        result.append({"id": key, "label": label.strip(), "ied_id": ied["id"], "ref": signal["ref"],
                       "report_ref": report_ref or None, "duration_ms": duration, "active_value": active})
        ids.add(key)
    return result
