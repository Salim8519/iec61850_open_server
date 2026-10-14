"""Mode badges for devices and diagram equipment, and the simulation-traffic notice.

Pure. Real IEDs: mode as read from the device. Simulated IEDs: the mode Studio set.
"""
from .health_reads import MODE_NAMES
from .test_mode import mode_points, mode_summary


def _simulated_summary(ied):
    settings = ied.get("test_mode") or {}
    devices, _ = mode_points(ied)
    names = {domain: MODE_NAMES[settings.get("devices", {}).get(domain, 1)] for domain in devices}
    order = list(MODE_NAMES.values())
    worst = max(names.values(), key=order.index, default=None)
    return {"devices": names, "simulation": bool(settings.get("simulation")), "worst": worst}


def _bound_ied(data):
    binding = data.get("control_binding")
    mapping = data.get("state_mapping") or {}
    source = mapping.get("source") or mapping.get("open")
    for item in (binding, source):
        if item and item.get("ied_id") and "/" in item.get("ref", ""):
            return item["ied_id"], item["ref"].split("/", 1)[0]
    return None


def mode_badges(ieds, sessions, diagram, running):
    """{ieds: {id: summary}, equipment: {node id: text}} for anything not in normal service."""
    summaries = {}
    for ied in ieds:
        if ied.get("role") == "simulated":
            summaries[ied["id"]] = _simulated_summary(ied)
        elif ied.get("role") == "real" and running:
            health = (sessions.get(ied["id"]) or {}).get("health")
            if health:
                summaries[ied["id"]] = mode_summary(ied, health["mode"]["values"])
    equipment = {}
    for node in diagram.get("nodes", []):
        bound = _bound_ied(node["data"])
        summary = summaries.get(bound[0]) if bound else None
        name = (summary or {}).get("devices", {}).get(bound[1]) if bound else None
        if name and name not in {"on", "not available"}:
            equipment[node["id"]] = name
        elif summary and summary.get("simulation"):
            equipment[node["id"]] = "simulation"
    return {"ieds": summaries, "equipment": equipment}


def simulation_sources(goose, sv):
    """Every observed source of simulation/test-flagged GOOSE or SV traffic."""
    sources = []
    for watch in goose.get("watches", []):
        if watch.get("received_test") and watch.get("state") in {"receiving", "problem"}:
            sources.append(f"GOOSE {watch['gocb_ref']} from {(watch.get('publisher') or {}).get('name') or 'unknown'}")
    sources += [f"GOOSE {item.get('gocb_ref')} (not in project)" for item in goose.get("unknown", []) if item.get("test")]
    for watch in sv.get("watches", []):
        if "simulation" in watch.get("problems", []):
            sources.append(f"Sampled Values {watch.get('sv_id') or watch['key']} from {watch.get('publisher')}")
    sources += [f"Sampled Values {item.get('sv_id') or item.get('app_id')} (not in project)"
                for item in sv.get("unknown", []) if item.get("simulate")]
    return sources
