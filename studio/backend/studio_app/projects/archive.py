"""Read and write versioned portable archives with resource and path limits."""
import io
import json
import zipfile
from studio_app.contracts.errors import StudioError
from .document import COMPACT_FORMATS, compact, expand

MAX_UNPACKED = 100 * 1024 * 1024
MAX_LEGACY_UNPACKED = 256 * 1024 * 1024


def encode(project):
    document = json.dumps(compact(project), allow_nan=False).encode("utf-8")
    if len(document) > MAX_UNPACKED:
        raise StudioError("Engineering project exceeds the 100 MB save limit; nothing was written")
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("project.json", document)
    return stream.getvalue()


def decode(payload):
    if len(payload) > MAX_UNPACKED:
        raise StudioError("Project archive exceeds 100 MB")
    try:
        with zipfile.ZipFile(io.BytesIO(payload)) as archive:
            entries = archive.infolist()
            if len(entries) != 1 or entries[0].filename != "project.json":
                raise StudioError("Expected a single project.json entry")
            if entries[0].file_size > MAX_LEGACY_UNPACKED:
                raise StudioError("Unpacked project exceeds the 256 MB legacy recovery limit")
            result = json.loads(archive.read("project.json"))
            if (isinstance(result, dict) and result.get("format") in COMPACT_FORMATS
                    and entries[0].file_size > MAX_UNPACKED):
                raise StudioError("Compact project exceeds 100 MB")
    except (zipfile.BadZipFile, json.JSONDecodeError) as error:
        raise StudioError("Invalid project archive") from error
    if not isinstance(result, dict):
        raise StudioError("Unsupported project format")
    return expand(result)
