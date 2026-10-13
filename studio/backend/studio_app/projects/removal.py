"""Describe and apply IED removal without deleting physical diagram equipment."""
import copy
from studio_app.domain.equipment_state import binding_keys


def _uses_ied(data, ied_ids):
    state = any(ied_id in ied_ids for ied_id, _ in binding_keys(data.get("state_mapping")))
    relay = any(entry["source"]["ied_id"] in ied_ids for entry in data.get("relay_statuses") or [])
    control = data.get("control_binding")
    return state or relay or bool(control and control["ied_id"] in ied_ids)


def impact(project, ied_id):
    return impacts(project, {ied_id})


def impacts(project, ied_ids):
    steps = [step for scenario in project["scenarios"] for step in scenario["steps"]
             if step["ied_id"] in ied_ids]
    return {"bindings": sum(_uses_ied(node["data"], ied_ids) for node in project["diagram"]["nodes"]),
            "sources": sum(item["ied_id"] in ied_ids for item in project["sources"]),
            "test_controls": sum(item["ied_id"] in ied_ids for item in project.get("test_panel", [])),
            "scenario_steps": len(steps),
            "scenarios_removed": sum(all(step["ied_id"] in ied_ids for step in scenario["steps"])
                                     for scenario in project["scenarios"])}


def without_ied(project, ied_id):
    return without_ieds(project, {ied_id})


def without_ieds(project, ied_ids):
    result = copy.deepcopy(project)
    result["ieds"] = [ied for ied in result["ieds"] if ied["id"] not in ied_ids]
    result["sources"] = [item for item in result["sources"] if item["ied_id"] not in ied_ids]
    result["test_panel"] = [item for item in result.get("test_panel", []) if item["ied_id"] not in ied_ids]
    for node in result["diagram"]["nodes"]:
        data = node["data"]
        if any(ied_id in ied_ids for ied_id, _ in binding_keys(data.get("state_mapping"))):
            data["state_mapping"] = None
        if data.get("relay_statuses"):
            data["relay_statuses"] = [entry for entry in data["relay_statuses"]
                                      if entry["source"]["ied_id"] not in ied_ids]
        if (data.get("control_binding") or {}).get("ied_id") in ied_ids:
            data["control_binding"] = None
    for scenario in result["scenarios"]:
        scenario["steps"] = [step for step in scenario["steps"] if step["ied_id"] not in ied_ids]
    result["scenarios"] = [scenario for scenario in result["scenarios"] if scenario["steps"]]
    return result
