"""Try the whole licensing flow on this PC without building an EXE.

    python studio/tools/license_sandbox.py                 start (Ctrl+C stops everything)
    python studio/tools/license_sandbox.py --days-ahead 8  pretend it is 8 days later (trial expiry)
    python studio/tools/license_sandbox.py --reset         forget all sandbox keys, trials and licenses

Starts the real Go license server on 127.0.0.1 and a Studio backend with packaged-mode
licensing that trusts only that server's key. Everything lives in
studio/build/license-sandbox; your real data, license folder and release keys are not
touched. Native workers are the normal development build (not license-enforced).
PayPal: set PAYPAL_CLIENT_ID / PAYPAL_CLIENT_SECRET (sandbox) before starting to try /buy.
"""
import argparse
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import sys
import time
from urllib.request import urlopen

STUDIO = Path(__file__).resolve().parents[1]
ROOT = STUDIO.parent
SANDBOX = STUDIO / "build/license-sandbox"
SERVER_DATA = SANDBOX / "server-data"
SERVER_EXE = SANDBOX / "license-server.exe"
sys.path.insert(0, str(STUDIO / "backend"))


def free_port(preferred):
    for port in (preferred, 0):
        try:
            with socket.socket() as listener:
                listener.bind(("127.0.0.1", port))
                return listener.getsockname()[1]
        except OSError:
            continue
    raise SystemExit("No free local port")


def build_server():
    if shutil.which("go") is None:
        raise SystemExit("Go is not installed (https://go.dev/dl). It is needed to build the license server.")
    print("Building the license server...", flush=True)
    subprocess.run(["go", "build", "-o", str(SERVER_EXE), "./cmd/license-server"], cwd=ROOT / "license-server", check=True)


def server_env(port, trial_days):
    return {**os.environ, "LICENSE_DATA_DIR": str(SERVER_DATA), "LICENSE_LISTEN_ADDR": f"127.0.0.1:{port}",
            "LICENSE_PUBLIC_URL": f"http://127.0.0.1:{port}", "LICENSE_TRIAL_DAYS": str(trial_days)}


def public_key(env):
    if not (SERVER_DATA / "signing.key").exists():
        subprocess.run([str(SERVER_EXE), "keygen"], env=env, check=True, capture_output=True)
    status = subprocess.run([str(SERVER_EXE), "status"], env=env, check=True, capture_output=True, text=True).stdout
    return status.split("--public-key", 1)[1].split()[0]


def wait_healthy(url, process):
    deadline = time.monotonic() + 15
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise SystemExit("The license server stopped during startup.")
        try:
            with urlopen(url + "/api/v1/health", timeout=1) as response:
                return json.loads(response.read())
        except OSError:
            time.sleep(0.2)
    raise SystemExit("The license server did not answer within 15 s.")


def studio_app(server_url, public_hex, days_ahead):
    from studio_app.app_factory import create_app
    from studio_app.licensing.release_key import key_id
    from studio_app.licensing.service import LicenseService
    from studio_app.settings import Settings
    key = bytes.fromhex(public_hex)
    settings = Settings(SANDBOX / "studio-data", STUDIO / "native/build/Release/studio_ied.exe", STUDIO / "frontend/dist")
    offset = days_ahead * 86400
    # allow_test_server is the explicit test-only seam: the product never talks to a server on this PC.
    service = LicenseService(SANDBOX / "studio-license", settings.directory, enforce=True,
                             clock=lambda: time.time() + offset, keys={key_id(key): key}, allow_test_server=True)
    service.set_server_url(server_url)
    token = secrets.token_urlsafe(16)
    return create_app(settings, token, license_service=service), token


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--days-ahead", type=int, default=0, help="shift Studio's clock forward (test expiry)")
    parser.add_argument("--trial-days", type=int, default=7)
    parser.add_argument("--reset", action="store_true", help="delete the sandbox folder first")
    args = parser.parse_args(argv)
    if args.reset and SANDBOX.exists():
        shutil.rmtree(SANDBOX)
        print(f"Sandbox cleared: {SANDBOX}")
    if not (STUDIO / "frontend/dist/index.html").exists():
        raise SystemExit("Build the UI first: cd studio/frontend && npm run build")
    SANDBOX.mkdir(parents=True, exist_ok=True)
    build_server()
    port = free_port(8090)
    if port != 8090:
        print(f"Port 8090 is busy; using {port}. A --sandbox EXE expects 8090, so free it for EXE tests.")
    env = server_env(port, args.trial_days)
    key = public_key(env)
    server_url = f"http://127.0.0.1:{port}"
    log = open(SANDBOX / "server-console.log", "ab")
    server = subprocess.Popen([str(SERVER_EXE), "serve"], env=env, stdout=log, stderr=log)
    try:
        health = wait_healthy(server_url, server)
        app, token = studio_app(server_url, key, args.days_ahead)
        studio_port = free_port(8798)
        issue = f'"{SERVER_EXE}" issue --data "{SERVER_DATA}" --email you@example.com'
        print(f"""
License server : {server_url}   (key id {health['kid']}, PayPal {'on' if health['checkout'] else 'off'})
  buy page     : {server_url}/buy
  offline page : {server_url}/activate
Studio (packaged licensing{f', clock +{args.days_ahead} days' if args.days_ahead else ''}):
  open         : http://127.0.0.1:{studio_port}/#session={token}

Make a lifetime key (second terminal):
  {issue}
Ctrl+C stops both.
""", flush=True)
        from waitress import serve
        serve(app, host="127.0.0.1", port=studio_port, threads=8)
    except KeyboardInterrupt:
        pass
    finally:
        server.terminate()
        server.wait(timeout=10)
        log.close()
        print("Sandbox stopped.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
