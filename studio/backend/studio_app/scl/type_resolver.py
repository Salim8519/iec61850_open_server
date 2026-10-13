"""Resolve SCL basic types and initial values into stack configuration types."""
import json
import math
import re
from datetime import datetime, timezone
from studio_app.contracts.errors import StudioError

TYPES = {
    "BOOLEAN": 0, "INT8": 1, "INT16": 2, "INT32": 3, "INT64": 4,
    "INT128": 5, "INT8U": 6, "INT16U": 7, "INT24U": 8, "INT32U": 9,
    "FLOAT32": 10, "FLOAT64": 11, "Enum": 12, "Octet64": 13,
    "Octet6": 14, "Octet8": 15, "VisString32": 16, "VisString64": 17,
    "VisString65": 18, "VisString129": 19, "VisString255": 20,
    "Unicode255": 21, "Timestamp": 22, "Quality": 23, "Check": 24,
    "Dbpos": 25, "Tcmd": 25, "Struct": 27, "EntryTime": 28,
    "Currency": 30, "OptFlds": 31, "TrgOps": 32, "ObjRef": 19, "EntryID": 15,
}
FCS = {name: index for index, name in enumerate(
    "ST MX SP SV CF DC SG SE SR OR BL EX CO US MS RP BR LG GO".split())}


def identifier(value, context="identifier"):
    if not value or not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", value):
        raise StudioError(f"Invalid {context}: {value!r}")
    return value


def initial_value(kind, text, enum_values):
    if text is None or not text.strip():
        if kind == "BOOLEAN":
            return False
        if "String" in kind or kind in {"Currency", "ObjRef"}:
            return ""
        return 0
    text = text.strip()
    if kind == "Timestamp" and "T" in text:
        parsed = datetime.fromisoformat(text.replace("Z", "+00:00"))
        if parsed.tzinfo is None:
            parsed = parsed.replace(tzinfo=timezone.utc)
        return int(parsed.timestamp() * 1000)
    if kind == "BOOLEAN":
        if text.lower() not in {"true", "false", "1", "0"}:
            raise StudioError(f"Invalid BOOLEAN initial value {text!r}")
        return text.lower() in {"true", "1"}
    if "String" in kind or kind in {"Currency", "ObjRef"}:
        return text
    if kind == "Enum" and text in enum_values:
        return enum_values[text]
    if kind.startswith("FLOAT"):
        number = float(text)
        if not math.isfinite(number):
            raise StudioError("Non-finite initial measurement")
        return number
    try:
        return int(text)
    except ValueError as error:
        raise StudioError(f"Unsupported {kind} initial value {text!r}") from error


def config_initial(kind, value):
    if kind in {"Struct", "Timestamp", "Quality", "Check", "Dbpos", "Tcmd",
                "EntryTime", "EntryID", "Octet64", "Octet6", "Octet8"}:
        return ""
    if isinstance(value, str):
        if any(char in value for char in ('"', '\n', '\r')):
            raise StudioError("Quoted/newline string values need native model encoding")
        return "=" + json.dumps(value, ensure_ascii=True)
    if isinstance(value, bool):
        return "=" + str(int(value))
    return "=" + str(value)


def trigger_bits(element):
    return sum(bit for key, bit in (("dchg", 1), ("qchg", 2), ("dupd", 4))
               if element.get(key) == "true")
