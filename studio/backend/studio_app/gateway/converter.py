"""Protocol converter role: the gateway serves what the SCADA master reads from the device.

Values come from the master's discovered table instead of the IEDs; an upstream SCADA
command on a remote point is relayed to the device through the master, after the same
"commands to real devices" switch as the station gateway.
"""
from studio_app.domain.master_points import quality_words
from studio_app.domain.passthrough import device_value, map_points, remote_sample


class ConverterSource:
    def __init__(self, master):
        self.master = master

    def points(self, protocol):
        """(upstream points, notes); refuses until the device is connected and has points."""
        if not self.master.link_open:
            raise ValueError("Connect to the device first: the converter serves the points ScadaBench reads from it")
        points, notes = map_points(self.master.points(), protocol, self.master.config()["routes"])
        if not points:
            raise ValueError("No watched points from the device yet. Ask for all values, or tick the points to serve.")
        return points, notes

    def sample(self, point):
        device = self.master.device[0] if self.master.device else self.master.config()["protocol"]
        with self.master.table_lock:
            row = dict(self.master.table.get(point.get("remote"), {})) or None
        good = row is not None and not quality_words(device, row.get("quality", 0), row.get("group"))
        return remote_sample(point, row, good, self.master.link_open)

    def table(self, protocol):
        """The mapping as gateway point rows, with what each point carries now."""
        if not self.master.link_open:
            return []
        values = {row["key"]: row for row in self.master.points()}
        rows = []
        for point in map_points(self.master.points(), protocol, self.master.config()["routes"])[0]:
            row = values.get(point.get("remote"))
            text = f"{row['value_text']} · {row['quality_text']}" if row else "relayed to the device"
            rows.append({**point, "ied_name": "Device", "ied_role": "real", "monitor_only": False,
                         "value_text": text, "online": bool(row and row["good"] and not row["stale"])})
        return rows

    def relay(self, point, value, allowed):
        """Send an upstream command on to the device: (ok, plain reason)."""
        route = point.get("route")
        if route is None:
            return False, "this point is not a command"
        if not allowed:
            return False, "commands to real devices are switched off in the gateway settings"
        kind = route["kind"]
        command = {"kind": kind, "address": route["address"], "value": device_value(route, value),
                   "select": route.get("select", False)}
        if kind == "binary_output":
            command["code"] = "latch"
        answer = self.master.command(command, who="SCADA via the converter")
        return bool(answer.get("ok")), answer.get("text", "")
