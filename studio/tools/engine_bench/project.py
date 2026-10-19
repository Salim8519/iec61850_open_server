"""Build a benchmark project over HTTP and move its IEDs onto the isolated loopback range.

IEDs are imported one SCL file each (preview, import). Their addresses are then rewritten
in the downloaded project archive (127.66.<block>.<n>, high ports) and the archive is opened
again through POST /project/open, so no IED ever binds the owner's 127.50.0.x:102.
"""
import io
import json
import time
import zipfile

from engine_bench.fixtures import make_scl

RANGE = "127.66"
BASE_PORT = 18700


def address(index, block=0):
    return f"{RANGE}.{block}.{index + 2}"


def import_ieds(engine, count, points, prefix="Bench"):
    timings = []
    for index in range(count):
        name = f"{prefix}{index:02d}"
        data = make_scl(name, points)
        started = time.perf_counter()
        engine.upload("/api/v1/scl/preview", "file", f"{name}.scd", data)
        engine.upload("/api/v1/scl/import", "file", f"{name}.scd", data, {"selected_names": json.dumps([name])})
        timings.append(time.perf_counter() - started)
    return timings


def relocate(engine, block=0):
    """Rewrite ip/port of every IED onto 127.66.<block>.x and open the archive again."""
    status, archive, _ = engine.request("GET", "/api/v1/project/download")
    if status != 200:
        raise RuntimeError(f"download failed: {status}")
    with zipfile.ZipFile(io.BytesIO(archive)) as source:
        document = json.loads(source.read("project.json"))
    ieds = document["ieds"]
    for index, ied in enumerate(ieds):
        ied["ip"], ied["port"] = address(index, block), BASE_PORT
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as target:
        target.writestr("project.json", json.dumps(document))
    started = time.perf_counter()
    engine.upload("/api/v1/project/open", "file", "bench.subsim", stream.getvalue())
    return time.perf_counter() - started


def project(engine):
    return engine.json("GET", "/api/v1/project")


def wait_state(engine, state, timeout=120):
    deadline = time.perf_counter() + timeout
    while time.perf_counter() < deadline:
        current = project(engine)
        if current.get("runtime") == state:
            return current
        time.sleep(0.05)
    raise TimeoutError(f"runtime did not reach {state!r}")
