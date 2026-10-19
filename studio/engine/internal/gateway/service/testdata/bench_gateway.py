"""CPU of the Python gateway value push with N points and a value change rate.

The frozen GatewayManager runs against a real studio_gateway; a fake coordinator holds the
signal index and a thread changes `rate` values per second. Prints one JSON line with the
backend process CPU seconds over the measuring window. Synthetic points only.
"""
import json
import os
import sys
import threading
import time
from pathlib import Path

sys.path.insert(0, os.environ["STUDIO_BACKEND"])
from studio_app.domain.gateway_rules import empty_gateway  # noqa: E402
from studio_app.gateway.manager import GatewayManager  # noqa: E402

points_count, rate, seconds, port = (int(value) for value in sys.argv[1:5])
BIN = Path(os.environ["STUDIO_NATIVE_BIN"])


def refs(number):
    if number % 2:
        return f"BayCTRL/GGIO1.AnIn{number}.mag.f", "FLOAT32", "float", 1.5
    return f"BayCTRL/GGIO1.Ind{number}.stVal", "BOOLEAN", "single", False


signals, points, index = [], [], {}
for number in range(points_count):
    ref, kind, point_kind, value = refs(number)
    record = {"ref": ref, "fc": "MX" if kind == "FLOAT32" else "ST", "kind": kind, "value": value,
              "quality_ref": ref.rsplit(".", 1)[0] + ".q", "timestamp_ref": ref.rsplit(".", 1)[0] + ".t"}
    signals.append(record)
    index[("bay", ref)] = record
    index[("bay", record["quality_ref"])] = {"value": 0}
    index[("bay", record["timestamp_ref"])] = {"value": 1700000000000}
    points.append({"kind": point_kind, "ied": "bay", "ref": ref, "address": number + 1, "scale": 1.0, "description": ""})
ied = {"id": "bay", "name": "Bay", "role": "simulated", "state": "running", "signals": signals, "controls": []}
gateway = empty_gateway()
gateway["settings"]["iec104"]["port"] = port
gateway["points"]["iec104"] = points


class Coordinator:
    state = "running"
    index = index

    def emit(self, kind, **payload):
        pass


class Projects:
    project = {"ieds": [ied], "gateway": gateway}

    def gateway(self, config):
        self.project["gateway"] = config


manager = GatewayManager(Coordinator(), Projects(), BIN / "studio_ied.exe")
manager.start()
stop = threading.Event()


def change():
    step, number = 0, 0
    while rate and not stop.is_set():
        for _ in range(max(1, rate // 100)):
            ref, kind, _, _ = refs(number % points_count)
            index[("bay", ref)]["value"] = (step % 100) / 10 if kind == "FLOAT32" else bool(step % 2)
            number += 1
        step += 1
        stop.wait(0.01)


thread = threading.Thread(target=change, daemon=True)
thread.start()
time.sleep(2)  # settle: the first full batch has left
cpu, wall = time.process_time(), time.perf_counter()
time.sleep(seconds)
cpu, wall = time.process_time() - cpu, time.perf_counter() - wall
stop.set()
manager.stop(remember=False)
print(json.dumps({"engine": "python", "points": points_count, "rate": rate, "cpu_seconds": cpu, "wall_seconds": wall,
                  "cpu_percent_of_one_core": 100 * cpu / wall}))
