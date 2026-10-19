"""Record the Python engine's answers to the runtime routes while nothing runs (golden file).

Usage: studio/.venv/Scripts/python.exe record_flask.py
Reads stopped_requests.json, sends each request through Flask's test client of the frozen
Python app (synthetic template project), writes stopped_golden.json: [status, body text].
"""
import json
import os
import sys
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[5]
sys.path.insert(0, str(ROOT / "studio" / "backend"))
os.environ["SCADABENCH_UPDATE_CHECK"] = "off"

from studio_app.app_factory import create_app  # noqa: E402
from studio_app.settings import Settings  # noqa: E402


def main():
    requests = json.loads((HERE / "stopped_requests.json").read_text(encoding="utf-8"))
    project = json.loads((HERE.parents[2] / "runtime" / "testdata" / "project.json").read_text(encoding="utf-8"))
    with tempfile.TemporaryDirectory() as directory:
        folder = Path(directory)
        app = create_app(Settings(folder, folder / "studio_ied.exe", folder), "record-token")
        app.extensions["startup"].wait()
        app.extensions["projects"].project = project
        client = app.test_client()
        client.post("/api/v1/session", json={"token": "record-token"})
        golden = []
        for method, path, content_type, body in requests:
            headers = {"Content-Type": content_type} if content_type else {}
            response = client.open("/api/v1" + path, method=method, data=body.encode("utf-8"), headers=headers)
            golden.append([response.status_code, response.get_data(as_text=True)])
        app.extensions["runtime"].close()
    (HERE / "stopped_golden.json").write_text(json.dumps(golden, indent=1), encoding="utf-8")


if __name__ == "__main__":
    main()
