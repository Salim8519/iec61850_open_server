"""Validate portable timed scenario definitions and referenced signal types."""
import math
import re
from .signal import validate_value


def validate_scenarios(value, ieds):
    if not isinstance(value, list) or len(value) > 100:
        raise ValueError("Expected at most 100 scenarios")
    signals = {(ied["id"], item["ref"]): item for ied in ieds for item in ied["signals"]}
    result, ids = [], set()
    for scenario in value:
        if not isinstance(scenario, dict):
            raise ValueError("Invalid scenario definition")
        key = scenario.get("id", "")
        if not isinstance(key, str) or not re.fullmatch(r"[A-Za-z0-9_-]{1,64}", key) or key in ids:
            raise ValueError("Scenario identifiers must be valid and unique")
        ids.add(key)
        name = scenario.get("name", "")
        if not isinstance(name, str) or not 1 <= len(name.strip()) <= 100:
            raise ValueError("Scenario name must contain 1 to 100 characters")
        enabled = scenario.get("enabled", False)
        if not isinstance(enabled, bool):
            raise ValueError("Scenario enabled must be true or false")
        steps = scenario.get("steps")
        if not isinstance(steps, list) or not 1 <= len(steps) <= 1000:
            raise ValueError("Each scenario needs 1 to 1000 steps")
        normalized, step_ids = [], set()
        for step in steps:
            if not isinstance(step, dict):
                raise ValueError("Invalid scenario step")
            step_id = step.get("id", "")
            if (not isinstance(step_id, str) or not re.fullmatch(r"[A-Za-z0-9_-]{1,64}", step_id)
                    or step_id in step_ids):
                raise ValueError("Step identifiers must be valid and unique")
            step_ids.add(step_id)
            at = step.get("at")
            if (isinstance(at, bool) or not isinstance(at, (int, float))
                    or not math.isfinite(at) or not 0 <= at <= 86400):
                raise ValueError("Step time must be 0 to 86400 seconds")
            ied_id, reference = step.get("ied_id"), step.get("ref")
            if not isinstance(ied_id, str) or not isinstance(reference, str):
                raise ValueError("Choose a signal for each scenario step")
            signal = signals.get((ied_id, reference))
            action = step.get("action", "set")
            if signal is None or action not in {"set", "assert"}:
                raise ValueError("Unknown scenario signal or action")
            if action == "set" and (not signal["writable"] or signal["fc"] == "CF"):
                raise ValueError("Scenario cannot write this signal")
            expected = validate_value(signal, step.get("value"))
            normalized.append({"id": step_id, "at": float(at), "ied_id": ied_id,
                               "ref": reference, "action": action, "value": expected})
        result.append({"id": key, "name": name.strip(), "enabled": enabled,
                       "steps": sorted(normalized, key=lambda item: item["at"])})
    return result
