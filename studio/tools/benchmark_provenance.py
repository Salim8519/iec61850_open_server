"""Record reproducible local source/binary identity without including customer data."""
import hashlib
from importlib.metadata import version
import json
import platform
import sys


def capture(studio):
    roots = (studio / "backend", studio / "native/worker", studio / "native/probe", studio / "native/cmake",
             studio / "frontend/src", studio / "tools", studio / "schemas")
    suffixes = {".py", ".cpp", ".h", ".hpp", ".ts", ".tsx", ".css", ".xsd", ".cmake"}
    paths = {path for root in roots for path in root.rglob("*")
             if path.is_file() and path.suffix in suffixes}
    paths.update(studio / name for name in ("native/CMakeLists.txt", "requirements.txt",
                                          "frontend/package-lock.json", "pyproject.toml",
                                          "native/build/Release/studio_ied.exe"))
    files = {path.relative_to(studio).as_posix(): hashlib.sha256(path.read_bytes()).hexdigest()
             for path in sorted(paths) if path.is_file()}
    identity = hashlib.sha256(json.dumps(files, sort_keys=True).encode()).hexdigest()
    return {"sha256": identity, "files": files, "python": sys.version, "platform": platform.platform(),
            "dependencies": {name: version(name) for name in ("Flask", "waitress", "lxml", "defusedxml")}}


def changed_files(before, after):
    paths = before["files"].keys() | after["files"].keys()
    return sorted(path for path in paths if before["files"].get(path) != after["files"].get(path))
