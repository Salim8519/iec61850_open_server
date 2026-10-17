"""Compact engineering archives: original XML once, derived runtime models rebuilt."""
import hashlib
from studio_app.contracts.errors import StudioError

IED_FIELDS = ("id", "name", "filename", "ip", "port", "role", "live", "lab_ip", "goose_publish", "test_mode")
CURRENT_FORMAT = 13  # 13: gateway simulator section (ADR-0057)
COMPACT_FORMATS = set(range(2, CURRENT_FORMAT + 1))


def compact(project):
    omitted = {"ieds", "format", "scl_sources", "migration_notice"}
    result = {key: value for key, value in project.items() if key not in omitted}
    result.update(format=CURRENT_FORMAT, ieds=[], scl_sources={})
    for ied in project.get("ieds", []):
        source = ied["source"]
        digest = hashlib.sha256(source.encode("utf-8")).hexdigest()
        result["scl_sources"][digest] = source
        result["ieds"].append({**{key: ied[key] for key in IED_FIELDS if key in ied}, "source_id": digest})
    return result


def expand(document):
    if document.get("format") == 1:
        return document
    if document.get("format") not in COMPACT_FORMATS:
        raise StudioError("Unsupported project format")
    sources = document.get("scl_sources")
    ieds = document.get("ieds")
    if not isinstance(sources, dict) or len(sources) > 20 or not isinstance(ieds, list) or len(ieds) > 20:
        raise StudioError("Invalid compact project source table")
    for digest, source in sources.items():
        if not isinstance(source, str) or hashlib.sha256(source.encode("utf-8")).hexdigest() != digest:
            raise StudioError("Project SCL source checksum mismatch")
    restored = []
    for ied in ieds:
        if not isinstance(ied, dict) or not isinstance(ied.get("source_id"), str) or ied["source_id"] not in sources:
            raise StudioError("IED refers to a missing project SCL source")
        restored.append({**{key: ied[key] for key in IED_FIELDS if key in ied}, "source": sources[ied["source_id"]]})
    return {**{key: value for key, value in document.items() if key not in {"ieds", "scl_sources"}},
            "ieds": restored}
