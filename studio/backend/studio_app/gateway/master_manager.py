"""ScadaBench as SCADA master: run studio_master toward a device (gateway, RTU or outstation).

At connect it asks for everything (IEC 104 general interrogation, DNP3 integrity poll) and
repeats that on the user's interval, so every point is discovered without typing addresses.
Modbus has no discovery: the user's ranges are read every poll interval, and Find registers
(modbus_probe) proposes ranges. A link the device
drops is retried every RETRY_SECONDS until the user presses Disconnect.
"""
import threading
import time
from studio_app.domain.master_points import merge, rows
from studio_app.domain.master_rules import empty_master, validate_settings
from studio_app.runtime.worker_process import WorkerProcess
from .modbus_probe import ModbusProbe
from .traffic_log import COMMAND_LINES, TRAFFIC_LINES, NumberedLog

RETRY_SECONDS = 10
CAPABILITY = {"iec104": "master_iec104", "dnp3": "master_dnp3", "modbus": "master_modbus"}
NAMES = {"iec104": "IEC 104", "dnp3": "DNP3", "modbus": "Modbus TCP"}


# What "asking for every value" is called in each protocol (change 0089: Modbus said "Interrogation").
ASK_WORDS = {"iec104": "General interrogation", "dnp3": "Integrity poll", "modbus": "Reading the ranges"}


class MasterManager:
    def __init__(self, projects, native_executable, emit=lambda kind, **payload: None):
        self.projects = projects
        self.emit = emit
        self.executable = native_executable.with_name("studio_master.exe")
        self.lock = threading.RLock()  # connect/disconnect
        self.table_lock = threading.Lock()  # never held around a worker request: values arrive meanwhile
        self.process = None
        self.table = {}
        self.device = None  # (protocol, host, port) the table belongs to
        self.state, self.message, self.link_open = "disconnected", "", False
        self.last_interrogation = None
        self.poll_failures = []
        self.probe = ModbusProbe()
        self.traffic = NumberedLog(TRAFFIC_LINES)
        self.commands = NumberedLog(COMMAND_LINES)
        self.stopping = threading.Event()
        self.keeper = None

    def config(self):
        return (self.projects.project.get("gateway") or {}).get("master") or empty_master()

    def save(self, master):
        self.projects.gateway({**self.projects.project["gateway"], "master": master})

    def connect(self):
        with self.lock:
            if self.process is not None:
                raise ValueError("Already connected; press Disconnect first")
            if self.projects.project["gateway"].get("role") not in ("master", "converter"):
                raise ValueError("Switch the tool to SCADA master first")
            if not self.executable.exists():
                raise RuntimeError("The master program (studio_master.exe) is not built yet; rebuild the native programs")
            config = self.config()
            protocol, host = config["protocol"], config["host"]
            if not host:
                raise ValueError("Type the address of the gateway, RTU or outstation first")
            settings = validate_settings(protocol, config["settings"][protocol])
            process = WorkerProcess(self.executable, self._event)
            try:
                if CAPABILITY[protocol] not in process.request("hello").get("capabilities", []):
                    raise RuntimeError(f"ScadaBench cannot act as {NAMES[protocol]} master in this build yet")
                process.request("connect", timeout=settings.get("t0", 10) + 10, protocol=protocol,
                                settings={**settings, "host": host})
            except Exception:
                process.close()
                raise
            device = (protocol, host, settings["port"])
            if device != self.device:
                with self.table_lock:
                    self.table, self.device = {}, device
            self.process, self.link_open = process, True
            self.state, self.message = "connected", f"Connected to {host}:{settings['port']}"
            self.traffic.add(dir="info", peer="", text=f"{NAMES[protocol]} master connected to {host}:{settings['port']}")
            self.stopping = threading.Event()
            self.keeper = threading.Thread(target=self._keep, args=(self.stopping,), daemon=True, name="master-keeper")
            self.keeper.start()
        if not config.get("auto_connect"):
            self.save({**config, "auto_connect": True})
        self.emit("master_connected", protocol=protocol, host=host, port=settings["port"])
        self._ask_all()
        return self.status()

    def autoconnect(self):
        """Connect again after the app restarts if it was connected when it closed; never raises."""
        if self.config().get("auto_connect") and self.process is None:
            try:
                self.connect()
            except Exception as error:
                self.state, self.message = "failed", f"Automatic connect failed: {error}"

    def _ask_all(self):
        """GI, integrity poll or Modbus range reads. Errors are shown, never raised from background use."""
        try:
            return self.interrogate()
        except Exception as error:
            self.message = f"{ASK_WORDS[self.config()['protocol']]} failed: {error}"
            return None

    def interrogate(self):
        process = self._process()
        if self.config()["protocol"] == "modbus":
            return self.poll(process)
        # GI confirmation (t1) + its end + the counter interrogation (t1) can take about 50 s.
        answer = process.request("interrogate", timeout=60)
        self.last_interrogation = time.time()
        self.traffic.add(dir="info", peer="", text=f"Asked for all values: {answer.get('text', '')}")
        return answer

    def poll(self, process):
        ranges = self.config()["modbus_ranges"]
        if not ranges:
            self.message = "Add the Modbus ranges to read, or use Find registers (Modbus has no discovery)"
            return {"confirmed": False, "text": self.message}
        answer = process.request("poll", timeout=30, ranges=ranges)
        self.poll_failures, self.last_interrogation = answer["failed"], time.time()
        self.message = "" if not answer["failed"] else f"{len(answer['failed'])} of {len(ranges)} ranges refused"
        return {"confirmed": True, "text": f"read {answer['values']} values"}

    def _keep(self, stopping):
        """Repeat the interrogation or Modbus reads on the user's interval and retry a dropped link."""
        while True:
            config = self.config()
            protocol, settings = config["protocol"], config["settings"][config["protocol"]]
            if stopping.wait(settings["poll_ms"] / 1000 if protocol == "modbus" else 1):
                return
            minutes = settings.get("gi_minutes", settings.get("integrity_minutes", 0))
            if not self.link_open:
                if self.state == "lost" and not stopping.wait(RETRY_SECONDS):
                    self._reconnect(config)
            elif protocol == "modbus":
                if not self.probe.running:
                    self._ask_all()
            elif minutes and time.time() - (self.last_interrogation or 0) >= minutes * 60:
                self._ask_all()

    def find_registers(self, table, start, end):
        process = self._process()
        self.traffic.add(dir="info", peer="", text=f"Find registers: {table} {start} to {end}")
        return self.probe.start(process, table, start, end, lambda result: self.traffic.add(
            dir="info", peer="", text=f"Find registers {result['state']}: {len(result.get('found', []))} ranges answer"))

    def _reconnect(self, config):
        process = self.process
        if process is None:
            return
        protocol = config["protocol"]
        settings = validate_settings(protocol, config["settings"][protocol])
        try:
            process.request("disconnect", timeout=10)
            process.request("connect", timeout=settings.get("t0", 10) + 10, protocol=protocol,
                            settings={**settings, "host": config["host"]})
            self.link_open, self.state, self.message = True, "connected", "Connected again"
            self._ask_all()
        except Exception as error:
            self.message = f"Still trying to reach the device: {error}"

    def _event(self, event):
        kind = event.get("event")
        if kind == "master_values":
            with self.table_lock:
                merge(self.table, event["values"])
        elif kind == "master_link":
            if event["state"] == "open":
                self.link_open = True
            elif event["state"] == "closed" and self.process is not None and event.get("reason") != "Disconnected by you":
                self.link_open, self.state = False, "lost"
                self.message = f"{event.get('reason', 'The link closed')}. Trying again every {RETRY_SECONDS} s."
        elif kind == "master_probe":
            self.probe.progress(event)
        elif kind == "master_interrogation":
            finished = event.get("state") == "finished"
            self.traffic.add(dir="info", peer="", text="All values received" if finished
                             else f"Asking for all values {event.get('state', 'failed')}")
        elif kind == "gw_traffic":
            text = event["text"] + (f" ({event['skipped_before']} lines skipped)" if event.get("skipped_before") else "")
            self.traffic.add(dir=event.get("dir", ""), peer=event.get("peer", ""), text=text)
        elif kind == "worker_failed":
            self.link_open, self.state = False, "failed"
            self.message = "The master program stopped unexpectedly. Press Connect again."
            self.stopping.set()

    def _process(self):
        process = self.process
        if process is None or not self.link_open:
            raise ValueError("Not connected to the device; press Connect first")
        return process

    def command(self, request, who="You · this page"):
        """A command the user confirmed on the page (or SCADA sent through the converter); logged
        as requested, then as answered."""
        process = self._process()
        entry = self.commands.add(who=who, kind=request["kind"], address=request.get("address"),
                                  value=request.get("value"), select=request.get("select", False), state="sent",
                                  text="Sent, waiting for the device")
        try:
            answer = process.request("command", timeout=40, **request)
        except RuntimeError as error:
            self.commands.update(entry["seq"], state="refused", text=str(error))
            return {"ok": False, "steps": [], "text": str(error)}
        text = "; ".join(f"{step['step']}: {step['text']}" for step in answer.get("steps", []))
        self.commands.update(entry["seq"], state="confirmed" if answer["ok"] else "refused", text=text)
        return {**answer, "text": text}

    def disconnect(self, remember=True):
        """remember=False keeps auto_connect, for a disconnect caused by the app closing."""
        if remember and self.config().get("auto_connect"):
            self.save({**self.config(), "auto_connect": False})
        with self.lock:
            self.stopping.set()
            process, self.process = self.process, None
            if process is not None:
                try:
                    process.request("stop", timeout=10)
                except Exception:
                    pass
                finally:
                    process.close()
                self.traffic.add(dir="info", peer="", text="Disconnected")
                self.emit("master_disconnected")
            self.state, self.message, self.link_open = "disconnected", "", False
        return self.status()

    def status(self):
        config = self.config()
        protocol = config["protocol"]
        return {"state": self.state, "message": self.message, "link_open": self.link_open, "protocol": protocol,
                "host": config["host"], "port": config["settings"][protocol]["port"],
                "points": len(self._table_for(config)), "last_interrogation": self.last_interrogation,
                "available": self.executable.exists(), "traffic_last": self.traffic.sequence,
                "auto_connect": config.get("auto_connect", False), "probe": self.probe.snapshot(),
                "poll_failures": self.poll_failures, "ranges": len(config["modbus_ranges"])}

    def _table_for(self, config):
        """The discovered points belong to one device; after the settings point at another
        protocol, address or port they are not shown (change 0089: Modbus rows under IEC 104)."""
        protocol = config["protocol"]
        if self.device and self.device != (protocol, config["host"], config["settings"][protocol]["port"]):
            return {}
        return self.table

    def points(self):
        config = self.config()
        protocol = config["protocol"]
        with self.table_lock:
            return rows(self._table_for(config), protocol, config["unticked"][protocol], config["labels"][protocol],
                        self.link_open)
