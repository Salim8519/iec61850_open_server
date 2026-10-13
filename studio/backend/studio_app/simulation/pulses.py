"""Backend-owned Boolean pulses with native revision-guarded restoration."""
import threading
import time
from uuid import uuid4


class Pulses:
    def __init__(self, coordinator):
        self.coordinator = coordinator
        self.pending = {}
        self.lock = threading.RLock()
        self.closed = threading.Event()
        self.thread = threading.Thread(target=self._run, daemon=True, name="studio-pulses")
        self.thread.start()

    def start(self, ied_id, reference, active, duration):
        c = self.coordinator
        if not isinstance(active, bool) or isinstance(duration, bool) or not isinstance(duration, int) or not 100 <= duration <= 60000:
            raise ValueError("Pulse requires a Boolean value and 100 to 60000 milliseconds")
        with c.cycle_lock:
            if c.state not in {"running", "paused"}:
                raise ValueError("Run the IED before triggering a pulse")
            if ied_id not in c.sample_writer.locks:
                raise ValueError("Pulses can only be sent to simulated IEDs")
            signal = c.signal(ied_id, reference)
            if signal["kind"] != "BOOLEAN":
                raise ValueError("Pulses require a Boolean signal")
            with c.sample_writer.locks[ied_id]:
                before = c.supervisor.read_versioned(ied_id, reference)
                result = c.sample_writer.apply(ied_id, [(reference, active, None)], "pulse",
                                               {"ref": reference, "revision": before["revision"]})
                self.cancel(ied_id, reference, "Replaced by a newer pulse")
                pulse = {"id": uuid4().hex, "ied_id": ied_id, "ref": reference,
                         "previous": before["value"], "revision": result["revision"],
                         "deadline": time.monotonic() + duration / 1000}
                with self.lock:
                    self.pending[(ied_id, reference)] = pulse
                c.emit("pulse_started", correlation_id=pulse["id"], ied_id=ied_id, ref=reference, duration_ms=duration)
                return {"id": pulse["id"], "value": active}

    def cancel(self, ied_id=None, reference=None, reason="Simulation stopped"):
        with self.lock:
            removed = [self.pending.pop(key) for key in list(self.pending)
                       if (ied_id is None or key[0] == ied_id) and (reference is None or key[1] == reference)]
        for pulse in removed:
            self.coordinator.emit("pulse_cancelled", correlation_id=pulse["id"],
                                  ied_id=pulse["ied_id"], ref=pulse["ref"], message=reason)

    def advance(self):
        c = self.coordinator
        with c.cycle_lock:
            with self.lock:
                due = [self.pending.pop(key) for key, pulse in list(self.pending.items())
                       if pulse["deadline"] <= time.monotonic()]
            for pulse in due:
                try:
                    c.sample_writer.apply(pulse["ied_id"], [(pulse["ref"], pulse["previous"], None)], "pulse_return",
                                          {"ref": pulse["ref"], "revision": pulse["revision"]})
                    kind, message = "pulse_completed", "Previous value restored"
                except Exception as error:
                    kind, message = "pulse_failed", str(error)
                c.emit(kind, correlation_id=pulse["id"], ied_id=pulse["ied_id"], ref=pulse["ref"], message=message)

    def _run(self):
        while not self.closed.wait(0.02):
            self.advance()

    def close(self):
        self.closed.set()
        self.thread.join(timeout=12)
        self.cancel()
