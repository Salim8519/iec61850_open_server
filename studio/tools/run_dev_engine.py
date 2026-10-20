"""Development launcher with the engine switch (ADR-0074, ADR-0083): Go (default) or Python.

Usage: python studio/tools/run_dev_engine.py [--engine python|go] [--no-browser] [--print-command]
Without --engine the SCADABENCH_ENGINE environment variable decides; unset means Go, the
engine the released app ships (0.1.15), so development behaves like the product.
Python runs exactly studio/tools/run_dev.py. Go first rebuilds studio_server.exe from
studio/engine (go build is incremental, so dev never runs an older engine than its source),
then starts Vite and the Go server with the same arguments, port, data folder, dev token and
cookie secret as the Python backend, so an open tab stays signed in across a switch.
--print-command shows what would run and starts nothing.
"""
from pathlib import Path
import os
import subprocess
import sys

TOOLS = Path(__file__).resolve().parent
STUDIO = TOOLS.parent
ENGINE = STUDIO / "engine"
SERVER = STUDIO / "build/engine/studio_server.exe"
ENGINES = ("python", "go")

sys.path.insert(0, str(TOOLS))
import run_dev  # noqa: E402  (shares the process helpers; run_dev itself is unchanged)
from build_goose_engine import go_command  # noqa: E402


def choose_engine(arguments):
    """--engine wins over SCADABENCH_ENGINE; Go is the default (ADR-0083); unknown names stop."""
    remaining, choice, source = [], None, "--engine"
    iterator = iter(arguments)
    for argument in iterator:
        if argument == "--engine":
            choice = next(iterator, "")
        elif argument.startswith("--engine="):
            choice = argument.split("=", 1)[1]
        else:
            remaining.append(argument)
    if choice is None:
        choice, source = os.environ.get("SCADABENCH_ENGINE", ""), "SCADABENCH_ENGINE"
    engine = choice.strip().lower() or "go"
    if engine not in ENGINES:
        raise SystemExit(f'{source} must be "python" or "go", not "{choice}".')
    return engine, remaining


def build_server():
    environment = {**os.environ, "CGO_ENABLED": "0", "GOOS": "windows", "GOARCH": "amd64"}
    SERVER.parent.mkdir(parents=True, exist_ok=True)
    completed = subprocess.run([go_command(), "build", "-o", str(SERVER), "./cmd/studio_server"],
                               cwd=ENGINE, env=environment)
    if completed.returncode:
        raise SystemExit(f"Building the Go engine failed with exit code {completed.returncode}.")


def server_command(native, frontend_url, arguments):
    return [str(SERVER), "--dev", "--data-dir", str(STUDIO / "data"), "--native-exe", str(native),
            "--frontend-url", frontend_url, *arguments]


def main_go(arguments, print_only):
    if any(argument != "--no-browser" for argument in arguments):
        raise SystemExit("start-studio.cmd accepts only the optional --no-browser and --engine flags.")
    native = run_dev.require(STUDIO / "native/build/Release/studio_ied.exe",
                             "Build studio/native/build/Release/studio_ied.exe first.")
    if print_only:
        print("go build -o", SERVER, "./cmd/studio_server")
        print(" ".join(server_command(native, "http://127.0.0.1:<vite port>", arguments)))
        return 0
    node = run_dev.require(run_dev.shutil.which("node.exe") or run_dev.shutil.which("node"),
                           "Install Node.js before starting Studio development.")
    vite = run_dev.require(STUDIO / "frontend/node_modules/vite/bin/vite.js",
                           "Install the frontend dependencies before starting Studio development.")
    token = os.environ.get("STUDIO_DEV_TOKEN", "dev")
    try:
        run_dev.require_free_port(8765, "another application")
    except SystemExit:
        current = run_dev.existing_frontend(5173, 5199, token)
        if not current:
            raise
        print(f"ScadaBench development mode is already running: {current}", flush=True)
        print(f"Open in your browser: {current}/#session={token}", flush=True)
        return 0
    subprocess.run([sys.executable, str(TOOLS / "build_goose_engine.py"), "--if-stale", "--skip-tests"], check=True)
    subprocess.run([sys.executable, str(TOOLS / "build_snmp_engine.py"), "--if-stale"], check=True)
    build_server()
    frontend_port = run_dev.available_port(5173, 5199)
    frontend_url = f"http://127.0.0.1:{frontend_port}"
    print("Starting ScadaBench development mode with the Go engine (as released). Frontend changes "
          "reload by themselves; restart this window after Go changes.", flush=True)
    ui = run_dev.start([str(node), str(vite), "--host", "127.0.0.1", "--port", str(frontend_port),
                        "--logLevel", "warn"], cwd=STUDIO / "frontend", env=os.environ.copy())
    api = run_dev.start(server_command(native, frontend_url, arguments), cwd=STUDIO.parent, env=os.environ.copy())
    try:
        return run_dev_wait(api, ui)
    except KeyboardInterrupt:
        print("Stopping ScadaBench...", flush=True)
        return 0
    finally:
        run_dev.stop_tree(api)
        run_dev.stop_tree(ui)


def run_dev_wait(api, ui):
    while True:
        api_code, ui_code = api.poll(), ui.poll()
        if api_code is not None:
            if api_code:
                print(f"Studio backend stopped with exit code {api_code}.", flush=True)
            return api_code
        if ui_code is not None:
            print(f"Vite stopped with exit code {ui_code}; stopping the backend.", flush=True)
            return ui_code or 1
        run_dev.time.sleep(0.25)


def fold_before_python(data, print_only):
    """A Go run that ended without its final snapshot leaves autosave.journal; fold it into
    autosave.subsim (studio_server --fold-journal) so Python opens exactly what Go saved."""
    if not (data / "autosave.journal").exists():
        return
    if print_only:
        print(SERVER, "--fold-journal", data)
        return
    build_server()
    completed = subprocess.run([str(SERVER), "--fold-journal", str(data)])
    if completed.returncode:
        raise SystemExit("The Go engine's saved changes could not be handed to the Python engine; "
                         "both files were kept. Start with --engine go, or inspect the data folder.")


def main(arguments):
    print_only = "--print-command" in arguments
    arguments = [argument for argument in arguments if argument != "--print-command"]
    engine, remaining = choose_engine(arguments)
    if engine == "python":
        fold_before_python(STUDIO / "data", print_only)
        if print_only:
            print("python studio/tools/run_dev.py", *remaining)
            return 0
        return run_dev.main(remaining)
    return main_go(remaining, print_only)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
