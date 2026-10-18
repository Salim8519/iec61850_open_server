"""Own simulation lifecycle, source selection, and acknowledged runtime state."""
import threading
import time
from pathlib import Path
from studio_app.domain.signal import validate_value
from studio_app.domain.source import validate_source, source_summary
from studio_app.runtime.supervisor import Supervisor
from studio_app.observability.log_writer import RunLog
from .generators import value_at
from .scenario_runner import ScenarioRunner
from .sample_writer import SampleWriter
from .automation import Automation
from .bindings import publish_bindings
from .initial_state import apply_initial_states
from .test_mode_apply import apply_test_modes
from .energization_states import EnergizationStates
from .equipment_states import EquipmentStates
from .equipment_control import EquipmentControl
from .native_control_feedback import NativeControlFeedback
from .mms_access import MmsAccess
from .cadence import Cadence
from .pulses import Pulses
from .control_compatibility import profiles as compatibility_profiles
from studio_app.acquisition.manager import LiveManager
from studio_app.projects.ied_settings import endpoint_conflicts
from .run_selection import add_ieds, remove_ieds, runnable, simulated_of


class Coordinator:
    def __init__(self, projects, events, executable, directory):
        self.projects = projects
        self.events = events
        self.directory = directory
        self.state = "stopped"
        self.lock = threading.RLock()
        self.cycle_lock = threading.RLock()
        self.index = {}
        self.ied_indexes = {}
        self.sources = {}
        self.active = set()  # IEDs in the current run (simulated and real, ADR-0067)
        self.elapsed = 0
        self.last_automation_time = time.monotonic()
        self.cadence = Cadence(self.last_automation_time)
        self.log = None
        self.supervisor = Supervisor(executable, directory / "runtime", self.on_worker_event)
        self.scenario_runner = ScenarioRunner(self.update, self.supervisor.read, self.emit)
        self.sample_writer = SampleWriter(self)
        self.automation = Automation(self)
        self.pulses = Pulses(self)
        self.equipment_states = EquipmentStates(self)
        self.energization = EnergizationStates(self)
        self.equipment_states.after_publish = self.energization.publish
        self.equipment_control = EquipmentControl(self)
        self.native_control_feedback = NativeControlFeedback(self)
        self.mms_access = MmsAccess(self)
        self.live = LiveManager(self, Path(executable).with_name("studio_live.exe"))
        self.shutdown_event = threading.Event()
        self.thread = threading.Thread(target=self._tick, daemon=True)
        self.thread.start()

    def emit(self, kind, **payload):
        event = self.events.publish(kind, **payload)
        if self.log:
            self.log.write(event)
        return event

    def on_worker_event(self, ied_id, event):
        kind = event.get("event", "native_event")
        if kind == "mms_access":  # Per-second client summary: live view only (ADR-0066).
            self.mms_access.update(ied_id, event)
            return
        if kind == "control_applied":
            event["external"] = self.mms_access.control(ied_id, event)
            self.pulses.cancel(ied_id, event.get("status_ref"), "New MMS control feedback")
            with self.lock:
                signal = self.index.get((ied_id, event.get("status_ref")))
                if signal:
                    signal.update(value=event["value"], timestamp=event["timestamp"])
                    time_ref = signal.get("timestamp_ref")
                    if time_ref:
                        self.index[(ied_id, time_ref)]["value"] = event["timestamp"]
                        event["updates"] = {time_ref: event["timestamp"]}
        if kind == "worker_failed":
            self.pulses.cancel(ied_id, reason="IED worker failed")
            self.projects.find_ied(ied_id)["state"] = "failed"
        reserved = {"kind": "native_kind", "id": "native_id", "timestamp": "source_timestamp"}
        payload = {reserved.get(key, key): value for key, value in event.items() if key != "event"}
        self.emit(kind, ied_id=ied_id, **payload)
        if kind == "control_applied":
            self.native_control_feedback.enqueue(ied_id, event)

    def start(self, ied_ids=None):
        """Run every IED not set to Off, or only `ied_ids`; adds them when a run is already on."""
        with self.cycle_lock:
            if ied_ids is not None and self.state in {"running", "paused"}:
                return add_ieds(self, set(ied_ids))
            self._start(None if ied_ids is None else set(ied_ids))

    def _start(self, selection=None):
        with self.lock:
            if self.state != "stopped":
                raise ValueError("Stop the current simulation first")
            project = self.projects.project
            ieds = runnable(project, selection)
            simulated = simulated_of(ieds)
            if not project["ieds"]:
                raise ValueError("Import or create an IED before Run")
            if not ieds:
                raise ValueError("Set at least one IED to 'Simulate' or 'Connect to real device' before Run")
            conflicts = endpoint_conflicts(ieds, project["network"])
            if conflicts:
                raise ValueError(conflicts[0])
            self.state = "starting"
            self.active = {ied["id"] for ied in ieds}
            self.index = {(ied["id"], signal["ref"]): signal for ied in ieds for signal in ied["signals"]}
            self.ied_indexes = {ied["id"]: {signal["ref"]: signal for signal in ied["signals"]} for ied in ieds}
            running = {ied["id"] for ied in simulated}
            self.sources = {(item["ied_id"], item["ref"]): dict(item["spec"])
                            for item in project["sources"] if item["ied_id"] in running}
            self.elapsed = 0
            self.sample_writer.reset(simulated)
            self.mms_access.reset()
            self.equipment_states.reset(self.projects.project["diagram"])
            self.energization.reset(self.projects.project["diagram"])
            self.native_control_feedback.reset(self.projects.project["diagram"])
        try:
            self.log = RunLog(self.directory / "logs")
            profiles = compatibility_profiles(project["diagram"])
            if simulated:
                self.supervisor.start(simulated, profiles, project["network"])
            self.scenario_runner.reset(self.projects.project["scenarios"])
            self.state = "running"
            self.last_automation_time = time.monotonic()
            self.emit("runtime", state=self.state, running=sorted(self.active))
            publish_bindings(self)
            apply_initial_states(self)
            apply_test_modes(self)
            self.live.start(project, self.active)
            self.last_automation_time = time.monotonic()
            self.cadence.reset(self.last_automation_time)
        except Exception as error:
            self.live.stop()
            self.native_control_feedback.clear()
            self.scenario_runner.stop()
            self.supervisor.stop(project["ieds"])
            self.state = "stopped"
            self.active = set()
            self.emit("runtime", state=self.state, message=str(error))
            if self.log:
                self.log.close()
                self.log = None
            raise

    def stop(self, ied_ids=None):
        """Stop the whole run, or only `ied_ids` while the other IEDs keep running."""
        with self.cycle_lock:
            if ied_ids is not None and self.state in {"running", "paused"}:
                selection = set(ied_ids)
                if self.active - selection:
                    return remove_ieds(self, selection)
            self.pulses.cancel()
            self.state = "stopping"
            self.live.stop()
            self.native_control_feedback.clear()
            self.scenario_runner.stop()
            self.supervisor.stop(self.projects.project["ieds"])
            self.state = "stopped"
            self.active = set()
            self.emit("runtime", state=self.state)
            if self.log:
                self.log.close()
                self.log = None

    def pause(self):
        with self.cycle_lock:
            if self.state not in {"running", "paused"}:
                raise ValueError("Start the simulation before pausing automation")
            now = time.monotonic()
            if self.state == "running":
                self.elapsed += now - self.last_automation_time
            self.last_automation_time = now
            self.cadence.reset(now)
            self.state = "paused" if self.state == "running" else "running"
            self.emit("runtime", state=self.state)

    def signal(self, ied_id, reference):
        if self.state != "stopped" and (ied_id, reference) in self.index:
            return self.index[(ied_id, reference)]
        ied = self.projects.find_ied(ied_id)
        result = next((item for item in ied["signals"] if item["ref"] == reference), None)
        if result is None:
            raise ValueError("Unknown signal reference")
        return result

    def update(self, ied_id, reference, value, source="manual", quality=None):
        with self.cycle_lock:
            return self.sample_writer.apply(ied_id, [(reference, value, quality)], source)[reference]

    def release_override(self, ied_id, reference):
        with self.cycle_lock:
            if self.state not in {"running", "paused"}:
                raise ValueError("Run the IED before resuming a source")
            with self.sample_writer.locks[ied_id]:
                spec = self.sources.get((ied_id, reference))
                if not spec or spec["mode"] == "manual":
                    raise ValueError("This signal has no automatic source")
                self.pulses.cancel(ied_id, reference, "Automatic source resumed")
                spec["overridden"] = False
                self.emit("source_resumed", ied_id=ied_id, ref=reference)

    def configure_source(self, ied_id, reference, spec):
        signal = self.signal(ied_id, reference)
        candidate = validate_source(signal, spec)
        if candidate["mode"] != "manual":
            validate_value(signal, value_at(candidate, 0) if signal["kind"].startswith("FLOAT")
                           else int(value_at(candidate, 0)))
        self.projects.source(ied_id, reference, candidate)
        candidate["start_elapsed"] = self.elapsed if self.state in {"running", "paused"} else 0
        with self.lock:
            self.sources[(ied_id, reference)] = candidate
        self.emit("source", ied_id=ied_id, ref=reference, spec=source_summary(candidate))

    def _tick(self):
        while not self.shutdown_event.wait(max(0, self.cadence.deadline - time.monotonic())):
            with self.cycle_lock:
                now = time.monotonic()
                if self.state != "running":
                    self.cadence.reset(now)
                    self.last_automation_time = now
                    continue
                timing = self.cadence.take(now)
                if timing is None:
                    continue
                delta = now - self.last_automation_time
                self.last_automation_time = now
                self.elapsed += delta
                if timing["lateness_seconds"] >= self.cadence.period:
                    self.emit("automation_late", **timing)
                try:
                    self.scenario_runner.advance(self.elapsed)
                    self.automation.advance(self.elapsed)
                except Exception as error:
                    self.state = "paused"
                    self.emit("automation_failed", message=str(error))
                    self.emit("runtime", state="paused", message="Automation failed; MMS remains active")

    def close(self):
        self.shutdown_event.set()
        self.thread.join(timeout=3)
        self.stop()
        self.native_control_feedback.close()
        self.pulses.close()
        self.automation.close()

    def start_scenario(self, key):
        if self.state not in {"running", "paused"}:
            raise ValueError("Run the IEDs before starting a scenario")
        scenario = next((item for item in self.projects.project["scenarios"] if item["id"] == key), None)
        if scenario is None:
            raise ValueError("Unknown scenario")
        self.scenario_runner.start(scenario, self.elapsed)
