"""pytest plugin loaded by tests/conftest.py only when STUDIO_ENGINE is an HTTP engine.

- installs remote_create_app before any test module imports create_app;
- records monkeypatch.setattr on studio_app targets as fault seams for the engine;
- shuts every engine of a test down afterwards and fails on orphan studio_*.exe workers;
- skips `inprocess` tests (counted by `pytest -rs`), hashes the native binaries once and
  aborts if they change mid-session (no stale natives).
"""
import hashlib
import json
import os
from pathlib import Path
import types

import pytest
from _pytest.monkeypatch import MonkeyPatch

import studio_app.app_factory as factory
from parity import launch
from parity import remote_app

factory.create_app = remote_app.remote_create_app
NATIVE_BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", launch.STUDIO / "native/build/Release"))
_original_setattr = MonkeyPatch.setattr
_NOTSET = object()


def _studio_target(target, name):
    if isinstance(target, str):
        return target if target.startswith("studio_app.") else None
    if isinstance(target, types.ModuleType) and target.__name__.startswith("studio_app"):
        return f"{target.__name__}.{name}"
    if isinstance(target, type) and target.__module__.startswith("studio_app"):
        return f"{target.__module__}.{target.__qualname__}.{name}"
    return None


def _recording_setattr(self, target, name, value=_NOTSET, raising=True):
    if value is _NOTSET:
        dotted, function = _studio_target(target, None), name
    else:
        dotted, function = _studio_target(target, name), value
    result = _original_setattr(self, target, name, raising=raising) if value is _NOTSET \
        else _original_setattr(self, target, name, value, raising=raising)
    if dotted and callable(function):
        remote_app.PENDING_FAULTS.append((dotted, function))
        for app in remote_app.LIVE:
            app.install_fault(dotted, function)
    return result


MonkeyPatch.setattr = _recording_setattr


def _natives():
    return sorted(NATIVE_BIN.glob("studio_*.exe")) if NATIVE_BIN.is_dir() else []


def native_hashes():
    return {path.name: hashlib.sha256(path.read_bytes()).hexdigest() for path in _natives()}


def native_stamps():
    return {path.name: (path.stat().st_size, path.stat().st_mtime_ns) for path in _natives()}


_HASHES = pytest.StashKey[dict]()
_STAMPS = pytest.StashKey[dict]()


def pytest_configure(config):
    config.stash[_HASHES], config.stash[_STAMPS] = native_hashes(), native_stamps()


def pytest_collection_modifyitems(config, items):
    engine = launch.engine_name()
    for item in items:
        if item.get_closest_marker("inprocess"):
            item.add_marker(pytest.mark.skip(reason=f"inprocess under {engine}"))


@pytest.fixture(scope="session", autouse=True)
def _parity_session(request, record_testsuite_property):
    record_testsuite_property("studio_engine", launch.engine_name())
    record_testsuite_property("native_hashes", json.dumps(request.config.stash[_HASHES], sort_keys=True))
    yield


@pytest.fixture(autouse=True)
def _parity_engines(request):
    stamps = native_stamps()
    if stamps != request.config.stash[_STAMPS] and native_hashes() != request.config.stash[_HASHES]:
        pytest.exit("native binaries changed during the parity session (stale natives)", returncode=3)
    remote_app.PENDING_FAULTS.clear()
    yield
    remote_app.PENDING_FAULTS.clear()
    orphans = {}
    while remote_app.LIVE:
        orphans.update(remote_app.LIVE.pop().shutdown())
    if orphans:
        pytest.fail(f"orphan processes after the test: {orphans}")
