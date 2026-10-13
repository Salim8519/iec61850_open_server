"""Format 11 adds network settings without changing how older projects behave."""
import io
import json
import zipfile

from studio_app.projects.document import CURRENT_FORMAT
from studio_app.projects.service import ProjectService


def _as_format(projects, version):
    payload = json.loads(zipfile.ZipFile(io.BytesIO(projects.export())).read("project.json"))
    payload["format"] = version
    payload.pop("network", None)
    for ied in payload["ieds"]:
        for key in ("role", "live", "lab_ip", "goose_publish"):
            ied.pop(key, None)
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("project.json", json.dumps(payload))
    return stream.getvalue()


def test_format_10_opens_in_loopback_mode(tmp_path):
    source = ProjectService(tmp_path / "a")
    source.add_template("Bay")
    legacy = _as_format(source, 10)
    projects = ProjectService(tmp_path / "b")
    projects.restore(legacy)
    assert CURRENT_FORMAT == 11
    assert projects.project["network"] == {"mode": "loopback", "adapter_guid": "", "adapter_label": ""}
    ied = projects.project["ieds"][0]
    assert ied["role"] == "simulated" and ied["lab_ip"] == "" and ied["goose_publish"] is True
    assert ied["ip"].startswith("127.50.0.")
    notice = projects.migration_notice
    assert "simulated on this PC only" in notice["message"] and notice["backup"]


def test_format_9_archives_are_accepted(tmp_path):
    source = ProjectService(tmp_path / "a")
    source.add_template("Nine")
    projects = ProjectService(tmp_path / "b")
    projects.restore(_as_format(source, 9))
    assert projects.project["ieds"][0]["name"] == "Nine"


def test_settings_survive_save_and_reopen(tmp_path):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("Keep")[0]
    ied.update(role="real", lab_ip="192.168.10.21",
               live={"ip": "10.1.1.5", "port": 102, "access": "polling", "poll_ms": 500})
    projects.project["network"] = {"mode": "lab", "adapter_guid": "{15A6105F-A715-444F-9D7C-AA259A2EED8A}",
                                   "adapter_label": "Lab NIC"}
    reopened = ProjectService(tmp_path / "copy")
    reopened.restore(projects.export())
    stored = reopened.project["ieds"][0]
    assert stored["role"] == "real" and stored["lab_ip"] == "192.168.10.21"
    assert stored["live"] == {"ip": "10.1.1.5", "port": 102, "access": "polling", "poll_ms": 500}
    assert reopened.project["network"]["mode"] == "lab"
    assert reopened.migration_notice is None
