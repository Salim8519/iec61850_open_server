"""Operate IEC 61850 controls of real or simulated IEDs through studio_control.

One warm worker serves every command, one at a time, so a SCADA command through the
gateway is answered within the device's own operate time. The owner allows commands to
real IEDs (2026-10-17); manual commands need an explicit confirmation from the page.
"""
import threading
from studio_app.runtime.worker_process import WorkerProcess

OR_CATEGORIES = {"bay": 1, "station": 2, "remote": 3, "maintenance": 7}


def control_options(options):
    """Validate the optional control parameters a user may change."""
    options = options or {}
    category = options.get("originator", "remote")
    if category not in OR_CATEGORIES:
        raise ValueError("Originator must be bay, station, remote or maintenance control")
    flags = {}
    for key in ("interlock_check", "synchro_check", "test"):
        value = options.get(key, key == "interlock_check")
        if not isinstance(value, bool):
            raise ValueError(f"{key.replace('_', ' ')} is on or off")
        flags[key] = value
    return {"or_cat": OR_CATEGORIES[category], **flags}


def control_value(value):
    if isinstance(value, bool) or (isinstance(value, int) and -2147483648 <= value <= 2147483647):
        return value
    raise ValueError("A control value is on/off (true/false) or a whole number step")


class ControlClient:
    def __init__(self, executable):
        self.executable = executable
        self.lock = threading.Lock()
        self.process = None

    def _worker(self):
        if self.process is None or self.process.process.poll() is not None:
            if not self.executable.exists():
                raise RuntimeError("The control program (studio_control.exe) is not built yet; rebuild the native programs")
            process = WorkerProcess(self.executable)
            if "control_operate" not in process.request("hello").get("capabilities", []):
                process.close()
                raise RuntimeError("The control program is outdated; rebuild the native programs")
            self.process = process
        return self.process

    def operate(self, host, port, reference, value, options=None):
        """Return the device's answer: ok, model, selected, operated, terminated, add_cause_text, error."""
        settings = control_options(options)
        with self.lock:
            return self._worker().request("operate", timeout=30, ip=host, port=port, ref=reference,
                                          value=control_value(value), or_ident="ScadaBench", **settings)

    def close(self):
        with self.lock:
            process, self.process = self.process, None
        if process is not None:
            process.close()


def summary(result):
    """One plain sentence for logs and the page."""
    if result.get("ok"):
        how = "selected and operated" if result.get("selected") else "operated"
        return f"The device {how} it ({result.get('model')})."
    return result.get("error") or "The device refused the command."
