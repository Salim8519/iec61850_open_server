"""Autonomous IED Explorer: restore or build the watch list and start watching by itself."""
from studio_app.domain.explorer_watch import key_nodes, key_points, merge


def endpoint_key(host, port):
    return f"{host}:{port}"


def build_watch(service):
    """Open only the key logical nodes and take their key points."""
    points = []
    for device in service.devices():
        for node in key_nodes(device, service.nodes(device)):
            try:
                points += key_points(service.node(node))
            except RuntimeError:
                continue
    return merge(points)


def start_watching(service, memory):
    """Remembered watch list for this device, otherwise the key points; then poll."""
    host, port = service.endpoint
    remembered = memory.get()["explorer"]
    items = remembered["watch_by_host"].get(endpoint_key(host, port)) or build_watch(service)
    if not items:
        return service.status()
    service.set_watch(items, remembered["interval_ms"])
    return service.start_polling()


def remember(service, memory):
    """Save the endpoint, interval and watch list after every change."""
    if not service.endpoint:
        return
    host, port = service.endpoint
    current = memory.get()["explorer"]
    watches = dict(current["watch_by_host"])
    watches[endpoint_key(host, port)] = [{"ref": item["ref"], "fc": item["fc"], "type": item.get("type", "")}
                                         for item in service.watch]
    memory.update("explorer", host=host, port=port, interval_ms=service.interval_ms, watch_by_host=watches)
