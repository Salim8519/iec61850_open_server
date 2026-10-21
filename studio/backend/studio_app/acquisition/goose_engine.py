"""Own the GOOSE engine process (studio_goose.exe) for one Run (ADR-0073).

Simulated IEDs publish to the engine; it forwards to studio_live and, only when the
user shares GOOSE and the smart guard allows it, to a network adapter.
"""
import threading
from studio_app.runtime.worker_process import WorkerProcess

REQUIRED_CAPABILITIES = {"goose_engine_v1"}
MISSING = "The GOOSE engine (studio_goose.exe) is not built yet; run studio/tools/build_goose_engine.py"
STALE = "The GOOSE engine is outdated. Rebuild it with studio/tools/build_goose_engine.py before Run"
CLOSED_LINK = {"adapter_guid": "", "uplink": "closed", "share": "off", "message": ""}


def publishers(ieds):
    """Simulated IEDs whose GOOSE publishing is on, with the gocbRefs they own."""
    return [{"id": ied["id"], "gocb_refs": [control["gocb_ref"] for control in ied["goose"]["publishes"]]}
            for ied in ieds if (ied.get("goose") or {}).get("publishes") and ied.get("goose_publish", True)
            and ied.get("role", "simulated") == "simulated"]


class GooseEngine:
    def __init__(self, manager, starter=WorkerProcess):
        self.manager = manager
        self.starter = starter
        # `calls` serialises requests to the process; `lock` guards the state that events update.
        # Never hold `lock` while waiting for a reply: the reader thread needs it to deliver
        # events that arrive before that reply (Run froze on "uplink request timed out").
        self.calls = threading.RLock()
        self.lock = threading.Lock()
        self.process = None
        self.port = None
        self.sources = {}
        self.link = dict(CLOSED_LINK)
        self.conflicts = {}

    @property
    def executable(self):
        return self.manager.executable.with_name("studio_goose.exe")

    @property
    def running(self):
        return self.process is not None

    def ensure(self):
        """Start the engine once per Run and refuse a missing or outdated binary (GOOSE-27)."""
        with self.calls:
            if self.process is not None:
                return
            if not self.executable.exists():
                raise ValueError(MISSING)
            process = self.starter(self.executable, self._event)
            hello = process.request("hello")
            if not REQUIRED_CAPABILITIES <= set(hello.get("capabilities", [])):
                process.close()
                raise ValueError(STALE)
            self.process = process
            self.port = process.request("open", ieds=[])["port"]

    def bus_for(self, ieds):
        """Register publishing IEDs and return where each one sends GOOSE: {id: "udp:PORT:MAC"}."""
        entries = publishers(ieds)
        if not entries:
            return {}
        with self.calls:
            self.ensure()
            opened = self.process.request("open", ieds=entries)
            with self.lock:
                self.sources.update(opened["sources"])
            return {ied_id: f"udp:{opened['port']}:{mac.replace('-', '')}" for ied_id, mac in opened["sources"].items()}

    def deliver(self, inbox_port):
        with self.calls:
            self.ensure()
            self.process.request("deliver", port=inbox_port)

    def uplink(self, adapter_guid, share):
        """Listen on the adapter ('' = none) and share outside only when `share` is true."""
        with self.calls:
            if self.process is None:
                with self.lock:
                    return dict(self.link)
            result = self.process.request("uplink", adapter_guid=adapter_guid, share=bool(share))
            with self.lock:
                self.link.update(result)
                return dict(self.link)

    def gate(self, ied_id, choice):
        with self.calls:
            if self.process is None:
                raise ValueError("Press Run first: GOOSE is not running")
            result = self.process.request("gate", ied_id=ied_id, choice=choice)
            with self.lock:
                if ied_id in self.conflicts:
                    self.conflicts[ied_id]["choice"] = choice
            return result

    def stats(self):
        with self.calls:
            return self.process.request("stats") if self.process is not None else {}

    def _event(self, event):
        kind = event.get("event")
        emit = self.manager.coordinator.emit
        if kind == "goose_conflict":
            with self.lock:
                previous = self.conflicts.get(event["ied_id"], {})
                self.conflicts[event["ied_id"]] = {**{key: event.get(key) for key in
                                                      ("ied_id", "gocb_ref", "src_mac", "sharing")},
                                                   "choice": previous.get("choice")}
            emit("goose_conflict", **{key: event.get(key) for key in ("ied_id", "gocb_ref", "src_mac", "sharing")})
        elif kind == "goose_share":
            with self.lock:
                self.link["share"] = event.get("state", "off")
            emit("goose_share", state=self.link["share"])
        elif kind == "goose_uplink":
            with self.lock:
                self.link.update(uplink=event.get("state", ""), message=event.get("message", ""))
            emit("goose_uplink", state=event.get("state"), message=event.get("message", ""))
        elif kind == "goose_storm":
            emit("goose_storm", ied_id=event.get("ied_id"), limit_per_second=event.get("limit_per_second"))
        elif kind == "worker_failed":
            with self.lock:
                self.link = {**CLOSED_LINK, "uplink": "failed", "message": "The GOOSE engine stopped unexpectedly"}
            emit("goose_listener_failed", message=event.get("error") or "The GOOSE engine stopped")

    def snapshot(self):
        with self.lock:
            return {"running": self.process is not None, **self.link,
                    "conflicts": [dict(item) for item in self.conflicts.values()]}

    def stop(self):
        with self.calls:
            process, self.process = self.process, None
            with self.lock:
                self.port, self.sources, self.conflicts = None, {}, {}
                self.link = dict(CLOSED_LINK)
        if process is not None:
            process.close()
