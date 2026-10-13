"""Execute deterministic one-shot timelines using the coordinator's paused clock."""
import copy
import math
import threading


class ScenarioRunner:
    def __init__(self, apply, read, emit):
        self.apply = apply
        self.read = read
        self.emit = emit
        self.lock = threading.RLock()
        self.runs = {}

    def reset(self, scenarios):
        with self.lock:
            self.runs.clear()
            for scenario in scenarios:
                if scenario["enabled"]:
                    self.start(scenario, 0)

    def start(self, scenario, elapsed):
        with self.lock:
            existing = self.runs.get(scenario["id"])
            if existing and existing["state"] == "running":
                raise ValueError("This scenario is already running")
            self.runs[scenario["id"]] = {"scenario": copy.deepcopy(scenario), "start": elapsed,
                                        "next": 0, "state": "running"}
            self.emit("scenario_started", scenario_id=scenario["id"], name=scenario["name"])

    def stop(self):
        with self.lock:
            for key, run in self.runs.items():
                if run["state"] == "running":
                    run["state"] = "cancelled"
                    self.emit("scenario_cancelled", scenario_id=key)

    def advance(self, elapsed):
        with self.lock:
            for key, run in self.runs.items():
                steps = run["scenario"]["steps"]
                while run["state"] == "running" and run["next"] < len(steps):
                    step = steps[run["next"]]
                    if step["at"] > elapsed - run["start"]:
                        break
                    try:
                        if step["action"] == "set":
                            actual = self.apply(step["ied_id"], step["ref"], step["value"], "scenario")
                        else:
                            actual = self.read(step["ied_id"], step["ref"])
                            expected = step["value"]
                            equal = (math.isclose(actual, expected, rel_tol=1e-6, abs_tol=1e-6)
                                     if isinstance(expected, float) else actual == expected)
                            if not equal:
                                raise ValueError(f"Assertion failed: expected {expected}, observed {actual}")
                        run["next"] += 1
                        self.emit("scenario_step_applied", scenario_id=key, step_id=step["id"],
                                  ied_id=step["ied_id"], ref=step["ref"], action=step["action"],
                                  value=actual, scheduled_seconds=step["at"],
                                  observed_seconds=elapsed - run["start"])
                        if run["next"] == len(steps):
                            run["state"] = "completed"
                            self.emit("scenario_completed", scenario_id=key)
                    except Exception as error:
                        run["state"] = "failed"
                        self.emit("scenario_failed", scenario_id=key, step_id=step["id"], message=str(error))

    def snapshot(self):
        with self.lock:
            return [{"id": key, "state": run["state"], "completed_steps": run["next"]}
                    for key, run in self.runs.items()]
