import copy

import pytest
from studio_app.domain.scenario import validate_scenarios
from studio_app.simulation.scenario_runner import ScenarioRunner


def timeline():
    return {"id": "sequence", "name": "Voltage change", "enabled": True, "steps": [
        {"id": "set", "at": 1, "ied_id": "ied", "ref": "value", "action": "set", "value": 110.0},
        {"id": "check", "at": 2, "ied_id": "ied", "ref": "value", "action": "assert", "value": 110.0}]}


def test_timeline_order_once_and_explicit_restart():
    values, events = {}, []
    def apply(ied, ref, value, source):
        values[ref] = value
        return value
    runner = ScenarioRunner(apply, lambda ied, ref: values.get(ref),
                            lambda kind, **payload: events.append((kind, payload)))
    runner.reset([timeline()])
    runner.advance(0.9)
    assert values == {}
    runner.advance(1)
    assert values == {"value": 110.0}
    runner.advance(1)  # A paused clock must not repeat a step.
    runner.advance(2)
    runner.advance(100)
    assert runner.snapshot()[0]["state"] == "completed"
    assert len([kind for kind, _ in events if kind == "scenario_step_applied"]) == 2
    runner.start(timeline(), 100)
    runner.stop()
    assert runner.snapshot()[0]["state"] == "cancelled"


def test_failed_assertion_stops_only_its_sequence():
    failed = timeline()
    failed["steps"] = [failed["steps"][1], failed["steps"][0]]
    runner = ScenarioRunner(lambda *args: pytest.fail("Must not apply after failed assertion"),
                            lambda *args: 0, lambda *args, **kwargs: None)
    runner.reset([failed])
    runner.advance(10)
    assert runner.snapshot()[0]["state"] == "failed"
    assert runner.snapshot()[0]["completed_steps"] == 0


def test_scenario_validation_and_sorting():
    ieds = [{"id": "ied", "signals": [{"ref": "value", "kind": "FLOAT32", "fc": "MX", "writable": True}]}]
    scenario = timeline()
    scenario["steps"].reverse()
    assert validate_scenarios([scenario], ieds)[0]["steps"][0]["at"] == 1
    invalid = copy.deepcopy(scenario)
    invalid["steps"][0]["ref"] = "unknown"
    with pytest.raises(ValueError, match="Unknown"):
        validate_scenarios([invalid], ieds)
