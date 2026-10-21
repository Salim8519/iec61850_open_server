"""Turn the Python objects a test hands to create_app into the seams file (harness side).

A real LicenseService / UpdateService becomes data (home, hwid, frozen clock, test keys), so
the engine under test runs its own licensing and update code. A duck-typed fake becomes an
object seam whose methods call back into the test. Outbound HTTP (the update opener) and
monkeypatched studio_app functions become callback seams.
"""
import time
from urllib.request import Request


def _hex_keys(keys):
    return None if keys is None else {kid: bytes(value).hex() for kid, value in keys.items()}


def _frozen_clock(clock):
    return None if clock is time.time else clock()


def object_seam(value, callbacks):
    attrs, methods = {}, {}
    for name in dir(value):
        if name.startswith("_"):
            continue
        member = getattr(value, name)
        if callable(member):
            methods[name] = callbacks.register(member)
        elif member is None or isinstance(member, (bool, int, float, str, list, dict)):
            attrs[name] = member
    return {"object": {"attrs": attrs, "methods": methods}}


def license_seam(service, callbacks):
    if service is None:
        return None
    from studio_app.licensing.client import LicenseClient
    from studio_app.licensing.service import LicenseService
    if not isinstance(service, LicenseService):
        return object_seam(service, callbacks)
    if service.client_factory is not LicenseClient or service._legacy_hwid_provider is not None:
        raise NotImplementedError("parity: client_factory/legacy_hwid seams are not translated yet")
    runs = service.anchor.runs_file
    return {"home": str(service.home), "data_dir": str(runs.parent.parent) if runs else str(service.home),
            "runs_file": str(runs) if runs else None, "enforce": service.enforce,
            "hwid": service._hwid_provider() if service._hwid_provider else None,
            "clock": _frozen_clock(service.clock), "keys": _hex_keys(service.keys),
            "native_folder": str(service.native_folder) if service.native_folder else None,
            "expected_files": service.expected_files, "allow_test_server": service.allow_test_server}


def update_seam(service, callbacks):
    if service is None:
        return None
    from studio_app.licensing.client import open_https
    from studio_app.updates.service import UpdateService
    if not isinstance(service, UpdateService):
        return object_seam(service, callbacks)
    opener = None if service.opener is open_https else callbacks.register(opener_bridge(service.opener))
    return {"directory": str(service.path.parent), "server": service.server_url(), "current": service.current,
            "keys": _hex_keys(service.keys), "clock": _frozen_clock(service.clock), "opener": opener}


def opener_bridge(opener):
    """Run the test's canned opener for a request the engine wanted to send."""
    def bridge(record, timeout=None):
        request = Request(record["url"], data=record.get("data"), headers=record.get("headers") or {},
                          method=record.get("method"))
        with opener(request, timeout=timeout) as response:
            return {"body": response.read(), "headers": dict(response.headers)}
    return bridge


def fault_seams(pending, callbacks):
    return {target: callbacks.register(function) for target, function in pending}
