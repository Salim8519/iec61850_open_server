"""Who reads a simulated IED: merge the worker's per-second summaries into a client list. Pure.

The worker counts reads per logical node, control writes and the report blocks a
client enabled, and sends one summary per second (ADR-0066). Clients are identified by
their IP address and port, which is all MMS tells a server about its client.
"""
READING_S = 5
SHOWN_NODES = 12


def _ip(peer):
    return peer.rsplit(":", 1)[0] if ":" in peer else peer


def merge_summary(clients, summary, now):
    """Update {peer: client} in place from one `mms_access` summary; drop clients that left."""
    for item in summary:
        peer = item.get("peer") or ""
        if not peer:
            continue
        if not item.get("connected", True):
            clients.pop(peer, None)
            continue
        entry = clients.setdefault(peer, {"peer": peer, "ip": _ip(peer), "first_seen": now, "last_read": None,
                                          "reads": {}, "writes": {}, "total_reads": 0, "rcbs": [], "own": False})
        for node, count in (item.get("reads") or {}).items():
            entry["reads"][node] = entry["reads"].get(node, 0) + int(count)
        for name, count in (item.get("writes") or {}).items():
            entry["writes"][name] = entry["writes"].get(name, 0) + int(count)
        if item.get("reads"):
            entry["last_read"] = now
        entry["total_reads"] = int(item.get("total_reads", entry["total_reads"]))
        entry["rcbs"] = sorted(item.get("rcbs") or [])
    return clients


def mark_own_peer(clients, peer, now):
    """ScadaBench's own command client (studio_probe) is shown as ScadaBench, never as a reader."""
    entry = clients.setdefault(peer, {"peer": peer, "ip": _ip(peer), "first_seen": now, "last_read": None,
                                      "reads": {}, "writes": {}, "total_reads": 0, "rcbs": [], "own": True})
    entry["own"] = True


def client_view(clients, now):
    """Display rows, most active first: what each client reads, writes and receives as reports."""
    rows = []
    for entry in clients.values():
        nodes = sorted(entry["reads"].items(), key=lambda item: (-item[1], item[0]))
        rows.append({"peer": entry["peer"], "ip": entry["ip"], "own": entry["own"],
                     "this_pc": entry["ip"].startswith("127."),
                     "reading": entry["last_read"] is not None and now - entry["last_read"] <= READING_S,
                     "reads": [{"node": node, "count": count} for node, count in nodes[:SHOWN_NODES]],
                     "more_nodes": max(0, len(nodes) - SHOWN_NODES), "total_reads": entry["total_reads"],
                     "writes": sum(entry["writes"].values()), "rcbs": entry["rcbs"],
                     "connected_s": round(now - entry["first_seen"])})
    return sorted(rows, key=lambda row: (row["own"], -row["total_reads"], row["peer"]))
