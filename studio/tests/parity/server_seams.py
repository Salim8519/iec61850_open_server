"""Engine side of the seams file (python-http). The Go parity build reads the same schema.

{"token", "secret_key", "admin_origin", "parity_key",
 "license": {"home", "data_dir", "enforce", "hwid", "clock", "keys": {kid: hex}, "native_folder",
             "expected_files"} | {"object": <object seam>} | null,
 "updates": {"directory", "server", "current", "keys": {kid: hex}, "clock", "opener": <callback>} | null,
 "faults": {"studio_app.dotted.target": <callback>}}
<callback> = {"url", "key", "id"} (parity.callbacks); <object seam> = {"attrs": {...}, "methods": {name: <callback>}}.
"""
import importlib
import io
from pathlib import Path
from urllib.request import Request

from parity.callbacks import remote_function


def resolve(dotted):
    """The object a dotted name points at: longest importable module, then attributes."""
    parts = dotted.split(".")
    for size in range(len(parts), 0, -1):
        try:
            value = importlib.import_module(".".join(parts[:size]))
        except ImportError:
            continue
        for name in parts[size:]:
            value = getattr(value, name)
        return value
    raise ImportError(dotted)


def apply_faults(faults):
    for target, reference in faults.items():
        owner_name, name = target.rsplit(".", 1)
        owner = resolve(owner_name)
        setattr(owner, name, remote_function(reference, getattr(owner, name)))


def object_seam(record):
    """A duck-typed fake: copied data attributes, methods that call back into the test."""
    namespace = dict(record.get("attrs") or {})
    for name, reference in (record.get("methods") or {}).items():
        call = remote_function(reference)
        namespace[name] = staticmethod(call)
    return type("ParityObjectSeam", (), namespace)()


def _keys(table):
    return None if table is None else {kid: bytes.fromhex(value) for kid, value in table.items()}


def _constant(value):
    return lambda: value


def license_service(seam):
    if not seam:
        return None
    if "object" in seam:
        return object_seam(seam["object"])
    from studio_app.licensing.service import LicenseService
    options = {"keys": _keys(seam.get("keys")), "native_folder": _path(seam.get("native_folder")),
               "expected_files": seam.get("expected_files")}
    if seam.get("hwid") is not None:
        options["hwid_provider"] = _constant(seam["hwid"])
    if seam.get("clock") is not None:
        options["clock"] = _constant(seam["clock"])
    service = LicenseService(Path(seam["home"]), Path(seam["data_dir"]), seam["enforce"], **options)
    if seam.get("runs_file") is None:
        service.anchor.runs_file = None
    return service


def update_service(seam):
    if not seam:
        return None
    if "object" in seam:
        return object_seam(seam["object"])
    from studio_app.updates.service import UpdateService
    options = {"current": seam["current"], "keys": _keys(seam.get("keys"))}
    if seam.get("clock") is not None:
        options["clock"] = _constant(seam["clock"])
    if seam.get("opener"):
        options["opener"] = remote_opener(seam["opener"])
    return UpdateService(Path(seam["directory"]), _constant(seam.get("server")), **options)


class _Answer(io.BytesIO):
    def __init__(self, body, headers):
        super().__init__(body)
        self.headers = headers


def remote_opener(reference):
    """urllib opener whose answers come from the test's canned opener (outbound HTTP seam)."""
    call = remote_function(reference)

    def opener(request: Request, timeout=None):
        answer = call({"url": request.full_url, "method": request.get_method(),
                       "headers": dict(request.header_items()), "data": request.data}, timeout)
        return _Answer(answer["body"], answer["headers"])
    return opener


def _path(value):
    return None if value is None else Path(value)
