"""Write generated/scenarios.json: LicenseService runs (status/actions/files) for the Go parity test.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/licensing/testdata/make_scenarios.py
Each scenario drives one service through steps with a fixed clock; every step records the
HTTP answer exactly as api/license.py builds it, the client calls and every file's bytes.
"""
import json
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vector_common import APP, b64, flask_body, license_signer, write  # noqa: E402
from scenario_defs import PARTS, SCENARIOS  # noqa: E402
from license_fakes import FakeRegistry  # noqa: E402  (in-memory HKCU\Software\ScadaBench)
from studio_app.api import license as license_api  # noqa: E402
from studio_app.licensing import integrity  # noqa: E402
from studio_app.licensing.client import LicenseServerError  # noqa: E402
from studio_app.licensing.service import LicenseService  # noqa: E402
from studio_app.version import __version__  # noqa: E402


class FakeClient:
    """Canned license-server answers, recording every call."""

    def __init__(self, answers, calls):
        self.answers, self.calls = answers, calls

    def _answer(self, name, *args):
        self.calls.append([name, *args])
        queue = self.answers.get(name, [])
        result = queue.pop(0) if queue else {"raise": ["no canned answer", "offline"]}
        if "raise" in result:
            raise LicenseServerError(*result["raise"])
        return result["return"]

    def start_trial(self, hwid, parts=None, email=None, code=None):
        return self._answer("trial", hwid, parts, email, code)

    def request_trial_code(self, hwid, email):
        return self._answer("email-code", hwid, email)

    def activate(self, key, hwid):
        return self._answer("activate", key, hwid)

    def transfer(self, key, hwid):
        return self._answer("transfer", key, hwid)

    def verify(self, key, hwid):
        return self._answer("verify", key, hwid)


def answer(result):
    response, status = result if isinstance(result, tuple) else (result, 200)
    return {"http": status, "body": response.get_data(as_text=True)}


def run_step(service, step):
    op = step["op"]
    if op == "status":
        return {"http": 200, "body": flask_body({**service.status(), "app_version": __version__})}
    if op == "licensed":
        return {"value": service.is_licensed()}
    if op == "verify":
        return {"value": service.verify_online_if_due(trial_only=step.get("trial_only", False))}
    actions = {"trial": lambda: service.start_trial(step.get("email"), step.get("code")),
               "email_code": lambda: service.request_trial_code(step.get("arg", "")),
               "check": lambda: service.check_online_now(),
               "activate": lambda: service.activate(step.get("arg", ""), move=bool(step.get("move"))),
               "offline": lambda: service.activate_offline(step.get("arg", "")),
               "config": lambda: service.set_server_url(step.get("arg", ""))}
    with APP.app_context():
        return answer(license_api._run(actions[op]))


def snapshot(root):
    files = {}
    for path in sorted(root.rglob("*")):
        if path.is_file() and "native" not in path.relative_to(root).parts:
            files[path.relative_to(root).as_posix()] = b64(path.read_bytes())
    return files


def run(scenario):
    with tempfile.TemporaryDirectory() as folder:
        root = Path(folder)
        clock, calls = [0.0], []
        answers = {name: list(queue) for name, queue in scenario.get("client", {}).items()}
        native = root / "native"
        native.mkdir()
        (native / "studio_ied.exe").write_bytes(b"worker")
        integrity.debugger_attached = lambda: scenario.get("debugger", False)
        hwid = scenario.get("hwid", license_signer.HWID)

        def provider():
            if hwid is None:
                raise OSError("Machine licensing is available on Windows only")
            return hwid

        def factory(url):
            calls.append(["factory", url])
            return FakeClient(answers, calls)

        keys = {} if scenario.get("keys") == "none" else license_signer.keys()
        legacy = scenario.get("legacy")
        registry = FakeRegistry() if scenario.get("registry") else None
        service = LicenseService(root / "home", root / "data", scenario.get("enforce", True), hwid_provider=provider,
                                 clock=lambda: clock[0], keys=keys, client_factory=factory,
                                 native_folder=native, expected_files=scenario.get("expected"),
                                 legacy_hwid_provider=(lambda: legacy) if legacy else None,
                                 registry=registry, parts_provider=lambda: dict(PARTS))
        results = []
        for step in scenario["steps"]:
            clock[0] = step.get("now", clock[0])
            calls.clear()
            if step["op"] in ("write", "remove"):
                target = root / step["path"]
                target.parent.mkdir(parents=True, exist_ok=True)
                target.unlink(missing_ok=True)
                if step["op"] == "write":
                    target.write_bytes(step["data"].encode("utf-8") if "data" in step else bytes.fromhex(step["hex"]))
                results.append({})
                continue
            try:
                result = run_step(service, step)
            except Exception as error:  # noqa: BLE001 - an uncaught exception is the vector
                result = {"crash": type(error).__name__}
            result = {**result, "calls": [list(call) for call in calls], "files": snapshot(root)}
            if registry is not None:
                result["registry"] = dict(registry.values)
            results.append(result)
        return {**scenario, "results": results}


def main():
    write("scenarios.json", {"app_version": __version__, "scenarios": [run(scenario) for scenario in SCENARIOS]})
    json.loads((Path(__file__).resolve().parent / "generated" / "scenarios.json").read_text(encoding="utf-8"))


if __name__ == "__main__":
    main()
