"""Serve the frozen Python app (create_app + Waitress) for the Go parity test.

Usage: python -I flask_server.py --backend <studio/backend> --port N --data-dir D
       --token T --secret S [--admin-origin URL] [--frontend DIR]
       [--license none|unlicensed] [--startup ok|timeout|error]
Prints "ScadaBench: http://127.0.0.1:<port>" when ready, like the launcher. License and
update services are offline fakes; --startup replaces the background recovery result.
"""
import argparse
import os
import pathlib
import sys


class FakeLicense:
    def __init__(self, enforce):
        self.enforce = enforce

    def is_licensed(self):
        return False

    def start_background_verify(self):
        pass

    def server_url(self):
        return "https://license.invalid"


class FakeUpdates:
    def start_background(self):
        pass


class FakeRecovery:
    def __init__(self, ready, error):
        self.ready, self.error = ready, error

    def wait(self, timeout=None):
        return self.ready


def main():
    parser = argparse.ArgumentParser()
    for name in ("--backend", "--port", "--data-dir", "--token", "--secret"):
        parser.add_argument(name, required=True)
    parser.add_argument("--admin-origin")
    parser.add_argument("--frontend")
    parser.add_argument("--license", default="none")
    parser.add_argument("--startup", default="ok")
    args = parser.parse_args()
    os.environ["SCADABENCH_UPDATE_CHECK"] = "off"
    sys.path.insert(0, args.backend)
    from waitress import create_server
    from studio_app.settings import Settings
    from studio_app.app_factory import create_app
    settings = Settings.defaults()
    settings.directory = pathlib.Path(args.data_dir)
    if args.frontend:
        settings.frontend = pathlib.Path(args.frontend)
    app = create_app(settings, args.token, secret_key=args.secret, admin_origin=args.admin_origin,
                     license_service=FakeLicense(args.license == "unlicensed"), update_service=FakeUpdates())
    if args.startup == "timeout":
        app.extensions["startup"] = FakeRecovery(False, None)
    elif args.startup == "error":
        app.extensions["startup"] = FakeRecovery(True, "the autosave is a link")
    server = create_server(app, host="127.0.0.1", port=int(args.port), threads=16)
    print(f"ScadaBench: http://127.0.0.1:{server.effective_port}", flush=True)
    server.run()


if __name__ == "__main__":
    main()
