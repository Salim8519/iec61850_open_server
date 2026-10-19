"""Drive the frozen Flask app through the Tools routes and print every exact response.

Public templates only. Prints {"ieds": ...} first (the Go fakes load the same models),
then one line per request: method, path, body, file, status and the response text.
Nothing here needs a running IED, a device or the network.
"""
import io
import json
import os
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, os.environ["STUDIO_BACKEND"])
os.environ["SCADABENCH_UPDATE_CHECK"] = "off"
from studio_app.app_factory import create_app  # noqa: E402
from studio_app.settings import Settings  # noqa: E402

BIN = Path(os.environ["STUDIO_NATIVE_BIN"])
folder = Path(tempfile.mkdtemp())
app = create_app(Settings(folder, BIN / "studio_ied.exe", folder), "oracle-token")
assert app.extensions["startup"].wait(60)
client = app.test_client()
client.post("/api/v1/session", json={"token": "oracle-token"})
projects = app.extensions["projects"]
projects.add_template("BayA")
projects.add_template("BayB")
print(json.dumps({"ieds": projects.project["ieds"]}))


def send(method, path, body=None, raw=None, content_type="application/json", file=None):
    if file is not None:
        response = client.open("/api/v1" + path, method=method,
                               data={"file": (io.BytesIO(file.encode("latin-1")), "trace.csv")},
                               content_type="multipart/form-data")
    elif raw is not None:
        response = client.open("/api/v1" + path, method=method, data=raw, content_type=content_type)
    elif body is not None:
        response = client.open("/api/v1" + path, method=method, data=json.dumps(body), content_type=content_type)
    else:
        response = client.open("/api/v1" + path, method=method)
    text = response.get_data(as_text=True)
    print(json.dumps({"method": method, "path": path, "body": body, "raw": raw, "content_type": content_type,
                      "file": file, "status": response.status_code, "response": text,
                      "headers": {key: response.headers.get(key) for key in ("Content-Type", "Content-Disposition")}}))
    return response


sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I leaves the script folder out
for name in sys.argv[1:]:
    __import__(name).run(send, projects)
app.extensions["runtime"].close()
