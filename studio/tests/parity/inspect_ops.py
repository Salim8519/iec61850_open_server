"""Parity inspect operations (python-http side of tests/03-parity-harness.md "Parity inspect API").

Each operation mirrors one in-process access the 28 create_app test files make. Values use
parity.wire tagged JSON. The Go parity build answers the same (method, path) table.
Project paths address list items by id ({"id": ...}) so a returned IED stays the same IED.
"""
from parity.callbacks import remote_function

NETWORK_SEAMS = {"_list_adapters", "_list_rules", "_probe", "_elevate"}
STOP_CALLS = {"explorer.disconnect", "explorer.close", "explorer.control.close", "scanner.stop",
              "gateway.stop", "master.disconnect"}


def _ext(context, name):
    return context["extensions"][name]


def walk(tree, path):
    for step in path:
        if isinstance(step, dict):
            tree = next(item for item in tree if isinstance(item, dict) and item.get("id") == step["id"])
        else:
            tree = tree[step]
    return tree


def project_raw(context, body, query):
    return _ext(context, "projects").project


def project_set(context, body, query):
    """Write-through of `projects.project[...][key] = value` made by a test."""
    target = walk(_ext(context, "projects").project, body["path"])
    target[body["key"]] = body["value"]


def project_update(context, body, query):
    """Write-through of `ied.update(ip=..., port=...)` on a dict a test received."""
    walk(_ext(context, "projects").project, body["path"]).update(body["values"])


def project_call(name):
    def call(context, body, query):
        return getattr(_ext(context, "projects"), name)(*body.get("args", []), **body.get("kwargs", {}))
    return call


def project_revision(context, body, query):
    return _ext(context, "projects").revision


def events_after(context, body, query):
    return _ext(context, "events").after(int(query.get("after", 0)))


def runtime_state(context, body, query):
    return _ext(context, "runtime").state


def runtime_set_state(context, body, query):
    _ext(context, "runtime").state = body["state"]


def runtime_start(context, body, query):
    return _ext(context, "runtime").start(*body.get("args", []), **body.get("kwargs", {}))


def runtime_close(context, body, query):
    _ext(context, "runtime").close()


def runtime_read(context, body, query):
    return _ext(context, "runtime").supervisor.read(query["ied"], query["ref"])


def runtime_workers(context, body, query):
    workers = _ext(context, "runtime").supervisor.workers
    return {key: getattr(getattr(worker, "process", None), "pid", None) for key, worker in workers.items()}


def startup_wait(context, body, query):
    return _ext(context, "startup").wait(body.get("timeout"))


def startup_done(context, body, query):
    return _ext(context, "startup").done.is_set()


def network_seam(context, body, query):
    if body["name"] not in NETWORK_SEAMS:
        raise ValueError(f"not a network seam: {body['name']}")
    setattr(_ext(context, "network"), body["name"], remote_function(body["callback"]))


def network_outstanding(context, body, query):
    return _ext(context, "network").journal.outstanding()


def service_stop(context, body, query):
    """Tool teardown the tests do in process (explorer/scanner/gateway/master stops)."""
    if body["call"] not in STOP_CALLS:
        raise ValueError(f"not a parity stop call: {body['call']}")
    service, *rest = body["call"].split(".")
    target = _ext(context, service)
    for name in rest[:-1]:
        target = getattr(target, name)
    return getattr(target, rest[-1])()


def install_fault(context, body, query):
    from parity.server_seams import apply_faults
    apply_faults({body["target"]: body["callback"]})


def shutdown(context, body, query):
    context["stopping"].set()


OPERATIONS = {
    ("GET", "project/raw"): project_raw,
    ("POST", "project/set"): project_set,
    ("POST", "project/update"): project_update,
    ("GET", "project/snapshot"): project_call("snapshot"),
    ("GET", "project/export"): project_call("export"),
    ("GET", "project/revision"): project_revision,
    ("POST", "project/restore"): project_call("restore"),
    ("POST", "project/diagram"): project_call("diagram"),
    ("POST", "project/scenarios"): project_call("scenarios"),
    ("POST", "project/source"): project_call("source"),
    ("POST", "project/add_template"): project_call("add_template"),
    ("POST", "project/add_scl"): project_call("add_scl"),
    ("GET", "events"): events_after,
    ("GET", "runtime/state"): runtime_state,
    ("POST", "runtime/state"): runtime_set_state,
    ("POST", "runtime/start"): runtime_start,
    ("POST", "runtime/close"): runtime_close,
    ("GET", "runtime/read"): runtime_read,
    ("GET", "runtime/workers"): runtime_workers,
    ("POST", "startup/wait"): startup_wait,
    ("GET", "startup/done"): startup_done,
    ("POST", "network/seam"): network_seam,
    ("GET", "network/journal/outstanding"): network_outstanding,
    ("POST", "service/stop"): service_stop,
    ("POST", "fault"): install_fault,
    ("POST", "shutdown"): shutdown,
}
