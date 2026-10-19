"""Shared helpers of the project golden-vector generator: fixtures, stubs and capture.

The frozen Python backend (studio/backend) is the reference. Every vector argument is
round-tripped through json so Python and Go see exactly the same decoded values.
"""
import functools
import hashlib
import json
import threading
from pathlib import Path

from pydantic import ValidationError
from studio_app.projects.service import ProjectService
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template
from studio_app.networking.loopback import assign_endpoints
from studio_app.domain.network_settings import validate_ied_settings

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[4]
SCD = REPO / "scd" / "model_substation.scd"
LIGHT_KEYS = ("id", "name", "filename", "ip", "port", "state", "edition", "role", "live", "lab_ip",
              "goose_publish", "test_mode", "scl_address")
FIXTURE_IDS = {"BayController": "1" * 32, "Bay2": "2" * 32, "FEED1": "3" * 32}


@functools.lru_cache(maxsize=1)
def fixture_sources():
    """{fixture name: (source text, filename)}; FEED1 comes from the public sample SCD."""
    return {"BayController": (make_template("BayController").decode("utf-8"), "BayController.cid"),
            "Bay2": (make_template("Bay2").decode("utf-8"), "Bay2.cid"),
            "FEED1": (SCD.read_text(encoding="utf-8"), "model_substation.scd")}


def build_fixtures():
    """IED models as ProjectService.add_scl builds them, with deterministic ids."""
    models = []
    for name, (source, filename) in fixture_sources().items():
        model = import_scl(source, filename, [name])[0]
        assert model["source"] == source, name
        model.update(validate_ied_settings({}))
        model["id"] = FIXTURE_IDS[name]
        models.append(model)
    assign_endpoints(models)
    return {model["name"]: roundtrip(model) for model in models}


def roundtrip(value):
    return json.loads(json.dumps(value))


def digest(value):
    return "sha256:" + hashlib.sha256(json.dumps(value).encode("utf-8")).hexdigest()


def reduce_ied(ied):
    """An IED with its heavy runtime model parts replaced by digests (same rule in Go)."""
    return {key: value if key in LIGHT_KEYS else digest(value) for key, value in ied.items()}


def reduce_project(project):
    if not isinstance(project, dict) or not isinstance(project.get("ieds"), list):
        return project
    return {**project, "ieds": [reduce_ied(ied) if isinstance(ied, dict) and "signals" in ied else ied
                                for ied in project["ieds"]]}


class Stub:
    """The parts of ProjectService the project functions use, without disk writes."""

    find_ied = ProjectService.find_ied
    removal_preview = ProjectService.removal_preview
    snapshot = ProjectService.snapshot

    def __init__(self, project, directory=None, notice=None, session_id="", revision=0):
        self.project = project
        self.lock = threading.RLock()
        self.directory = directory
        self.migration_notice = notice
        self.session_id = session_id
        self.revision = revision

    def save(self):
        pass

    def changed(self, source=None):
        self.revision += 1

    def export(self):
        return b"payload"


def capture(call):
    """{"out": json.dumps(result)} or {"err": {...}} with Python's class, text and details."""
    try:
        return {"out": json.dumps(call())}
    except ValidationError as error:
        return {"err": {"class": "ValidationError",
                        "details": error.errors(include_context=False, include_input=False)}}
    except Exception as error:  # every class is recorded; Go must raise the same one
        details = getattr(error, "details", None)
        return {"err": {"class": type(error).__name__, "message": str(error),
                        "code": getattr(error, "code", None), "details": details}}


class Fixed:
    """A uuid4() replacement with a fixed hex value."""

    def __init__(self, text):
        self.hex = text


def fixed_uuid(text):
    return lambda: Fixed(text)
