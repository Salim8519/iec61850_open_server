"""Own the complete source-development process tree behind one launcher.

Studio never opens browser tabs automatically; print addresses instead.
"""
from pathlib import Path
import json
import os
import shutil
import socket
import subprocess
import sys
import time
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[2]
STUDIO = ROOT / "studio"


def require(path, message):
    if not path:
        raise SystemExit(message)
    candidate = Path(path)
    if not candidate.exists():
        raise SystemExit(message)
    return candidate


def stop_tree(process):
    """Stop only a process tree created by this launcher."""
    if process.poll() is not None:
        return
    subprocess.run(["taskkill", "/PID", str(process.pid), "/T", "/F"],
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        process.kill()


def start(command, **options):
    return subprocess.Popen(command, creationflags=subprocess.CREATE_NEW_PROCESS_GROUP,
                            **options)


def require_free_port(port, service):
    try:
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", port))
    except OSError:
        raise SystemExit(
            f"Cannot start: port {port} is already used by {service}.\n"
            "An existing Studio development session is reused automatically; "
            "this listener could not be verified as Studio.") from None


def accepts_dev_session(frontend_url, token):
    request = Request(frontend_url + "/api/v1/session", method="POST",
                      data=json.dumps({"token": token}).encode("utf-8"),
                      headers={"Content-Type": "application/json", "Origin": frontend_url})
    try:
        with urlopen(request, timeout=0.5) as response:
            return response.status == 200
    except (HTTPError, URLError, OSError, TimeoutError):
        return False


def existing_frontend(first, last, token):
    for port in range(first, last + 1):
        url = f"http://127.0.0.1:{port}"
        if accepts_dev_session(url, token):
            return url
    return None


def available_port(first, last):
    for port in range(first, last + 1):
        try:
            with socket.socket() as listener:
                listener.bind(("127.0.0.1", port))
            return port
        except OSError:
            continue
    raise SystemExit(f"Cannot start: every development frontend port from {first} to {last} is busy.")


def main(arguments):
    if any(argument != "--no-browser" for argument in arguments):
        raise SystemExit("start-studio.cmd accepts only the optional --no-browser flag.")
    python = require(STUDIO / ".venv/Scripts/python.exe",
                     "Create studio/.venv before starting Studio development.")
    node = require(shutil.which("node.exe") or shutil.which("node"),
                   "Install Node.js before starting Studio development.")
    vite = require(STUDIO / "frontend/node_modules/vite/bin/vite.js",
                   "Install the frontend dependencies before starting Studio development.")
    native = require(STUDIO / "native/build/Release/studio_ied.exe",
                     "Build studio/native/build/Release/studio_ied.exe first.")
    environment = os.environ.copy()
    token = environment.get("STUDIO_DEV_TOKEN", "dev")
    try:
        require_free_port(8765, "another application")
    except SystemExit:
        current = existing_frontend(5173, 5199, token)
        if not current:
            raise
        target = f"{current}/#session={token}"
        print(f"ScadaBench development mode is already running: {current}", flush=True)
        print(f"Open in your browser: {target}", flush=True)
        return 0
    frontend_port = available_port(5173, 5199)
    frontend_url = f"http://127.0.0.1:{frontend_port}"
    backend = str(STUDIO / "backend")
    environment["PYTHONPATH"] = backend + (os.pathsep + environment["PYTHONPATH"]
                                              if environment.get("PYTHONPATH") else "")
    print("Starting ScadaBench development mode (frontend and backend hot reload).",
          flush=True)
    # Vite's own banner prints an address without the session token; show only ours.
    ui = start([str(node), str(vite), "--host", "127.0.0.1", "--port", str(frontend_port),
                "--logLevel", "warn"],
               cwd=STUDIO / "frontend", env=environment)
    api = start([str(python), "-m", "studio_app.launcher.entrypoint", "--dev",
                 "--data-dir", str(STUDIO / "data"), "--native-exe", str(native),
                 "--frontend-url", frontend_url, *arguments],
                cwd=ROOT, env=environment)
    try:
        while True:
            api_code, ui_code = api.poll(), ui.poll()
            if api_code is not None:
                if api_code:
                    print(f"Studio backend stopped with exit code {api_code}.", flush=True)
                return api_code
            if ui_code is not None:
                print(f"Vite stopped with exit code {ui_code}; stopping the backend.", flush=True)
                return ui_code or 1
            time.sleep(0.25)
    except KeyboardInterrupt:
        print("Stopping ScadaBench...", flush=True)
        return 0
    finally:
        stop_tree(api)
        stop_tree(ui)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
