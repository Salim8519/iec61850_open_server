"""python-http engine: the real create_app served by Waitress exactly as entrypoint._serve.

Test-only (`python -m parity.server`, cwd studio/tests). It reads the seams file, builds the
same Settings the test asked for, applies the seams, calls the unmodified create_app, wraps
it with the parity inspect API and serves it on 127.0.0.1 with 16 threads. No launcher
lease, no logging setup and no launcher.json: in-process tests never had them either, so
the data folder ends up byte-identical to the `python` engine.
"""
import argparse
import json
import os
from pathlib import Path
import threading


def parse(argv=None):
    parser = argparse.ArgumentParser(description="ScadaBench parity server (python-http)")
    parser.add_argument("--port", type=int, default=0)
    parser.add_argument("--no-browser", action="store_true")
    parser.add_argument("--data-dir", type=Path, required=True)
    parser.add_argument("--native-exe", type=Path, required=True)
    parser.add_argument("--frontend", type=Path, required=True)
    parser.add_argument("--parity-seams", type=Path, required=True)
    return parser.parse_args(argv)


def build_app(args, seams):
    from studio_app.settings import Settings
    from studio_app.app_factory import create_app
    from parity import server_seams
    settings = Settings(args.data_dir, args.native_exe, args.frontend)
    server_seams.apply_faults(seams.get("faults") or {})
    return create_app(settings, seams.get("token"), secret_key=seams.get("secret_key"),
                      admin_origin=seams.get("admin_origin"),
                      license_service=server_seams.license_service(seams.get("license")),
                      update_service=server_seams.update_service(seams.get("updates")))


def main(argv=None):
    args = parse(argv)
    seams = json.loads(args.parity_seams.read_text(encoding="utf-8"))
    from waitress import create_server
    from parity.inspect_api import InspectApp
    app = build_app(args, seams)
    stopping = threading.Event()
    wrapped = InspectApp(app, seams["parity_key"], stopping)
    server = create_server(wrapped, host="127.0.0.1", port=args.port, threads=16)
    print(f"ScadaBench: http://127.0.0.1:{server.effective_port}", flush=True)
    threading.Thread(target=_exit_when_stopped, args=(stopping, app, server), daemon=True).start()
    try:
        server.run()
    except KeyboardInterrupt:
        pass
    finally:
        app.extensions["runtime"].close()


def _exit_when_stopped(stopping, app, server):
    """`POST _parity/shutdown` = the launcher's exit path: runtime.close() then exit."""
    stopping.wait()
    try:
        app.extensions["runtime"].close()
        server.close()
    finally:
        os._exit(0)


if __name__ == "__main__":
    main()
