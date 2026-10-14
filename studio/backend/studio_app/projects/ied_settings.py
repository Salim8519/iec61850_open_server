"""Change project network mode and per-IED role, endpoint and GOOSE settings."""
from studio_app.contracts.errors import StudioError
from studio_app.domain.network_settings import (
    effective_endpoint, real_endpoint, validate_ied_settings, validate_network)

EDITABLE = {"role", "live", "lab_ip", "goose_publish", "port"}


def endpoint_conflicts(ieds, network):
    """Plain-language problems that would make two IEDs share one address."""
    problems, used = [], {}
    for ied in ieds:
        if ied.get("role", "simulated") == "simulated":
            endpoint = effective_endpoint(ied, network)
        elif ied.get("role") == "real":
            host, port, _ = real_endpoint(ied)
            endpoint = (host, port) if host else None
        else:
            endpoint = None
        if endpoint is None:
            continue
        other = used.get(endpoint)
        if other:
            problems.append(f"{ied['name']} and {other} both use {endpoint[0]}:{endpoint[1]}. "
                            "Give each IED its own address.")
        used[endpoint] = ied["name"]
    return problems


def _merged_network(current, value):
    """A frequency-only change keeps the mode; a mode change keeps the frequency."""
    if set(value) <= {"nominal_hz"}:
        return {**current, **value}
    return {**value, "nominal_hz": value.get("nominal_hz", current.get("nominal_hz", 50))}


def update_network(service, value):
    with service.lock:
        try:
            candidate = validate_network(_merged_network(service.project["network"], value or {}))
        except ValueError as error:
            raise StudioError(str(error)) from error
        problems = endpoint_conflicts(service.project["ieds"], candidate)
        if problems:
            raise StudioError(problems[0], details=problems)
        service.project["network"] = candidate
        service.save()
        service.changed()
        return candidate


def update_ied(service, ied_id, changes):
    if not isinstance(changes, dict) or not changes or not set(changes) <= EDITABLE:
        raise StudioError("Only role, live, lab_ip, goose_publish and port can be changed here")
    with service.lock:
        ied = service.find_ied(ied_id)
        merged = {key: ied.get(key) for key in ("role", "live", "lab_ip", "goose_publish", "test_mode")}
        if "live" in changes:
            changes = {**changes, "live": {**(ied.get("live") or {}), **(changes["live"] or {})}}
        merged.update({key: value for key, value in changes.items() if key != "port"})
        try:
            settings = validate_ied_settings(merged)
            port = changes.get("port", ied.get("port", 102))
            if isinstance(port, bool) or not isinstance(port, int) or not 1 <= port <= 65535:
                raise ValueError("Port must be a whole number from 1 to 65535 (MMS normally uses 102)")
        except ValueError as error:
            raise StudioError(f"{ied['name']}: {error}") from error
        trial = [{**item, **settings, "port": port} if item is ied else item for item in service.project["ieds"]]
        problems = endpoint_conflicts(trial, service.project["network"])
        if problems:
            raise StudioError(problems[0], details=problems)
        ied.update(settings, port=port)
        service.save()
        service.changed()
        return {key: ied[key] for key in ("role", "live", "lab_ip", "goose_publish", "port")}
