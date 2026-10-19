"""Shared golden helpers: exact JSON encoding, error capture, id/source normalisation, XSD recording."""
import hashlib
import json
import pathsetup


def dumps(value):
    """Python json.dumps defaults (', ' / ': ', ensure_ascii) as ASCII bytes, no trailing newline."""
    return json.dumps(value).encode("ascii")


def write_json(folder, name, value, files):
    pathsetup.write_bytes(folder / name, dumps(value))
    files.append(name)


def error_dict(error):
    return {"type": type(error).__name__, "message": str(error),
            "code": getattr(error, "code", None), "details": getattr(error, "details", None)}


def capture(function, *args):
    """(True, result) or (False, error dict) for any exception the frozen code raises."""
    try:
        return True, function(*args)
    except Exception as error:  # goldens record every failure class exactly as Python raises it
        return False, error_dict(error)


def normalise(ieds):
    """Replace random ids and the bulky source text; return the cfg texts separately."""
    cfgs = []
    for index, ied in enumerate(ieds):
        ied["id"] = f"ied-{index}"
        ied["source"] = "sha256:" + hashlib.sha256(ied["source"].encode("utf-8")).hexdigest()
        cfgs.append(ied["cfg"])
    return cfgs


class XsdRecorder:
    """Wrap validate_schema where the importer/exporter bound it, to seed the Go XSD cache."""

    def __init__(self):
        self.entries = {}

    def install(self):
        from studio_app.scl import exporter, importer, schema_validation
        original = schema_validation.validate_schema

        def recording(source, root=None):
            result = original(source, root)
            data = source.encode("utf-8") if isinstance(source, str) else source
            if result.get("schema") is not None:
                key = (hashlib.sha256(data).hexdigest(), result["schema"])
                self.entries[key] = {"sha256": key[0], "schema": key[1], "status": result["status"],
                                     "errors": [{"line": item["line"], "message": item["message"]}
                                                for item in result["errors"]]}
            return result

        importer.validate_schema = recording
        exporter.validate_schema = recording
        return recording

    def seed(self):
        return [self.entries[key] for key in sorted(self.entries)]
