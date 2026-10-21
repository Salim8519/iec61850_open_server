"""Deterministic stand-ins for the route golden scenario: clock, native workers, network.

The fake workers answer from canned rules and log every (process, op, args, reply), so the
Go test can replay exactly the same native conversation.
"""
import json
import threading
from datetime import datetime, timezone
from pathlib import Path

FIXED = datetime(2026, 10, 20, 8, 0, 0, tzinfo=timezone.utc)
GOCB_VALUES = {"GoEna": True, "GoID": "other-id", "DatSet": "FEED1LD1/LLN0$Other", "ConfRev": 1}
ADAPTERS = [
    {"guid": "{11111111-2222-3333-4444-555555555555}", "name": "Ethernet", "description": "Intel Ethernet",
     "status": "Up", "index": 4, "mac": "00-11-22-33-44-55", "speed": "1 Gbps", "kind": "ethernet",
     "default_route": True, "addresses": [{"ip": "192.168.1.20", "prefix": 24, "skip_as_source": False, "origin": "Dhcp"}]},
    {"guid": "{66666666-7777-8888-9999-000000000000}", "name": "Wi-Fi", "description": "Wireless",
     "status": "Up", "index": 7, "mac": "AA-BB-CC-DD-EE-FF", "speed": "300 Mbps", "kind": "wifi",
     "default_route": False, "addresses": []},
]


class FixedDatetime(datetime):
    @classmethod
    def now(cls, tz=None):
        return FIXED if tz is not None else FIXED.replace(tzinfo=None)


class Monotonic:
    value = 1000.0

    @staticmethod
    def monotonic():
        return Monotonic.value


def canonical(values):
    return json.dumps(values, sort_keys=True, separators=(",", ":"))


class Script:
    """Shared reply rules and the log of every request."""

    def __init__(self, signals_by_ref, discovery):
        self.signals = signals_by_ref
        self.discovery = discovery
        self.log = []
        self.lock = threading.Lock()
        self.reads = 0

    def reply(self, role, op, values):
        if role == "goose":
            return {"hello": {"capabilities": ["goose_engine_v1"]},
                    "open": {"port": 47001, "sources": {}},
                    "uplink": {"adapter_guid": values.get("adapter_guid", ""), "uplink": "open"},
                    "stats": {"delivered_wire": 0}, "gate": {"ok": True, "choice": values.get("choice")},
                    }.get(op, {})
        if op == "hello":
            return {"capabilities": ["live_discover", "live_reports", "live_poll", "read_only_client"]}
        if op == "discover":
            return self.discovery
        if op == "rcb_status":
            return {"blocks": [{"rcb": rcb, "readable": True, "free": index == 0,
                                "dataset": "FEED1LD1/LLN0$Wrong" if index else "", "conf_rev": 7}
                               for index, rcb in enumerate(values["rcbs"])]}
        if op == "rcb_reserve":
            return {"reserved": True}
        if op == "goose_start":
            return {"inbox_port": 47002}
        if op == "read":
            with self.lock:
                self.reads += 1
            return {"values": [self.read_one(item["ref"], index) for index, item in enumerate(values["refs"])]}
        return {}

    def read_one(self, ref, index):
        attribute = ref.rsplit(".", 1)[-1]
        if attribute in GOCB_VALUES and ("/LLN0.Feed1SwiPosCtl." in ref or "/LLN0.Bus1TripEventCb." in ref):
            return {"ref": ref, "value": GOCB_VALUES[attribute]}
        if ref.endswith((".Mod.stVal", ".Beh.stVal")):
            return {"ref": ref, "value": 3 if "LD1/LLN0.Beh" in ref else 1}
        if ref.endswith(".Sim.stVal"):
            return {"ref": ref, "value": False}
        if index == 4:
            return {"ref": ref, "error": 10}
        kind = (self.signals.get(ref) or {}).get("kind", "")
        if kind == "BOOLEAN":
            return {"ref": ref, "value": True}
        if kind.startswith("FLOAT"):
            return {"ref": ref, "value": 1.25}
        if kind == "Quality":
            return {"ref": ref, "value": {"bits": 13, "le": 2048, "be": 1}}
        if kind.startswith("INT") or kind in {"Enum", "Timestamp", "EntryTime", "Dbpos"}:
            return {"ref": ref, "value": 2}
        return {"ref": ref, "value": "text"}


def worker_class(script):
    class FakeWorker:
        def __init__(self, executable, on_event=None):
            self.role = "goose" if Path(executable).name == "studio_goose.exe" else "live"
            self.on_event = on_event

        def request(self, op, timeout=10, **values):
            reply = script.reply(self.role, op, values)
            with script.lock:
                script.log.append([self.role, op, canonical(values), reply])
            return reply

        def close(self):
            pass
    return FakeWorker


def discovery_for(ied, drop, retype, extra):
    """A device model from the IED's signals: one point missing, one retyped, one extra."""
    from studio_app.domain.live_value import EXPECTED_MMS
    devices = {}
    for signal in ied["signals"]:
        if signal["ref"] == drop:
            continue
        domain, rest = signal["ref"].split("/", 1)
        node = rest.split(".", 1)[0]
        nodes = devices.setdefault(domain, {})
        entry = nodes.setdefault(node, {"name": node, "leaves": [], "dataset_members": {},
                                        "buffered_reports": [], "unbuffered_reports": []})
        mms = sorted(EXPECTED_MMS.get(signal["kind"], {8}))[0]
        entry["leaves"].append({"ref": signal["ref"], "fc": signal["fc"], "mms_type": 6 if signal["ref"] == retype else mms})
    first = next(iter(devices.values()))
    next(iter(first.values()))["leaves"].append({"ref": extra, "fc": "ST", "mms_type": 2})
    for dataset in ied.get("datasets", []):
        owner, name = dataset["ref"].split("$", 1)
        domain, node = owner.split("/", 1)
        if node in devices.get(domain, {}):
            devices[domain][node]["dataset_members"][name] = {"members": [ref + "[ST]" for ref in dataset["fcda_refs"][1:]]}
    for report in ied.get("reports", [])[:1]:
        domain, rest = report["ref"].split("/", 1)
        node, kind, name = rest.split(".")
        devices[domain][node]["buffered_reports" if kind == "BR" else "unbuffered_reports"].append(name)
    return {"devices": [{"name": domain, "nodes": list(nodes.values())} for domain, nodes in devices.items()]}
