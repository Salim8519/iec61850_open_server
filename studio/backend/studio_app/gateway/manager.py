"""Run the gateway simulator worker and keep its points in step with the IED values."""
import threading
from studio_app.domain.gateway_points import propose
from studio_app.domain.gateway_rules import empty_gateway, listen_address, validate_points, validate_settings
from studio_app.domain.gateway_values import display, sample
from studio_app.runtime.worker_process import WorkerProcess
from .commands import CommandExecutor
from .traffic_log import COMMAND_LINES, TRAFFIC_LINES, NumberedLog

SYNC_SECONDS = 0.1
REQUIRED = {"gateway_iec104", "gateway_modbus", "gateway_dnp3", "commands_via_backend"}


class GatewayManager:
    def __init__(self, coordinator, projects, native_executable, control=None):
        self.coordinator = coordinator
        self.control = control
        self.projects = projects
        self.native_executable = native_executable
        self.executable = native_executable.with_name("studio_gateway.exe")
        self.lock = threading.RLock()
        self.process = None
        self.points = {}
        self.sent = {}
        self.state, self.message, self.opened, self.problems = "stopped", "", None, []
        self.clients = {}
        self.traffic = NumberedLog(TRAFFIC_LINES)
        self.commands = NumberedLog(COMMAND_LINES)
        self.executor = CommandExecutor(self)
        self.stopping = threading.Event()
        self.syncer = None
        self.converter = None  # ConverterSource, set by the app (protocol converter role)

    def emit(self, kind, **payload):
        self.coordinator.emit(kind, **payload)

    def config(self):
        return self.projects.project.get("gateway") or empty_gateway()

    def ied(self, ied_id):
        return next((ied for ied in self.projects.project["ieds"] if ied["id"] == ied_id), None)

    def point(self, point_id):
        """A point of the running gateway, else of the saved list (manual commands work while stopped)."""
        if point_id in self.points:
            return self.points[point_id]
        config = self.config()
        return next((point for point in config["points"].get(config["protocol"], []) if point.get("id") == point_id), None)

    def send(self, point_id, value):
        """A command typed on the page, run like a SCADA command and logged the same way."""
        ok, reason = self.executor.run({"point_id": point_id, "value": bool(value), "protocol": "manual",
                                        "peer": "this page"})
        return {"ok": ok, "reason": reason}

    def _prepare(self):
        """Propose points when the list is empty. Run stays the user's choice (owner, 2026-10-17)."""
        config = self.config()
        protocol = config["protocol"]
        if not config["points"].get(protocol):
            proposal, _ = propose(self.projects.project["ieds"], protocol)
            clean, _ = validate_points(protocol, proposal, self.projects.project["ieds"])
            self.projects.gateway({**config, "points": dict(config["points"], **{protocol: clean})})
        return self.config()

    def autostart(self):
        """Start again after the app restarts if the gateway was running when it closed; never raises."""
        if self.config().get("auto_start") and self.process is None:
            try:
                self.start()
            except Exception as error:
                self.state, self.message = "failed", f"Automatic start failed: {error}"

    def start(self):
        with self.lock:
            if self.process is not None:
                raise ValueError("The gateway is already running; stop it first")
            if not self.executable.exists():
                raise RuntimeError("The gateway program (studio_gateway.exe) is not built yet; rebuild the native programs")
            role = self.config().get("role", "gateway")
            if role == "master":
                raise ValueError("The tool is set to SCADA master; switch it to Station gateway first")
            config = self._prepare() if role == "gateway" else self.config()
            protocol = config["protocol"]
            if role == "converter":
                points, problems = self.converter.points(protocol)
            else:
                points, problems = validate_points(protocol, config["points"].get(protocol, []), self.projects.project["ieds"])
            if not points:
                raise ValueError("The point list is empty. Press 'Propose from IEDs' or import a CSV point list first.")
            settings = dict(validate_settings(protocol, config["settings"].get(protocol)), bind=listen_address(config["listen"]))
            native = [{"id": point["id"], "kind": point["kind"], "address": point["address"], "scale": point["scale"],
                       "monitor_only": self._monitor_only(point, config)} for point in points]
            process = WorkerProcess(self.executable, self._event)
            try:
                if not REQUIRED <= set(process.request("hello").get("capabilities", [])):
                    raise RuntimeError("The gateway program is outdated; rebuild the native programs")
                self.opened = process.request("start", timeout=15, protocol=protocol, settings=settings, points=native)
            except Exception:
                process.close()
                raise
            self.process, self.points, self.sent, self.problems = process, {p["id"]: p for p in points}, {}, problems
            self.clients = {}
            self.state, self.message = "running", f"Listening on {self.opened['bind']}:{self.opened['port']}"
            self.traffic.add(dir="info", peer="", text=f"Gateway started: {protocol} on {self.opened['bind']}:"
                                                      f"{self.opened['port']} with {len(points)} points")
            self.stopping = threading.Event()
            self.syncer = threading.Thread(target=self._sync, args=(self.stopping,), daemon=True, name="gateway-sync")
            self.syncer.start()
        if not self.config().get("auto_start"):
            self.projects.gateway({**self.config(), "auto_start": True})
        self.emit("gateway_started", protocol=protocol, port=self.opened["port"], points=len(points))
        return self.status()

    def _event(self, event):
        kind = event.get("event")
        if kind == "gw_traffic":
            text = event["text"] + (f" ({event['skipped_before']} lines skipped)" if event.get("skipped_before") else "")
            self.traffic.add(dir=event.get("dir", ""), peer=event.get("peer", ""), text=text)
        elif kind == "gw_client":
            if event["state"] == "open":
                self.clients[event["peer"]] = "connected"
            else:
                self.clients.pop(event["peer"], None)
        elif kind == "gw_command":
            self.executor.submit(self.process, event)
        elif kind == "worker_failed":
            self.state, self.message = "failed", "The gateway program stopped unexpectedly. Press Start again."
            self.stopping.set()

    def _sync(self, stopping):
        while not stopping.wait(SYNC_SECONDS):
            try:
                self.push()
            except Exception as error:
                self.message = f"Values could not be sent: {error}"

    def push(self):
        process = self.process
        if process is None:
            return 0
        index, run_state = self.coordinator.index, self.coordinator.state
        changed = []
        for point in self.points.values():
            if point["kind"].startswith("command_"):
                continue
            current = (self.converter.sample(point) if point.get("source") == "remote"
                       else sample(point, index, self.ied(point["ied"]), run_state))
            if self.sent.get(point["id"]) != current:
                self.sent[point["id"]] = current
                changed.append(current)
        for start in range(0, len(changed), 500):
            process.request("update", values=changed[start:start + 500])
        return len(changed)

    def _monitor_only(self, point, config):
        if point.get("source") == "remote":
            return not config.get("real_commands", True)
        role = (self.ied(point["ied"]) or {}).get("role", "off")
        return role == "off" or (role == "real" and not config.get("real_commands", True))

    def stop(self, remember=True):
        """remember=False keeps auto_start, for a stop caused by the app closing."""
        if remember and self.config().get("auto_start"):
            self.projects.gateway({**self.config(), "auto_start": False})
        with self.lock:
            self.stopping.set()
            process, self.process = self.process, None
            if self.syncer is not None:
                self.syncer.join(timeout=5)
            if process is not None:
                try:
                    process.request("halt", timeout=10)
                finally:
                    process.close()
                self.traffic.add(dir="info", peer="", text="Gateway stopped")
                self.emit("gateway_stopped")
            self.state, self.message, self.clients = "stopped", "", {}
        return self.status()

    def status(self):
        config = self.config()
        return {"state": self.state, "message": self.message, "opened": self.opened if self.process else None,
                "protocol": config["protocol"], "listen": config["listen"], "clients": sorted(self.clients),
                "problems": self.problems, "run_state": self.coordinator.state,
                "available": self.executable.exists(), "traffic_last": self.traffic.sequence,
                "real_commands": config.get("real_commands", True), "auto_start": config.get("auto_start", False),
                "role": config.get("role", "gateway")}

    def table(self, protocol):
        """The saved point list of one protocol with what each point carries now."""
        config = self.config()
        if config.get("role") == "converter":
            return self.converter.table(protocol)
        index, run_state = self.coordinator.index, self.coordinator.state
        names = {ied["id"]: ied["name"] for ied in self.projects.project["ieds"]}
        rows = []
        for point in config["points"].get(protocol, []):
            ied = self.ied(point["ied"])
            row = {**point, "ied_name": names.get(point["ied"], "?"), "ied_role": (ied or {}).get("role", "off"),
                   "monitor_only": self._monitor_only(point, config)}
            if not point["kind"].startswith("command_"):
                current = sample(point, index, ied, run_state)
                row.update(value_text=display(point, current), online=current["source_ok"])
            rows.append(row)
        return rows
