"""Write generated/service.json: NetworkService and GatewayFirewall flows with fake Windows calls.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/netcfg/testdata/make_service.py
Every step records Flask's body (or the exception), and each scenario records the helper
launches (argv, request bytes), the journal bytes, the project changes and the probe calls.
Nothing is elevated and no Windows setting is read or changed.
"""
import copy
import hashlib
import json
import sys
import tempfile
import types
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vector_common import GATEWAY, PROGRAM, b64, flask_body, outcome, write  # noqa: E402
from service_scenarios import DEFAULT_ENV, SCENARIOS  # noqa: E402
from make_journal import FixedClock, TIMES  # noqa: E402
from studio_app.networking import elevation, journal, service as service_module  # noqa: E402
from studio_app.networking.change_plan import gateway_plan  # noqa: E402
from studio_app.networking.elevation import ElevationCancelled  # noqa: E402
from studio_app.networking.gateway_firewall import GatewayFirewall  # noqa: E402
from studio_app.networking.journal import item_key  # noqa: E402

EXE = r"C:\Program Files\ScadaBench\ScadaBench.exe"
GWRULE = gateway_plan(GATEWAY)["items"][0]["name"]
ERRORS = {"RuntimeError": RuntimeError, "ValueError": ValueError, "KeyError": KeyError}


class Projects:
    """The project service as NetworkService uses it; records every change."""

    def __init__(self, ieds, events):
        self.project = {"ieds": copy.deepcopy(ieds), "network": {"mode": "loopback"}}
        self.lock, self.events = __import__("threading").RLock(), events

    def find_ied(self, key):
        ied = next(item for item in self.project["ieds"] if item["id"] == key)
        return Recorder(ied, self.events)


class Recorder(dict):
    def __init__(self, ied, events):
        super().__init__(ied)
        self.events, self.id = events, ied["id"]

    def __setitem__(self, key, value):
        self.events.append(["set", self.id, key, value])


class Clock:
    now = 1000.0

    @classmethod
    def monotonic(cls):
        return cls.now


def resolve(value, env, last):
    if isinstance(value, str) and value.startswith("$"):
        if value == "$gwrule":
            return GWRULE
        if value == "$rule":
            return [item["name"] for item in last.get("$fp_items", []) if item["kind"] == "firewall"]
        return last.get(value, "")
    if isinstance(value, list):
        return [resolve(item, env, last) for item in value]
    if isinstance(value, dict):
        return {key: resolve(item, env, last) for key, item in value.items()}
    return value


def fake_env(env, record, folder):
    def readers(key):
        def read(*args):
            value = env[key]
            if isinstance(value, str) and value.startswith("!"):
                kind, message = value[1:].split(":", 1)
                raise ERRORS[kind](message)
            return json.loads(json.dumps(value))
        return read

    def probe(addresses):
        record["probes"].append(list(addresses))
        return {address for address in addresses if address in env["probe"]}

    def firewall(names):
        record["firewall"].append(list(names))
        return set(env["state"]["gateway"]), set(env["state"]["disabled"])

    def elevate(request, digest):
        exe, argv = elevation.helper_command(request, digest, PROGRAM)
        payload = Path(request).read_bytes()
        record["launches"].append({"argv": [exe] + [a.replace(folder, "<DATA>") for a in argv], "payload": b64(payload)})
        mode = env["elevate"]
        if mode == "cancel":
            raise ElevationCancelled("Windows administrator permission was not given, so nothing was changed")
        if mode == "runtime":
            raise RuntimeError("The network helper did not finish within two minutes")
        items = json.loads(payload)["items"]
        results = [{"key": item_key(item), "ok": not (mode == "fail" and n == 1), "message": "Access denied" if
                    mode == "fail" and n == 1 else ""} for n, item in enumerate(items)]
        if mode == "no_ok":
            results = [{"key": entry["key"], "message": "m"} for entry in results]
        if mode == "partial":
            results = results[:1]
        outcome_ = {"refused": True, "reason": "Unknown helper action", "results": []} if mode == "refused" else \
            {"refused": False, "results": results}
        if mode != "none":
            Path(request).with_suffix(".result.json").write_text(json.dumps(outcome_))
    return readers, probe, firewall, elevate


def run_scenario(name, scenario):
    env = {**copy.deepcopy(DEFAULT_ENV), **copy.deepcopy(scenario.get("env", {}))}
    record = {"probes": [], "firewall": [], "launches": [], "events": [], "adapter_calls": 0}
    FixedClock.calls, Clock.now = 0, 1000.0
    ids = iter(range(1, 1000))
    service_module.uuid4 = lambda: type("U", (), {"hex": "%032x" % next(ids)})()
    with tempfile.TemporaryDirectory() as folder:
        readers, probe, firewall, elevate = fake_env(env, record, folder)
        projects = Projects(scenario["ieds"], record["events"])

        def update_network(_, value):
            record["events"].append(["network", value])
            projects.project["network"].update(value)
        service_module.update_network = update_network

        def adapters():
            record["adapter_calls"] += 1
            return readers("adapters")()
        network = service_module.NetworkService(projects, folder, PROGRAM, adapters=adapters, rules=readers("rules"),
                                                elevate=elevate, probe=probe, firewall=firewall)
        gateway = GatewayFirewall(network, GATEWAY, rules=readers("gateway_rules"), blocking=readers("blocking"))
        last, results = {}, []
        for step in scenario["steps"]:
            step = resolve(step, env, last) if step[0] != "set" else step
            kind = step[0]
            if kind == "set":
                env[step[1]] = resolve(step[2], env, last)
                results.append(None)
                continue
            if kind == "tick":
                Clock.now += step[1]
                results.append(None)
                continue
            action = {
                "preview": lambda: network.preview(step[1], step[2], step[3]),
                "apply": lambda: network.apply(step[1], step[2]),
                "adapters": lambda: {"adapters": network.adapters(refresh=step[1])},
                "cleanup_preview": network.cleanup_preview,
                "cleanup": lambda: network.cleanup(step[1], step[2]),
                "gw_preview": gateway.preview,
                "gw_apply": lambda: gateway.apply(step[1], step[2]),
            }[kind]
            result = outcome(action, flask_body)
            if "ok" in result and kind in ("preview", "cleanup_preview", "gw_preview"):
                body = json.loads(result["ok"])
                tag = {"preview": "$fp", "cleanup_preview": "$clean", "gw_preview": "$gw"}[kind]
                last[tag], last[tag + "_items"] = body["fingerprint"], body["items"]
            results.append(result)
        path = Path(folder) / "network" / "journal.jsonl"
        requests = sorted(p.name for p in (Path(folder) / "network" / "requests").glob("*")) \
            if (Path(folder) / "network" / "requests").exists() else []
        return {"name": name, "ieds": scenario["ieds"], "env": {**DEFAULT_ENV, **scenario.get("env", {})},
                "steps": scenario["steps"], "results": results, "journal": b64(path.read_bytes()) if path.exists() else "",
                "requests": requests, **record}


def main():
    journal.datetime = FixedClock
    sys.frozen, sys.executable = True, EXE
    service_module.time = types.SimpleNamespace(monotonic=Clock.monotonic)
    scenarios = [run_scenario(name, scenario) for name, scenario in SCENARIOS.items()]
    write("service.json", {"times": [t.isoformat() for t in TIMES], "exe": EXE, "program": PROGRAM,
                           "gateway": GATEWAY, "gateway_rule": GWRULE, "scenarios": scenarios,
                           "check": hashlib.sha256(b"").hexdigest()})


if __name__ == "__main__":
    main()
