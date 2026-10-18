"""Join the declared GOOSE link map with what the listener last observed. Pure."""

LIVE_FIELDS = ("state", "problem", "st_num", "sq_num", "observed_at", "frames")


def with_live(link_map, watches, running):
    """Copy of the map where each declared link carries the listener's view of its block.

    A link is never marked received from the declaration alone: without a watch it has
    `live: None`, and while stopped every declared link reads `stopped`.
    """
    by_ref = {watch["gocb_ref"]: watch for watch in watches}
    links = []
    for link in link_map["links"]:
        live = None
        if link["status"] == "declared":
            watch = by_ref.get(link["gocb_ref"])
            if not running:
                live = {"state": "stopped"}
            elif watch is not None and watch.get("state") not in (None, "stopped"):
                # While running, "stopped" means the listener is off (no lab mode or no Npcap): not watched.
                live = {key: watch.get(key) for key in LIVE_FIELDS if watch.get(key) is not None}
        links.append({**link, "live": live})
    return {**link_map, "links": links, "running": running, "summary": summary(links)}


def summary(links):
    """Per project IED: how many it sends to, whom it gets from, and what is not heard."""
    result = {}

    def entry(name):
        return result.setdefault(name, {"sends_to": 0, "gets_from": [], "not_sent": 0, "remote": 0,
                                        "not_heard": 0, "problems": 0})
    for link in links:
        publisher, subscriber = entry(link["publisher"]), entry(link["subscriber"])
        if link["status"] == "declared":
            publisher["sends_to"] += 1
        if link["publisher"] not in subscriber["gets_from"]:
            subscriber["gets_from"].append(link["publisher"])
        if link["status"] == "remote":
            subscriber["remote"] += 1
        elif link["status"] != "declared":
            subscriber["not_sent"] += 1
        state = (link.get("live") or {}).get("state")
        if state in {"waiting", "timeout"}:
            subscriber["not_heard"] += 1
        elif state == "problem":
            subscriber["problems"] += 1
    return result
