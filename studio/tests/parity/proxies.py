"""`app.extensions[...]` proxies: exactly the internal accesses the create_app tests make.

Anything not listed raises InProcessOnly, so an unported access fails loudly.
"""
from parity.errors import InProcessOnly
from parity.live_view import LiveRoot, wrap

NETWORK_SEAMS = ("_list_adapters", "_list_rules", "_probe", "_elevate")


class Proxy:
    service = "?"

    def __init__(self, app):
        object.__setattr__(self, "_app", app)

    @property
    def _engine(self):
        return self._app.engine

    def __getattr__(self, name):
        if name.startswith("__"):
            raise AttributeError(name)
        raise InProcessOnly(f"{self.service}.{name}")


class Projects(Proxy):
    service = "projects"

    @property
    def project(self):
        return wrap({}, LiveRoot(self._engine), [])

    @property
    def revision(self):
        return self._engine.parity("GET", "project/revision")

    def _ieds(self, ieds):
        root = LiveRoot(self._engine)
        return [wrap(ied, root, ["ieds", {"id": ied["id"]}]) for ied in ieds]

    def add_template(self, *args, **kwargs):
        return self._ieds(self._engine.parity("POST", "project/add_template", {"args": list(args), "kwargs": kwargs}))

    def add_scl(self, *args, **kwargs):
        return self._ieds(self._engine.parity("POST", "project/add_scl", {"args": list(args), "kwargs": kwargs}))

    def snapshot(self):
        return self._engine.parity("GET", "project/snapshot")

    def export(self):
        return self._engine.parity("GET", "project/export")

    def _call(self, name, args, kwargs):
        return self._engine.parity("POST", "project/" + name, {"args": list(args), "kwargs": kwargs})

    def restore(self, *args, **kwargs):
        return self._call("restore", args, kwargs)

    def diagram(self, *args, **kwargs):
        return self._call("diagram", args, kwargs)

    def scenarios(self, *args, **kwargs):
        return self._call("scenarios", args, kwargs)

    def source(self, *args, **kwargs):
        return self._call("source", args, kwargs)


class Events(Proxy):
    service = "events"

    def after(self, cursor):
        return self._engine.parity("GET", "events", after=cursor)


class Supervisor(Proxy):
    service = "runtime.supervisor"

    def read(self, ied_id, reference):
        return self._engine.parity("GET", "runtime/read", ied=ied_id, ref=reference)

    @property
    def workers(self):
        return self._engine.parity("GET", "runtime/workers")


class Runtime(Proxy):
    service = "runtime"

    @property
    def state(self):
        return self._engine.parity("GET", "runtime/state")

    @state.setter
    def state(self, value):
        self._engine.parity("POST", "runtime/state", {"state": value})

    @property
    def supervisor(self):
        return Supervisor(self._app)

    def start(self, *args, **kwargs):
        return self._engine.parity("POST", "runtime/start", {"args": list(args), "kwargs": kwargs})

    def close(self):
        if self._engine.process and self._engine.process.poll() is None:
            self._engine.parity("POST", "runtime/close", {})


class Done(Proxy):
    service = "startup.done"

    def is_set(self):
        return self._engine.parity("GET", "startup/done")


class Startup(Proxy):
    service = "startup"

    def wait(self, timeout=None):
        return self._engine.parity("POST", "startup/wait", {"timeout": timeout})

    @property
    def done(self):
        return Done(self._app)


class Journal(Proxy):
    service = "network.journal"

    def outstanding(self):
        return self._engine.parity("GET", "network/journal/outstanding")


class Network(Proxy):
    """Fakes assigned to the Windows calls become callback seams; other attributes a test
    sets (its own `calls` list) stay in the test process, as they did in process."""
    service = "network"

    def __setattr__(self, name, value):
        if name in NETWORK_SEAMS:
            reference = self._app.callbacks.register(value)
            self._engine.parity("POST", "network/seam", {"name": name, "callback": reference})
        object.__setattr__(self, name, value)

    @property
    def journal(self):
        return Journal(self._app)


class Stoppable(Proxy):
    """Tool services the tool tests stop in their teardown."""

    def __init__(self, app, service):
        super().__init__(app)
        object.__setattr__(self, "service", service)

    def __getattr__(self, name):
        if name == "control" and self.service == "explorer":
            return Stoppable(self._app, "explorer.control")
        if name.startswith("__"):
            raise AttributeError(name)
        call = f"{self.service}.{name}"
        return lambda: self._engine.parity("POST", "service/stop", {"call": call})


def make(app, name):
    simple = {"projects": Projects, "events": Events, "runtime": Runtime, "startup": Startup, "network": Network}
    if name in simple:
        return simple[name](app)
    if name in ("explorer", "scanner", "gateway", "master"):
        return Stoppable(app, name)
    raise InProcessOnly(f"app.extensions[{name!r}]")
