"""Prepare the studio_app copy that the protected build compiles.

The repository source is never edited: the copy receives the native integrity seal
(SHA-256 of each shipped worker) and, for sandbox builds, the local test server key.
"""
import importlib.util
from pathlib import Path
import re
import shutil
import subprocess

STUDIO = Path(__file__).resolve().parents[1]
PACKAGE = STUDIO / "backend/studio_app"
SEALED = ("studio_ied.exe", "studio_live.exe", "studio_probe.exe", "studio_gateway.exe", "studio_control.exe",
          "studio_master.exe")
SANDBOX = STUDIO / "build/license-sandbox"
SANDBOX_URL = "http://127.0.0.1:8090"


def _configure_release():
    spec = importlib.util.spec_from_file_location("studio_configure_release", STUDIO / "tools/configure_release.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def sandbox_release():
    """Return (url, [public key hex]) of the local sandbox license server."""
    exe, data = SANDBOX / "license-server.exe", SANDBOX / "server-data"
    if not exe.exists() or not (data / "signing.key").exists():
        raise SystemExit("Start the local sandbox once first: python studio/tools/license_sandbox.py")
    status = subprocess.run([str(exe), "status", "--data", str(data)], capture_output=True, text=True, check=True)
    return SANDBOX_URL, [status.stdout.split("--public-key", 1)[1].split()[0]]


def write_native_header(folder, keys):
    folder.mkdir(parents=True, exist_ok=True)
    (folder / "release_key.hpp").write_text(_configure_release().native_source(keys), encoding="utf-8", newline="\n")
    return folder


def integrity_source(native_bin):
    from studio_app.licensing.integrity import file_digest
    template = (PACKAGE / "licensing/integrity.py").read_text(encoding="utf-8")
    rows = "".join(f'    "{name}": "{file_digest(Path(native_bin) / name)}",\n' for name in SEALED)
    return template.replace("EXPECTED = {}\n", "EXPECTED = {\n" + rows + "}\n", 1)


def version_source(version):
    """Return version.py text with the placeholder replaced by the real release version."""
    template = (PACKAGE / "version.py").read_text(encoding="utf-8")
    result, count = re.subn(r'__version__ = "[0-9.]+\+dev"', f'__version__ = "{version}"', template, count=1)
    if count != 1:
        raise RuntimeError("version.py has no development version to replace")
    return result


def prepare(destination, native_bin, release=None, version=None):
    """Copy studio_app to destination/studio_app; seal it; optionally set the release key."""
    target = Path(destination) / "studio_app"
    if target.exists():
        shutil.rmtree(target)
    shutil.copytree(PACKAGE, target, ignore=shutil.ignore_patterns("__pycache__", "*.pyc"))
    (target / "licensing/integrity.py").write_text(integrity_source(native_bin), encoding="utf-8", newline="\n")
    if version:
        (target / "version.py").write_text(version_source(version), encoding="utf-8", newline="\n")
    if release is not None:
        url, keys = release
        source = _configure_release().python_source(url, keys)
        (target / "licensing/release_key.py").write_text(source, encoding="utf-8", newline="\n")
    return target
