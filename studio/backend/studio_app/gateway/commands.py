"""Execute SCADA commands from the gateway on the IEDs of the project over IEC 61850.

Simulated and real IEDs are operated the same way, through studio_control and with the
device's own control model. Commands to real IEDs are allowed by the owner (2026-10-17)
and can be switched off per gateway. Each command is logged in three separate steps:
the request from SCADA, the result applied by the IED, and the position observed after.
"""
import queue
import threading
import time
from studio_app.domain.network_settings import real_endpoint
from studio_app.tools.control_client import summary

OBSERVE_SECONDS = 3.0


class CommandExecutor:
    def __init__(self, manager):
        self.manager = manager
        self.jobs = queue.Queue()
        self.thread = threading.Thread(target=self._run, daemon=True, name="gateway-commands")
        self.thread.start()

    def submit(self, process, event):
        self.jobs.put((process, event))

    def _run(self):
        while True:
            process, event = self.jobs.get()
            ok, reason = self.run(event)
            try:
                process.request("command_result", token=event["token"], ok=ok, reason=reason)
            except Exception:
                pass

    def run(self, event):
        """Execute one command and never raise: (ok, plain reason)."""
        try:
            return self._execute(event)
        except Exception as error:
            return False, str(error)

    def _execute(self, event):
        manager = self.manager
        point = manager.point(event.get("point_id"))
        value = bool(event.get("value"))
        name = point.get("description") if point and point.get("source") == "remote" else point["ref"] if point else None
        label = f"{name or event.get('point_id')} {'ON / close' if value else 'OFF / open'}"
        entry = manager.commands.add(point_id=event.get("point_id"), label=label, protocol=event.get("protocol"),
                                     peer=event.get("peer"), request="received", applied="", observed="")
        if point is not None and point.get("source") == "remote":
            ok, text = manager.converter.relay(point, value, manager.config().get("real_commands", True))
            manager.commands.update(entry["seq"], applied=f"{'operated' if ok else 'refused'} by the device: {text}",
                                    observed="see the device value in Points")
            manager.emit("gateway_command", point=label, result="relayed", ok=ok, reason=text)
            return ok, "" if ok else text
        ied = manager.ied(point["ied"]) if point else None
        reason = refusal(point, ied, manager.config().get("real_commands", True))
        if reason:
            manager.commands.update(entry["seq"], applied=f"refused: {reason}")
            manager.emit("gateway_command", point=label, result="refused", reason=reason)
            return False, reason
        host, port = endpoint(ied, manager.coordinator)
        control = next(item for item in ied["controls"] if item["ref"] == point["ref"])
        originator = "station" if event.get("protocol") == "manual" else "remote"
        result = manager.control.operate(host, port, point["ref"], value, {"originator": originator})
        text = summary(result)
        manager.emit("gateway_command", point=label, ied=ied["name"], role=ied.get("role"), ok=result.get("ok"),
                     result=text)
        if not result.get("ok"):
            manager.commands.update(entry["seq"], applied=f"refused by the device: {text}")
            return False, text
        manager.commands.update(entry["seq"], applied=f"operated: {text}")
        expected = (2 if value else 1) if control.get("cdc") == "DPC" else value
        threading.Thread(target=self._observe, args=(entry["seq"], ied, control["status"], expected),
                         daemon=True).start()
        return True, ""

    def _observe(self, seq, ied, status_ref, expected):
        deadline = time.monotonic() + OBSERVE_SECONDS
        while time.monotonic() < deadline:
            signal = self.manager.coordinator.index.get((ied["id"], status_ref))
            current = signal.get("value") if signal is not None else None
            if type(current) is type(expected) and current == expected:
                self.manager.commands.update(seq, observed=f"{status_ref} is now {position_text(current)}")
                return
            time.sleep(0.1)
        self.manager.commands.update(seq, observed=f"{status_ref} did not change within {OBSERVE_SECONDS:g} s")


def position_text(value):
    """Plain words for an observed status: ON/OFF for a single point, closed/open for a position."""
    if isinstance(value, bool):
        return "ON" if value else "OFF"
    return {0: "intermediate", 1: "open (OFF)", 2: "closed (ON)", 3: "bad state"}.get(value, str(value))


def endpoint(ied, coordinator=None):
    """Where to send a command: the real device, or the address the simulated IED listens on."""
    if ied.get("role") == "real":
        host, port, _ = real_endpoint(ied)
        return host, port
    return coordinator.supervisor.endpoint(ied["id"])


def refusal(point, ied, real_commands=True):
    """Plain reason why this SCADA command is not sent, or "" when it may be."""
    if point is None:
        return "this point is not in the gateway point list"
    if point["kind"] not in ("command_single", "command_double"):
        return "this point is not a command"
    if ied is None:
        return "its IED is not in the project"
    role = ied.get("role", "simulated")
    if role == "off":
        return f"{ied['name']} is switched off in Devices"
    if role == "real" and not real_commands:
        return "commands to real IEDs are switched off in the gateway settings"
    if role == "real" and not endpoint(ied)[0]:
        return f"{ied['name']} has no address; type it in Devices"
    if role == "simulated" and ied.get("state") != "running":
        return f"{ied['name']} is not running; press Run first"
    if point["ref"] not in {item["ref"] for item in ied.get("controls", [])}:
        return f"{ied['name']} has no controllable object {point['ref']}"
    return ""
