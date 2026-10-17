"""Launch the local browser application and ensure worker cleanup on exit."""
import argparse
import atexit
import json
import os
from pathlib import Path
import secrets
import socket
import sys
import time
from urllib.parse import urlsplit
from studio_app.settings import Settings
from studio_app.launcher.directory_lease import DirectoryLease
from studio_app.launcher.windowed import attach_parent_terminal, configure_logging, headless, tell_user

# Heavy modules (Flask app, Waitress, reloader) are imported only once this launch owns
# the data folder, so starting ScadaBench again while it runs answers almost instantly.


def main():
    if len(sys.argv) > 1 and sys.argv[1] == "--network-helper":
        from studio_app.networking.helper.runner import main as network_helper
        raise SystemExit(network_helper(sys.argv[2:]))
    attach_parent_terminal()
    parser = argparse.ArgumentParser(description="ScadaBench")
    parser.add_argument("--port", type=int, default=8765)
    parser.add_argument("--no-browser", action="store_true",
                        help="Packaged app: do not open the app window (development never opens one)")
    parser.add_argument("--data-dir", type=Path)
    parser.add_argument("--self-test", action="store_true", help="Check bundled schemas and native MMS using temporary data, then exit")
    parser.add_argument("--dev", action="store_true", help="Reload the backend on source or worker changes")
    parser.add_argument("--dev-token", default=os.environ.get("STUDIO_DEV_TOKEN", "dev"),
                        help="Fixed session token used by --dev so the browser stays authenticated across reloads")
    parser.add_argument("--frontend-url", default=os.environ.get("STUDIO_FRONTEND_URL", "http://127.0.0.1:5173"),
                        help="Vite dev server address printed by --dev")
    parser.add_argument("--native-exe", type=Path, help="Use an explicit native IED worker executable")
    args = parser.parse_args()
    if args.dev and getattr(sys, "frozen", False):
        # Developer mode uses a fixed session secret/token and a source reloader; the
        # installed app must never run it. Dev is available only from source (run_dev.py).
        raise SystemExit("Developer mode (--dev) is not available in the installed ScadaBench. "
                         "Run it from source with studio/tools/run_dev.py instead.")
    if getattr(sys, "frozen", False):
        from studio_app.launcher.legacy_folder import migrate
        migrate(os.environ.get("LOCALAPPDATA"))
        _announce_signed_module()
    settings = Settings.defaults()
    if args.data_dir:
        settings.directory = args.data_dir
    if args.native_exe:
        settings.native_executable = args.native_exe.resolve()
    if args.self_test:
        from studio_app.launcher.self_test import check
        print(json.dumps(check(settings)), flush=True)
        return
    configure_logging(settings.directory)
    if args.dev:
        from werkzeug._reloader import run_with_reloader
        watched = [settings.native_executable] if settings.native_executable.exists() else []
        run_with_reloader(lambda: serve(settings, args, dev=True), extra_files=watched, interval=0.5)
    else:
        try:
            serve(settings, args)
        except Exception as error:
            if not headless():
                raise
            import logging
            logging.getLogger(__name__).exception("ScadaBench could not start")
            tell_user(f"ScadaBench could not start: {error}\n\nDetails are in the logs folder of "
                      f"{settings.directory}.", error=True)
            raise SystemExit(1) from error


def _announce_signed_module():
    """Point the native workers at this exact compiled module, but only when a signature
    was shipped beside it. The workers then verify it (integrity.cpp verify_module). An
    unsigned build sets nothing here, so it can never lock itself out by accident."""
    import studio_app
    module = getattr(studio_app, "__file__", "") or ""
    if module.endswith(".pyd") and os.path.exists(module + ".sig"):
        os.environ["STUDIO_MODULE_PATH"] = module
        os.environ["STUDIO_MODULE_SIG"] = module + ".sig"


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


def write_launcher_record(directory, url, window_pid=None):
    record = {"url": url, **({"window_pid": window_pid} if window_pid else {})}
    (directory / "launcher.json").write_text(json.dumps(record), encoding="utf-8")


def show_running_instance(directory, error):
    """Another ScadaBench owns this data folder: bring its window forward instead of failing.

    This freshly started process may take the foreground, so it focuses the recorded
    window itself; otherwise the running instance is asked to show or reopen it.
    """
    from studio_app.launcher.single_instance import request_show
    from studio_app.launcher.window_focus import focus_process_window
    try:
        window_pid = json.loads((directory / "launcher.json").read_text(encoding="utf-8")).get("window_pid")
    except (OSError, ValueError, AttributeError):
        window_pid = None
    if (window_pid and focus_process_window(window_pid)) or request_show(directory):
        if not headless():
            print("ScadaBench is already running; its window was brought to the front.", file=sys.stderr, flush=True)
        return 0
    tell_user(str(error), error=True)
    return 1


def start_window(target, url, settings, args):
    """Packaged launches open the app window; every non-dev launch answers 'show yourself'."""
    from studio_app.launcher.single_instance import ShowListener
    window = None
    if getattr(sys, "frozen", False) and not args.no_browser:
        import _thread
        from studio_app.launcher.app_window import AppWindow, find_browser
        window = AppWindow(target + ("&window=app" if find_browser() else ""), settings.directory,
                           _thread.interrupt_main).open()
        if window.process is not None:
            write_launcher_record(settings.directory, url, window.process.pid)
    return ShowListener(settings.directory, window.show if window else lambda: None).start()


def serve(settings, args, dev=False):
    lease = DirectoryLease(settings.directory)
    try:
        lease.__enter__()
    except RuntimeError as error:
        if dev:
            raise
        raise SystemExit(show_running_instance(settings.directory, error)) from None
    try:
        _serve(settings, args, dev)
    finally:
        lease.__exit__()


def _serve(settings, args, dev):
    from waitress import create_server
    from studio_app.app_factory import create_app
    listener = None
    token = args.dev_token if dev else secrets.token_urlsafe(32)
    app = create_app(settings, token, secret_key="studio-dev-secret" if dev else None,
                     admin_origin=args.frontend_url if dev else None)
    try:
        server = create_server(app, host="127.0.0.1", port=args.port, threads=16)
    except OSError:
        if dev:
            raise SystemExit(f"Port {args.port} is busy. Stop the other ScadaBench backend before starting dev mode.")
        server = create_server(app, host="127.0.0.1", port=0, threads=16)
    url = f"http://127.0.0.1:{server.effective_port}"
    settings.directory.mkdir(parents=True, exist_ok=True)
    write_launcher_record(settings.directory, url)
    print(f"ScadaBench: {url}", flush=True)
    if dev:
        atexit.register(app.extensions["runtime"].close)
        if not frontend_ready(args.frontend_url):
            print(f"Vite is not reachable at {args.frontend_url}; start it and open the printed address.", flush=True)
        target = f"{args.frontend_url}/#session={token}"
        print(f"Vite frontend: {target}", flush=True)
        print("Open the address above in your browser (ScadaBench never opens tabs automatically).", flush=True)
    else:
        target = url + "#session=" + token
        print(f"Open in your browser: {target}", flush=True)
    try:
        # Closing the app window interrupts the main thread, possibly while it is still
        # starting, so the window opens inside the same cleanup scope as the server.
        if not dev:
            listener = start_window(target, url, settings, args)
        server.run()
    except KeyboardInterrupt:
        pass
    finally:
        if listener:
            listener.close()
        app.extensions["runtime"].close()
        server.close()


if __name__ == "__main__":
    main()
