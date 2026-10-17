"""Choose the IEC 61850 values a SCADA gateway really needs (smart point list).

A real station model offers thousands of leaves; SCADA wants positions, alarms and
trips, measurements and device health. Everything else stays out of the proposal and
can still be added by hand or by CSV. Generic I/O (GGIO) holds many spare inputs, so a
GGIO value is kept only when the IED puts it in a dataset or the SCD describes it, and
generic values are ranked last when the gateway's point limit is reached.
"""
import re

STATUS_CLASSES = ("XCBR", "XSWI", "CSWI", "CILO", "GGIO", "SIMG", "SIML", "SARC", "SPDC", "SCBR", "SOPM")
MEASUREMENT_CLASSES = ("MMXU", "MMXN", "MSQI", "MHAI", "MMTR", "MSTA", "GGIO", "ATCC", "YLTC")
HEALTH_OBJECTS = ("Health", "Beh", "PhyHealth")
PRIORITY = ("double", "command_double", "command_single", "single", "integer", "float")
HARMONIC_TOTALS = ("ThdA", "ThdPhV", "ThdPPV", "ThdAmp", "ThdVol")  # MHAI: totals only, never every harmonic


def node_class(ref):
    """LN class of a leaf reference such as LD0/QA1XCBR1.Pos.stVal, or LLN0."""
    node = ref.split("/", 1)[-1].split(".", 1)[0]
    if node == "LLN0":
        return "LLN0"
    match = re.search(r"([A-Z]{4})\d*$", node)
    return match.group(1) if match else ""


def _data_object(ref):
    parts = ref.split("/", 1)[-1].split(".")
    return parts[1] if len(parts) > 1 else ""


def dataset_refs(ied):
    """Leaf references the IED puts in any dataset, or None when it has no datasets."""
    refs = {ref for dataset in ied.get("datasets", []) for ref in dataset.get("signal_refs", [])}
    return refs or None


def engineered(signal, refs):
    """False for a GGIO value that nothing in the SCD uses (a spare input)."""
    if node_class(signal["ref"]) != "GGIO" or refs is None:
        return True
    return signal["ref"] in refs or bool(signal.get("description"))


def rank(candidate):
    """Order kept when the point limit is reached: positions and commands first, generic I/O last."""
    generic = node_class(candidate["ref"]) == "GGIO"
    return (generic, PRIORITY.index(candidate["kind"]))


def smart_kind(signal):
    """Gateway kind of a useful leaf, or None when SCADA normally does not need it."""
    ref, kind, fc = signal["ref"], signal["kind"], signal["fc"]
    lnclass = node_class(ref)
    if fc == "ST" and ref.endswith(".stVal") and kind == "Dbpos":
        return "double"
    if fc == "ST" and ref.endswith(".stVal") and kind == "BOOLEAN":
        return "single" if lnclass in STATUS_CLASSES else None
    if fc == "ST" and ref.endswith(".general") and kind == "BOOLEAN":
        return "single" if lnclass.startswith(("P", "R")) else None
    if fc == "ST" and ref.endswith(".stVal") and _data_object(ref) in HEALTH_OBJECTS:
        if lnclass in ("LLN0", "LPHD") and (kind == "Enum" or kind.startswith("INT")):
            return "integer"
        return None
    if lnclass == "MHAI" and _data_object(ref) not in HARMONIC_TOTALS:
        return None
    if fc == "MX" and ref.endswith(".mag.f") and kind.startswith("FLOAT"):
        return "float" if lnclass in MEASUREMENT_CLASSES else None
    if fc == "MX" and ref.endswith(".mag.i") and kind.startswith("INT"):
        return "integer" if lnclass in MEASUREMENT_CLASSES else None
    return None
