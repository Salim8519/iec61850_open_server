"""Cross-check the Go store formats against the frozen Python backend.

  crosscheck.py generate <dir>   write Python-made goldens (see GOLDENS below)
  crosscheck.py verify <dir>     decode every Go-written <name>.subsim like Python does

Run with studio/.venv/Scripts/python.exe; studio/backend is put on sys.path from this
script's location. Only synthetic or public sample data is used.
"""
import io
import json
import sys
import zipfile
from pathlib import Path

BACKEND = Path(__file__).resolve().parents[4] / "backend"
sys.path.insert(0, str(BACKEND))

from studio_app.contracts.errors import StudioError  # noqa: E402
from studio_app.projects import archive  # noqa: E402
from studio_app.projects.preferences import PreferenceStore  # noqa: E402
from studio_app.tools.tool_memory import ToolMemory, _clean  # noqa: E402

INDENT_VALUES = [
    {}, [], 0, -1, 2 ** 70, 1.0, 1e16, 1e-05, 0.1 + 0.2, -0.0, float("nan"), float("inf"),
    float("-inf"), True, False, None, "", "é\U0001F600\x7f\x00\n\t\"\\/<&>",
    {"a": {}, "b": [], "c": [[], {}], "d": {"e": [1, [2, [3]]]}},
    [[1, 2], {"x": None}, "s", 1.5],
    {"tour_seen_version": 1, "sld_goose_layer": 1, "sld_readers_layer": 0,
     "list_filters": {"ied-list": {"query": "é", "live": True}}},
    {"éclé": " ", "k\U0001F600": [123456789012345678901234567890, -1e-300]},
]

SCL = "<SCL version=\"2007\">\r\n  <!-- é \U0001F600 \x7f -->\r\n  <IED name=\"IED1\"/>\r\n</SCL>\r\n"


def python_project():
    node = {"id": "cb1", "type": "equipment", "position": {"x": 360.0, "y": 1e-05},
            "data": {"label": "CB é <&>", "rotation": 0, "push_value": 0.1 + 0.2}}
    ied = {"id": "0" * 31 + "1", "name": "IED1", "filename": "a.scd", "ip": "127.50.0.2", "port": 102,
           "role": "simulated", "live": {"ip": "", "port": 102}, "source": SCL}
    second = dict(ied, id="0" * 31 + "2", name="IED2", source=SCL.replace("IED1", "IED2"))
    return {"id": "0123456789abcdef0123456789abcdef", "name": "Démo ☃", "ieds": [ied, second],
            "diagram": {"nodes": [node], "edges": []}, "sources": [], "scenarios": [],
            "test_panel": [], "network": {"mode": "local", "nominal_hz": 50.0},
            "gateway": {"points": [{"scale": 1e16, "big": 2 ** 70, "neg": -0.0}]}}


def zip_of(members):
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as target:
        for name, data in members:  # a fixed time keeps the recorded vectors stable
            info = zipfile.ZipInfo(name, date_time=(2026, 1, 1, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            target.writestr(info, data)
    return stream.getvalue()


DECODE_PAYLOADS = [
    b"not a zip",
    zip_of([("project.json", b'{"format": 1, "a": "\xff"}')]),
    zip_of([("project.json", b'{"format": 1, "a": "\xe2\x82"}')]),
    zip_of([("project.json", b'{"format": 1, "a": "\xe2')]),
    zip_of([("project.json", b'\xef\xbb\xbf{"format": 1, "a": "\xc0"}')]),
    zip_of([("project.json", b'{"format": 1, "a": "\xed\xa0\x41"}')]),
    zip_of([("project.json", b'{"format": 1, "a": "\xe0\x80\x80"}')]),
    zip_of([("project.json", b'{"format": 1, "a": "\xf4\x90\x80\x80"}')]),
    zip_of([("project.json", b'{"format": 1, "a": "\xed\xa0\x80"}')]),
    zip_of([("project.json", '{"format": 1, "a": "é"}'.encode("utf-16"))]),
    zip_of([("project.json", '{"format": 1, "a": "\U0001F600"}'.encode("utf-16-be"))]),
    zip_of([("project.json", '{"format": 1}'.encode("utf-32-le"))]),
    zip_of([("project.json", '{"format": 1}'.encode("utf-16-le") + b"\x00")]),
    zip_of([("project.json", b'\xef\xbb\xbf{"format": 1}')]),
    zip_of([("project.json", b'\xef\xbb\xbf\xef\xbb\xbf{"format": 1}')]),
    zip_of([("project.json", b"[1, 2]")]),
    zip_of([("project.json", b'{"format": 1')]),
    zip_of([("project.json", b"{}"), ("x", b"{}")]),
    zip_of([("other.json", b"{}")]),
]

TOOL_INPUTS = [
    None, [], {"explorer": 5, "find": []},
    {"explorer": {"interval_ms": True, "port": False, "host": "h" * 65}},
    {"explorer": {"interval_ms": 60000, "port": 2 ** 70, "host": "h" * 64}},
    {"find": {"port": 1.5, "range": "r" * 64, "choice": "c" * 201}},
    {"find": {"port": True, "range": "r" * 65, "choice": "é" * 200}},
    {"explorer": {"watch_by_host": {"h": [
        {"ref": "r", "fc": "ST", "type": "t" * 40}, {"ref": "r", "fc": "ST", "type": 1e16},
        {"ref": "r", "fc": "ST", "type": "é" * 40}, {"ref": "r", "fc": "ST", "type": None},
        {"ref": "r", "fc": "ST", "type": [1, "it's", {"k": -0.0}, False]},
        {"ref": "x" * 201, "fc": "ST"}, {"ref": "r", "fc": "STXYZ"}, {"ref": "r", "fc": ""}, 5]}}},
    {"explorer": {"watch_by_host": {f"host{i}": [] for i in range(60)} | {"x" * 65: []}}},
]


def failure(payload):
    try:
        return {"ok": archive.decode(payload)}
    except StudioError as error:
        return {"kind": "StudioError", "message": str(error)}
    except ValueError as error:
        return {"kind": type(error).__name__, "message": str(error)}


def generate(out):
    (out / "indent").mkdir(parents=True, exist_ok=True)
    for index, value in enumerate(INDENT_VALUES):
        (out / "indent" / f"{index:02d}.json").write_bytes(json.dumps(value).encode())
        (out / "indent" / f"{index:02d}.golden").write_text(json.dumps(value, indent=2), encoding="utf-8")
    project = python_project()
    (out / "python_archive.subsim").write_bytes(archive.encode(project))
    compact = archive.compact(project)
    (out / "python_archive.json").write_bytes(json.dumps(compact, allow_nan=False).encode())
    vectors = [{"payload": payload.hex(), **failure(payload)} for payload in DECODE_PAYLOADS]
    (out / "decode_vectors.json").write_bytes(json.dumps(vectors, indent=1).encode())
    try:
        json.dumps({"a": [float("nan")]}, allow_nan=False)
    except ValueError as error:
        (out / "nan_message.txt").write_bytes(str(error).encode())
    PreferenceStore(out).update({"tour_seen_version": 1, "sld_goose_layer": 1,
                                 "list_filters": {"ied-list": {"query": "é", "live": True}}})
    memory = ToolMemory(out)
    memory.update("explorer", host="10.0.0.5", interval_ms=2000, watch_by_host={
        "10.0.0.5": [{"ref": "LD0/LLN0.Mod.stVal", "fc": "ST", "type": "Enum"}]})
    memory.update("find", range="10.0.0.0/24", port=True, choice="é")
    pairs = [[raw, _clean(raw)] for raw in TOOL_INPUTS]
    (out / "toolclean_vectors.json").write_bytes(json.dumps(pairs, indent=1).encode())


def verify(folder):
    errors = []
    names = sorted(folder.glob("*.subsim"))
    for path in names:
        payload = path.read_bytes()
        try:
            with zipfile.ZipFile(io.BytesIO(payload)) as source:
                if source.testzip() is not None:
                    errors.append(f"{path.name}: testzip failed")
                info = source.infolist()[0]
                header = (info.create_system, info.create_version, info.extract_version,
                          info.flag_bits, info.external_attr, info.extra, info.compress_type)
                if header != (0, 20, 20, 0, 0o600 << 16, b"", zipfile.ZIP_DEFLATED):
                    errors.append(f"{path.name}: header {header}")
                if payload[26:30] != len("project.json").to_bytes(2, "little") + b"\0\0":
                    errors.append(f"{path.name}: local header lengths")
                data = source.read("project.json")
            document = archive.decode(payload)
            if json.dumps(json.loads(data), allow_nan=False).encode() != data:
                errors.append(f"{path.name}: project.json is not json.dumps output")
            if json.dumps(archive.compact(document), allow_nan=False).encode() != data:
                errors.append(f"{path.name}: compact(decode()) differs")
            expected = path.with_suffix(".json")
            if expected.exists() and expected.read_bytes() != data:
                errors.append(f"{path.name}: differs from {expected.name}")
        except Exception as error:  # every failure is reported, none is fatal
            errors.append(f"{path.name}: {type(error).__name__}: {error}")
    print(json.dumps({"files": len(names), "errors": errors}))
    return 1 if errors or not names else 0


if __name__ == "__main__":
    command, target = sys.argv[1], Path(sys.argv[2])
    if command == "generate":
        generate(target)
    else:
        raise SystemExit(verify(target))
