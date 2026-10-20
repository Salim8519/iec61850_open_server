"""Print the frozen Python app's routes as JSON [[method, rule], ...] (Flask url_map).

Usage: python -I url_map.py --backend <studio/backend> --data-dir D
HEAD and OPTIONS (added by Werkzeug) are left out. License and update services are offline
fakes; nothing listens on a port.
"""
import argparse
import json
import os
import pathlib
import sys


class FakeLicense:
    enforce = False

    def is_licensed(self):
        return True

    def start_background_verify(self):
        pass

    def server_url(self):
        return "https://license.invalid"


class FakeUpdates:
    def start_background(self):
        pass


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--backend", required=True)
    parser.add_argument("--data-dir", required=True)
    args = parser.parse_args()
    os.environ["SCADABENCH_UPDATE_CHECK"] = "off"
    sys.path.insert(0, args.backend)
    from studio_app.settings import Settings
    from studio_app.app_factory import create_app
    settings = Settings.defaults()
    settings.directory = pathlib.Path(args.data_dir)
    app = create_app(settings, "token", secret_key="secret", license_service=FakeLicense(),
                     update_service=FakeUpdates())
    routes = sorted({(method, rule.rule) for rule in app.url_map.iter_rules()
                     for method in rule.methods if method not in {"HEAD", "OPTIONS"}})
    print(json.dumps([list(route) for route in routes]), flush=True)
    os._exit(0)  # the recovery and gateway-resume threads are not joined


if __name__ == "__main__":
    main()
