"""Start and stop real-IED sessions and the GOOSE monitor together with Run/Stop."""
from pathlib import Path
from studio_app.domain.equipment_state import binding_keys
from studio_app.networking.npcap import npcap_status
from .goose_monitor import GooseMonitor
from .live_values import LiveValues
from .real_session import RealSession
from .sv_monitor import SvMonitor


def watched_points(project, ied):
    """Signals of a real IED that the canvas or test panel shows, plus quality/time."""
    keys = {key for node in project["diagram"]["nodes"] for key in binding_keys(node["data"].get("state_mapping"))}
    for node in project["diagram"]["nodes"]:
        for entry in node["data"].get("relay_statuses") or []:
            keys.update(binding_keys({"mode": "direct", "source": entry["source"]}))
    keys.update((entry["ied_id"], entry["ref"]) for entry in project.get("test_panel", []))
    by_ref = {signal["ref"]: signal for signal in ied["signals"]}
    points = {}
    for ied_id, reference in keys:
        signal = by_ref.get(reference) if ied_id == ied["id"] else None
        if not signal:
            continue
        for ref in (reference, signal.get("quality_ref"), signal.get("timestamp_ref")):
            if ref and ref in by_ref:
                points[ref] = {"ref": ref, "fc": by_ref[ref]["fc"]}
    return list(points.values())[:500]


class LiveManager:
    def __init__(self, coordinator, executable):
        self.coordinator = coordinator
        self.executable = Path(executable)
        self.values = LiveValues(coordinator)
        self.sessions = {}
        self.goose = GooseMonitor(self)
        self.sv = SvMonitor(self)

    def start(self, project, selected=None):
        """Start the listeners and connect the real IEDs in `selected` (all when None)."""
        ieds = project["ieds"]
        real = [ied for ied in ieds if ied.get("role") == "real" and (selected is None or ied["id"] in selected)]
        needs_live = real or any((ied.get("goose") or {}).get("publishes") or (ied.get("goose") or {}).get("subscribes")
                                 or (ied.get("sv") or {}).get("publishes") for ied in ieds)
        if needs_live and not self.executable.exists():
            raise ValueError("The live client (studio_live.exe) is not built yet; rebuild the native programs")
        self.connect(project, real)
        try:
            self.goose.start(ieds, project["network"], npcap_status()["installed"])
        except Exception as error:
            self.goose.status.update(state="failed", message=f"GOOSE listener could not start: {error}")
            self.coordinator.emit("goose_listener_failed", message=str(error))
        try:
            self.sv.start(project, npcap_status()["installed"])
        except Exception as error:
            self.sv.status.update(state="failed", message=f"Sampled Values listener could not start: {error}")
            self.coordinator.emit("sv_listener_failed", message=str(error))

    def connect(self, project, real):
        """Connect more real IEDs while the run continues (client read only, ADR-0067)."""
        if real and not self.executable.exists():
            raise ValueError("The live client (studio_live.exe) is not built yet; rebuild the native programs")
        for ied in real:
            session = RealSession(self, ied, watched_points(project, ied))
            self.sessions[ied["id"]] = session
            session.start()

    def disconnect(self, ied_id):
        session = self.sessions.pop(ied_id, None)
        if session:
            session.stop()

    def stop(self):
        self.goose.stop()
        self.sv.stop()
        sessions, self.sessions = self.sessions, {}
        for session in sessions.values():
            session.stop()

    def session(self, ied_id):
        session = self.sessions.get(ied_id)
        if session is None:
            raise ValueError("This IED is not connected as a real device. Set it to 'real' and press Run.")
        return session

    def status(self):
        return {"sessions": [dict(session.status, comparison=None, health=None) for session in self.sessions.values()],
                "goose": self.goose.snapshot()}
