"""Interpret a value received from a real IED using the SCD engineering type."""

EXPECTED_MMS = {
    "BOOLEAN": {2}, "INT8": {4}, "INT16": {4}, "INT32": {4}, "INT64": {4}, "INT128": {4},
    "INT8U": {5, 4}, "INT16U": {5, 4}, "INT24U": {5, 4}, "INT32U": {5, 4},
    "FLOAT32": {6}, "FLOAT64": {6}, "Enum": {4}, "Dbpos": {3}, "Tcmd": {3}, "Quality": {3},
    "Check": {3}, "Timestamp": {14}, "EntryTime": {10}, "Octet64": {7}, "Octet6": {7}, "Octet8": {7},
    "EntryID": {7}, "VisString32": {8}, "VisString64": {8}, "VisString65": {8}, "VisString129": {8},
    "VisString255": {8}, "ObjRef": {8}, "Currency": {8}, "Unicode255": {13}, "OptFlds": {3},
    "TrgOps": {3},
}
MMS_NAMES = {0: "array", 1: "structure", 2: "boolean", 3: "bit string", 4: "integer",
             5: "unsigned", 6: "float", 7: "octet string", 8: "visible string",
             9: "generalized time", 10: "binary time", 11: "BCD", 12: "object id",
             13: "unicode string", 14: "UTC time", 15: "access error"}
BIG_ENDIAN_CODED = {"Dbpos", "Tcmd"}


def convert(kind, raw):
    """Return the ScadaBench value for one device value, or raise ValueError with a reason."""
    if raw is None:
        raise ValueError("the device returned no value")
    if isinstance(raw, dict):
        if "access_error" in raw:
            raise ValueError(f"the device refused the read (access error {raw['access_error']})")
        if "bits" in raw:
            return raw["be"] if kind in BIG_ENDIAN_CODED else raw["le"]
        if "octets" in raw:
            return raw["octets"]
        raise ValueError("the device sent a value type ScadaBench cannot show yet")
    if kind == "BOOLEAN":
        if not isinstance(raw, bool):
            raise ValueError("expected true/false from the device")
        return raw
    if kind.startswith("FLOAT"):
        if isinstance(raw, bool) or not isinstance(raw, (int, float)):
            raise ValueError("expected a number from the device")
        return float(raw)
    if kind.startswith("INT") or kind in {"Enum", "Timestamp", "EntryTime"}:
        if isinstance(raw, bool) or not isinstance(raw, int):
            raise ValueError("expected a whole number from the device")
        return raw
    if isinstance(raw, str):
        return raw
    raise ValueError("the device value does not match the SCD type")


def type_matches(kind, mms_type):
    expected = EXPECTED_MMS.get(kind)
    return expected is None or mms_type in expected


def describe_mms(mms_type):
    return MMS_NAMES.get(mms_type, f"MMS type {mms_type}")
