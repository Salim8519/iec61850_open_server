"""Identify the original SCL documents stored with the project's IEDs."""
import hashlib
from studio_app.contracts.errors import StudioError


def source_id(text):
    return hashlib.sha256(text.encode("utf-8")).hexdigest()[:16]


def stored_sources(project):
    """{source id: {id, filename, ieds, text}} for every distinct original file."""
    found = {}
    for ied in project["ieds"]:
        key = source_id(ied["source"])
        entry = found.setdefault(key, {"id": key, "filename": ied["filename"], "ieds": [], "text": ied["source"]})
        entry["ieds"].append(ied["name"])
    return found


def require_source(project, key):
    entry = stored_sources(project).get(key)
    if entry is None:
        raise StudioError("This SCL file is not in the project", code="not_found")
    return entry
