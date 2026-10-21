"""`create_app` replacement for HTTP engines: same signature, returns a RemoteApp.

The conftest engine switch installs `remote_create_app` as studio_app.app_factory.create_app
before any test module imports it, so the 28 files that build an app need no edit.
"""
from parity import launch, seam_translate
from parity.callbacks import CallbackServer
from parity.errors import InProcessOnly
from parity.http_client import HttpClient
from parity import proxies

LIVE = []           # RemoteApps of the running test; the plugin shuts them down after it
PENDING_FAULTS = []  # (dotted target, fake) recorded from monkeypatch.setattr


class Extensions:
    def __init__(self, app):
        self._app, self._made = app, {}

    def __getitem__(self, name):
        if name not in self._made:
            self._made[name] = proxies.make(self._app, name)
        return self._made[name]

    def get(self, name, default=None):
        try:
            return self[name]
        except InProcessOnly:
            return default

    def __contains__(self, name):
        return self.get(name) is not None


class RemoteApp:
    def __init__(self, engine, callbacks, config):
        self.engine, self.callbacks, self.config = engine, callbacks, config
        self.extensions = Extensions(self)

    def test_client(self):
        return HttpClient(self.engine)

    def install_fault(self, target, function):
        self.engine.parity("POST", "fault", {"target": target, "callback": self.callbacks.register(function)})

    def shutdown(self):
        try:
            return self.engine.shutdown()
        finally:
            self.callbacks.close()


def remote_create_app(settings=None, token=None, secret_key=None, admin_origin=None,
                      license_service=None, update_service=None):
    if settings is None:
        raise InProcessOnly("create_app() without Settings would use the real data folder")
    callbacks = CallbackServer()
    seams = {"token": token, "secret_key": secret_key, "admin_origin": admin_origin,
             "license": seam_translate.license_seam(license_service, callbacks),
             "updates": seam_translate.update_seam(update_service, callbacks),
             "faults": seam_translate.fault_seams(PENDING_FAULTS, callbacks)}
    engine = launch.EngineProcess(launch.engine_name(), settings.directory, settings.native_executable,
                                  settings.frontend, seams)
    try:
        engine.start()
    except BaseException:
        callbacks.close()
        raise
    config = {"LAUNCH_TOKEN": token, "SECRET_KEY": secret_key,
              "ADMIN_ORIGIN": admin_origin.rstrip("/") if admin_origin else None}
    app = RemoteApp(engine, callbacks, config)
    LIVE.append(app)
    return app
