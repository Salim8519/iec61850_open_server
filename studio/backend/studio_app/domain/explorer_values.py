"""Turn raw MMS values read by IED Explorer into plain text, without any SCD type.

Explorer has only the server's own type description (MMS type number and the
attribute name), so it decodes what is unambiguous: quality (`.q`), timestamps
(`.t`), double-point positions (2-bit `stVal`) and scalars. Anything else is
shown raw instead of guessed.
"""
from datetime import datetime, timezone
from .live_value import MMS_NAMES

VALIDITY = {0: "good", 1: "reserved", 2: "invalid", 3: "questionable"}
QUALITY_FLAGS = ((4, "overflow"), (8, "out of range"), (16, "bad reference"), (32, "oscillatory"),
                 (64, "failure"), (128, "old data"), (256, "inconsistent"), (512, "inaccurate"),
                 (1024, "substituted"), (2048, "test"), (4096, "operator blocked"), (8192, "derived"))
DBPOS = {0: "intermediate", 1: "off (open)", 2: "on (closed)", 3: "bad state"}


def type_name(mms_type):
    return MMS_NAMES.get(mms_type, f"type {mms_type}")


def quality_text(bits):
    validity = VALIDITY[bits & 3]
    flags = [name for mask, name in QUALITY_FLAGS if bits & mask]
    return validity if not flags else f"{validity}: {', '.join(flags)}"


def time_text(milliseconds):
    if not isinstance(milliseconds, int) or milliseconds <= 0:
        return "not set"
    moment = datetime.fromtimestamp(milliseconds / 1000, tz=timezone.utc)
    return moment.strftime("%Y-%m-%d %H:%M:%S.") + f"{milliseconds % 1000:03d} UTC"


def describe(ref, raw):
    """Return {"text", "status"} for one read result. status: ok, warning, problem, error."""
    attribute = ref.rsplit(".", 1)[-1]
    if isinstance(raw, dict):
        if "access_error" in raw:
            return {"text": f"device refused the read (access error {raw['access_error']})", "status": "error"}
        if "bits" in raw:
            if attribute == "q" and raw["bits"] >= 2:
                text = quality_text(raw["le"])
                status = "ok" if text == "good" else "problem" if text.startswith("invalid") else "warning"
                return {"text": text, "status": status}
            if attribute == "stVal" and raw["bits"] == 2:
                state = raw["be"]
                return {"text": DBPOS[state], "status": "ok" if state in (1, 2) else "warning"}
            return {"text": f"bits {raw['be']:0{raw['bits']}b}", "status": "ok"}
        if "octets" in raw:
            return {"text": raw["octets"] or "(empty)", "status": "ok"}
        return {"text": "value type not shown", "status": "warning"}
    if attribute in ("t", "T") and isinstance(raw, int) and not isinstance(raw, bool):
        return {"text": time_text(raw), "status": "ok"}
    if isinstance(raw, bool):
        return {"text": "true" if raw else "false", "status": "ok"}
    if isinstance(raw, float):
        return {"text": f"{raw:.6g}", "status": "ok"}
    if raw is None:
        return {"text": "no value", "status": "warning"}
    return {"text": str(raw), "status": "ok"}
