"""Launch the local browser application and ensure worker cleanup on exit."""
import argparse
import atexit
import json
import logging
import os
from pathlib import Path
import secrets
import socket
import sys
import time
from urllib.parse import urlsplit
from waitress import create_server
from werkzeug._reloader import run_with_reloader
from studio_app.app_factory import create_app
from studio_app.settings import Settings
from studio_app.launcher.directory_lease import DirectoryLease


def main():
    if len(sys.argv) > 1 and sys.argv[1] == "--network-helper":
        from studio_app.networking.helper.runner import main as network_helper
        raise SystemExit(network_helper(sys.argv[2:]))
    parser = argparse.ArgumentParser(description="Substation Studio")
    parser.add_argument("--port", type=int, default=8765)
    parser.add_argument("--no-browser", action="store_true",
                        help="Accepted for compatibility; Studio never opens a browser automatically")
    parser.add_argument("--data-dir", type=Path)
    parser.add_argument("--self-test", action="store_true", help="Check bundled schemas and native MMS using temporary data, then exit")
    parser.add_argument("--dev", action="store_true", help="Reload the backend on source or worker changes")
    parser.add_argument("--dev-token", default=os.environ.get("STUDIO_DEV_TOKEN", "dev"),
                        help="Fixed session token used by --dev so the browser stays authenticated across reloads")
    parser.add_argument("--frontend-url", default=os.environ.get("STUDIO_FRONTEND_URL", "http://127.0.0.1:5173"),
                        help="Vite dev server address printed by --dev")
    parser.add_argument("--native-exe", type=Path, help="Use an explicit native IED worker executable")
    args = parser.parse_args()
    settings = Settings.defaults()
    if args.data_dir:
        settings.directory = args.data_dir
    if args.native_exe:
        settings.native_executable = args.native_exe.resolve()
    if args.self_test:
        from studio_app.launcher.self_test import check
        print(json.dumps(check(settings)), flush=True)
        return
    logging.basicConfig(level=logging.INFO)
    if args.dev:
        watched = [settings.native_executable] if settings.native_executable.exists() else []
        run_with_reloader(lambda: serve(settings, args, dev=True), extra_files=watched, interval=0.5)
    else:
        serve(settings, args)


def frontend_ready(url, timeout=15):
    parts = urlsplit(url)
    port = parts.port or (443 if parts.scheme == "https" else 80)
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            with socket.create_connection((parts.hostname, port), timeout=1):
                return True
        except OSError:
            time.sleep(0.25)
    return False


def serve(settings, args, dev=False):
    with DirectoryLease(settings.directory):
        token = args.dev_token if dev else secrets.token_urlsafe(32)
        app = create_app(settings, token, secret_key="studio-dev-secret" if dev else None,
                         admin_origin=args.frontend_url if dev else None)
        try:
            server = create_server(app, host="127.0.0.1", port=args.port, threads=16)
        except OSError:
            if dev:
                raise SystemExit(f"Port {args.port} is busy. Stop the other Studio backend before starting dev mode.")
            server = create_server(app, host="127.0.0.1", port=0, threads=16)
        url = f"http://127.0.0.1:{server.effective_port}"
        settings.directory.mkdir(parents=True, exist_ok=True)
        (settings.directory / "launcher.json").write_text(json.dumps({"url": url}), encoding="utf-8")
        print(f"Substation Studio: {url}", flush=True)
        if dev:
            atexit.register(app.extensions["runtime"].close)
            if not frontend_ready(args.frontend_url):
                print(f"Vite is not reachable at {args.frontend_url}; start it and open the printed address.", flush=True)
            target = f"{args.frontend_url}/#session={token}"
            print(f"Vite frontend: {target}", flush=True)
            print("Open the address above in your browser (Studio never opens tabs automatically).", flush=True)
        else:
            target = url + "#session=" + token
            print(f"Open in your browser: {target}", flush=True)
        try:
            server.run()
        except KeyboardInterrupt:
            pass
        finally:
            app.extensions["runtime"].close()
            server.close()


if __name__ == "__main__":
    main()
