"""Tagged JSON for values that cross the harness <-> engine boundary.

Plain JSON values stay plain, so a Go engine can answer the same inspect API. Python-only
types (bytes, set, tuple, path, non-string dict keys, exceptions) get a one-key tag. Objects
that cannot cross (for example a ProjectService handed to a patched function) become a
handle; touching a handle on the harness side means "run the original function instead".
"""
import base64
import builtins
import json
from pathlib import Path, PurePath

TAGS = ("__bytes__", "__set__", "__tuple__", "__path__", "__dict__", "__handle__", "__exc__")


class DelegateToOriginal(Exception):
    """A patched function touched an engine-side object: the engine runs the original."""


class Handle:
    """Stands for an engine-side object that cannot be copied into the test process."""

    def __init__(self, kind):
        object.__setattr__(self, "_kind", kind)

    def __getattr__(self, name):
        raise DelegateToOriginal(f"{self._kind}.{name}")

    def __repr__(self):
        return f"<engine {self._kind}>"


def encode(value):
    if value is None or isinstance(value, (bool, int, float, str)):
        return value
    if isinstance(value, (bytes, bytearray)):
        return {"__bytes__": base64.b64encode(bytes(value)).decode("ascii")}
    if isinstance(value, (set, frozenset)):
        return {"__set__": [encode(item) for item in sorted(value, key=repr)]}
    if isinstance(value, tuple):
        return {"__tuple__": [encode(item) for item in value]}
    if isinstance(value, PurePath):
        return {"__path__": str(value)}
    if isinstance(value, dict):
        if all(isinstance(key, str) and key not in TAGS for key in value):
            return {key: encode(item) for key, item in value.items()}
        return {"__dict__": [[encode(key), encode(item)] for key, item in value.items()]}
    if isinstance(value, list):
        return [encode(item) for item in value]
    if isinstance(value, BaseException):
        return {"__exc__": encode_exception(value)}
    return {"__handle__": type(value).__name__}


def decode(value):
    if isinstance(value, list):
        return [decode(item) for item in value]
    if not isinstance(value, dict):
        return value
    if len(value) == 1:
        tag, inner = next(iter(value.items()))
        if tag == "__bytes__":
            return base64.b64decode(inner)
        if tag == "__set__":
            return {decode(item) for item in inner}
        if tag == "__tuple__":
            return tuple(decode(item) for item in inner)
        if tag == "__path__":
            return Path(inner)
        if tag == "__dict__":
            return {decode(key): decode(item) for key, item in inner}
        if tag == "__handle__":
            return Handle(inner)
        if tag == "__exc__":
            return decode_exception(inner)
    return {key: decode(item) for key, item in value.items()}


def encode_exception(error):
    record = {"type": type(error).__name__, "module": type(error).__module__,
              "args": [encode(arg) for arg in error.args]}
    if hasattr(error, "code") and type(error).__name__ == "HTTPError":
        record["http"] = {"url": getattr(error, "url", ""), "code": error.code, "msg": str(getattr(error, "msg", ""))}
    return record


def decode_exception(record):
    """Rebuild the exception class the fake raised; unknown classes become RuntimeError."""
    args = [decode(arg) for arg in record.get("args", [])]
    if "http" in record:
        from urllib.error import HTTPError
        http = record["http"]
        return HTTPError(http["url"], http["code"], http["msg"], {}, None)
    kind = getattr(builtins, record.get("type", ""), None)
    if isinstance(kind, type) and issubclass(kind, BaseException):
        return kind(*args)
    if record.get("module") == "urllib.error" and record.get("type") == "URLError":
        from urllib.error import URLError
        return URLError(*args)
    return RuntimeError(*args)


def dumps(value):
    return json.dumps(encode(value))


def loads(text):
    return decode(json.loads(text))
