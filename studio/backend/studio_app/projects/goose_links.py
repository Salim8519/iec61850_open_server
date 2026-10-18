"""Serve the project's GOOSE link map (declared in SCL, joined with the listener's view)."""
import threading
from studio_app.domain.goose_link_live import with_live
from studio_app.domain.goose_link_map import build_link_map
from studio_app.scl.goose_sources import read_goose_sources
from .scl_sources import stored_sources

_cache = {}
_cache_lock = threading.Lock()


def _declarations(sources):
    """Parsing vendor files takes seconds, so the merged declarations are kept per set of files."""
    key = tuple(sorted((entry["id"], tuple(sorted(entry["ieds"]))) for entry in sources))
    with _cache_lock:
        cached = _cache.get(key)
    if cached is None:
        cached = read_goose_sources([(entry["text"], entry["ieds"]) for entry in sources])
        with _cache_lock:
            _cache.clear()
            _cache[key] = cached
    return cached


def link_map(service, watches=(), running=False):
    with service.lock:
        sources = list(stored_sources(service.project).values())
        ieds = [{"id": ied["id"], "name": ied["name"], "role": ied.get("role", "simulated")}
                for ied in service.project["ieds"]]
    declarations = _declarations(sources) if sources else {
        "ieds": {}, "blocks": {}, "subscriptions": {}, "problems": {}, "remote": {}}
    return with_live(build_link_map(declarations, ieds), list(watches), running)
