"""Fakes for http_gen.py: license server, update server, Windows network calls, runtime.

Nothing here touches the network, an adapter, a firewall rule or UAC: every call answers
from the step data the generator (and the Go replay) share.
"""
import io
import json
import threading
from pathlib import Path
from urllib.error import HTTPError, URLError

from studio_app.licensing.client import LicenseServerError
from studio_app.networking.elevation import ElevationCancelled
from studio_app.networking.journal import item_key

ERRORS = {"RuntimeError": RuntimeError, "ValueError": ValueError, "KeyError": KeyError}


class LicenseClient:
    """Canned license-server answers per call name (trial, activate, transfer, verify)."""

    def __init__(self):
        self.answers = {}

    def _answer(self, name):
        queue = self.answers.get(name, [])
        result = queue.pop(0) if queue else {"raise": ["no canned answer", "offline"]}
        if "raise" in result:
            raise LicenseServerError(*result["raise"])
        return json.loads(json.dumps(result["return"]))

    def start_trial(self, hwid):
        return self._answer("trial")

    def activate(self, key, hwid):
        return self._answer("activate")

    def transfer(self, key, hwid):
        return self._answer("transfer")

    def verify(self, key, hwid):
        return self._answer("verify")


class _Response(io.BytesIO):
    def __init__(self, body, etag):
        super().__init__(body)
        self.headers = {} if etag is None else {"ETag": etag}


class UpdateServer:
    """The update opener: answers {"body", "etag"}, {"http": code} or {"urlerror": true}."""

    def __init__(self):
        self.queue = []

    def __call__(self, request, timeout):
        item = self.queue.pop(0) if self.queue else {"urlerror": True}
        if item.get("urlerror"):
            raise URLError("no route")
        if "http" in item:
            raise HTTPError(request.full_url, item["http"], "x", {}, io.BytesIO(b""))
        return _Response(item["body"].encode("utf-8"), item.get("etag"))


class Windows:
    """The Windows state a scenario sets with "net" steps; reads may raise "!Class:message"."""

    def __init__(self, env):
        self.env = env

    def reader(self, key):
        def read(*args):
            value = self.env[key]
            if isinstance(value, str) and value.startswith("!"):
                kind, message = value[1:].split(":", 1)
                raise ERRORS[kind](message)
            return json.loads(json.dumps(value))
        return read

    def probe(self, addresses):
        return {address for address in addresses if address in self.env["probe"]}

    def firewall(self, names):
        state = self.env["state"]
        return set(state["gateway"]), set(state["disabled"])

    def elevate(self, request, digest):
        """Stands in for the UAC helper: writes the result file the mode asks for."""
        mode = self.env["elevate"]
        if mode == "cancel":
            raise ElevationCancelled("Windows administrator permission was not given, so nothing was changed")
        if mode == "runtime":
            raise RuntimeError("The network helper did not finish within two minutes")
        if mode == "none":
            return
        items = json.loads(Path(request).read_bytes())["items"]
        results = [{"key": item_key(item), "ok": not (mode == "fail" and n == 1),
                    "message": "Access denied" if mode == "fail" and n == 1 else ""} for n, item in enumerate(items)]
        outcome = {"refused": True, "reason": "Unknown helper action", "results": []} if mode == "refused" else \
            {"refused": False, "results": results}
        Path(request).with_suffix(".result.json").write_text(json.dumps(outcome))


class Runtime:
    """The coordinator as the network routes use it: state and cycle_lock."""

    def __init__(self):
        self.state = "stopped"
        self.cycle_lock = threading.RLock()


class Monotonic:
    """time.monotonic for the adapter cache, moved by "tick" steps."""
    now = 1000.0

    @classmethod
    def monotonic(cls):
        return cls.now
