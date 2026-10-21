"""Record the frozen Flask app's answers for the platformapi routes (http_golden.json).

  STUDIO_PYTHON -I -B studio/engine/internal/api/platformapi/testdata/http_gen.py

The real create_app() answers every request through its test client. Only the outside world
is faked (gen_fakes.py): the license server, the update server, the adapter list, firewall
queries and the UAC helper. Certificates and release answers are signed with the test seed
of studio/tests/unit/license_signer.py. Clocks are fixed by "now"/"tick" steps. The golden
file lists every step in order; TestPlatformRoutesAgainstFlask replays it against Go.
"""
import json
import os
import sys
import tempfile
import types
from pathlib import Path

HERE = Path(__file__).resolve().parent
STUDIO = HERE.parents[4]
sys.path[:0] = [str(HERE), str(STUDIO / "backend"), str(STUDIO / "tests" / "unit")]
os.environ["SCADABENCH_UPDATE_CHECK"] = "off"

import license_signer  # noqa: E402  (test-only signer)
from gen_fakes import LicenseClient, Monotonic, Runtime, UpdateServer, Windows  # noqa: E402
from gen_license import license_steps, update_steps  # noqa: E402
from gen_network import DEFAULT_ENV, GATEWAY, PROGRAM, network_steps  # noqa: E402
from studio_app.api import network as network_api  # noqa: E402
from studio_app.licensing import integrity  # noqa: E402
from studio_app.licensing.release_key import key_id  # noqa: E402
from studio_app.licensing.service import LicenseService  # noqa: E402
from studio_app.networking import npcap, service as network_module  # noqa: E402
from studio_app.networking.gateway_firewall import GatewayFirewall  # noqa: E402
from studio_app.networking.service import NetworkService  # noqa: E402
from studio_app.updates.service import UpdateService  # noqa: E402
from studio_app.version import __version__  # noqa: E402

SERVER = "https://license.scadabench.com"
CURRENT = "0.1.13"
SKIP_HEADERS = {"content-length", "date", "set-cookie", "vary", "server"}


class _Off:
    """Stands in for license/update services while the app is built (no threads, no gate)."""
    enforce = False

    def start_background_verify(self):
        pass

    def start_background(self):
        pass


class Session:
    """The app, its faked surroundings and the values later steps refer to ("$fp", ...)."""

    def __init__(self, app, root):
        self.app, self.root, self.client = app, root, app.test_client()
        self.clock, self.last, self.licenses = [0], {}, 0
        self.license_client, self.server = LicenseClient(), UpdateServer()
        self.windows = Windows(json.loads(json.dumps(DEFAULT_ENV)))
        (root / "net").mkdir()
        (root / "updates").mkdir()
        (root / "sysroot").mkdir()
        network = NetworkService(app.extensions["projects"], root / "net", PROGRAM,
                                 adapters=self.windows.reader("adapters"), rules=self.windows.reader("rules"),
                                 elevate=self.windows.elevate, probe=self.windows.probe, firewall=self.windows.firewall)
        gateway = GatewayFirewall(network, GATEWAY, rules=self.windows.reader("gateway_rules"),
                                  blocking=self.windows.reader("blocking"))
        updates = UpdateService(root / "updates", lambda: SERVER, current=CURRENT, keys=license_signer.keys(),
                                opener=self.server, clock=lambda: self.clock[0])
        self.coordinator = app.extensions["runtime"]
        self.runtime = Runtime()
        app.extensions.update(network=network, gateway_firewall=gateway, updates=updates, runtime=self.runtime)

    def resolve(self, value):
        if isinstance(value, str) and value.startswith("$"):
            return self.last[value]
        if isinstance(value, list):
            return [self.resolve(item) for item in value]
        if isinstance(value, dict):
            return {key: self.resolve(item) for key, item in value.items()}
        return value

    def license_root(self):
        return self.root / f"license{self.licenses}"

    def new_license(self, enforce):
        self.licenses += 1
        native = self.license_root() / "native"
        native.mkdir(parents=True)
        (native / "studio_ied.exe").write_bytes(b"worker")
        self.app.extensions["license"] = LicenseService(
            self.license_root() / "home", self.license_root() / "data", enforce,
            hwid_provider=lambda: license_signer.HWID, clock=lambda: self.clock[0], keys=license_signer.keys(),
            client_factory=lambda url: self.license_client, native_folder=native)

    def setup(self, step):
        kind = step["op"]
        if kind == "now":
            self.clock[0] = step["value"]
        elif kind == "license":
            self.new_license(step["enforce"])
        elif kind == "license_file":
            target = self.license_root() / step["path"]
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(bytes.fromhex(step["hex"]) if "hex" in step else step["data"].encode("utf-8"))
        elif kind == "client":
            self.license_client.answers = json.loads(json.dumps(step["answers"]))
        elif kind == "answers":
            self.server.queue = list(step["queue"])
        elif kind == "update_file":
            (self.root / "updates" / "update_check.json").write_bytes(step["data"].encode("utf-8"))
        elif kind == "runtime":
            self.runtime.state = step["state"]
        elif kind == "ieds":
            projects = self.app.extensions["projects"]
            with projects.lock:
                projects.project["ieds"] = json.loads(json.dumps(step["value"]))
                projects.save()
        elif kind == "net":
            step["value"] = self.resolve(step["value"])
            self.windows.env[step["key"]] = step["value"]
        elif kind == "tick":
            Monotonic.now += step["seconds"]
        else:
            raise ValueError(kind)
        return step

    def request(self, step):
        kwargs = {}
        if "json" in step:
            step["body"], step["content_type"] = json.dumps(self.resolve(step.pop("json"))), "application/json"
        if "body" in step:
            kwargs["data"] = step["body"].encode("utf-8")
            if "content_type" in step:
                kwargs["content_type"] = step["content_type"]
        response = self.client.open(step["path"], method=step["method"], **kwargs)
        text = response.get_data(as_text=True)
        self.remember(step["path"], response.status_code, text)
        return {**step, "status": response.status_code, "response": text,
                "headers": {k: v for k, v in response.headers.items() if k.lower() not in SKIP_HEADERS}}

    def remember(self, path, status, text):
        if status != 200:
            return
        tags = {"/api/v1/network/plan/preview": ("$fp", "$rules", "firewall"),
                "/api/v1/network/cleanup/preview": ("$clean", None, None),
                "/api/v1/network/gateway-firewall": ("$gw", "$gwrules", "gateway_firewall")}
        if path not in tags or "fingerprint" not in text:
            return
        body = json.loads(text)
        fingerprint, names, kind = tags[path]
        self.last[fingerprint] = body["fingerprint"]
        if names:
            self.last[names] = [item["name"] for item in body["items"] if item["kind"] == kind]


def main():
    integrity.debugger_attached = lambda: False
    network_module.time = types.SimpleNamespace(monotonic=Monotonic.monotonic)
    from studio_app.app_factory import create_app
    from studio_app.settings import Settings
    with tempfile.TemporaryDirectory() as folder:
        root = Path(folder)
        network_api.npcap_status = lambda: npcap.npcap_status(system_root=str(root / "sysroot"))
        app = create_app(Settings(root, root / "unused.exe", root), "platform-golden",
                         license_service=_Off(), update_service=_Off())
        session = Session(app, root)
        session.client.post("/api/v1/session", json={"token": "platform-golden"})
        app.extensions["startup"].wait()
        steps = []
        try:
            for step in network_steps() + update_steps() + license_steps():
                steps.append(session.request(step) if step["op"] == "request" else session.setup(step))
        finally:
            session.coordinator.close()
    public = license_signer.public_key()
    golden = {"app_version": __version__, "current": CURRENT, "server_url": SERVER, "hwid": license_signer.HWID,
              "program": PROGRAM, "gateway": GATEWAY, "network_env": DEFAULT_ENV,
              "test_key": {"kid": key_id(public), "public": public.hex()}, "steps": steps}
    (HERE / "generated").mkdir(exist_ok=True)
    text = json.dumps(golden, indent=1, ensure_ascii=False)
    (HERE / "generated" / "http_golden.json").write_text(text + "\n", encoding="utf-8", newline="\n")
    print(sum(step["op"] == "request" for step in steps), "requests,", len(steps), "steps")


if __name__ == "__main__":
    main()
