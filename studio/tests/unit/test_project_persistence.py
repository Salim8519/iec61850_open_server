import copy
import math

import pytest
from studio_app.contracts.errors import StudioError
from studio_app.domain.source import validate_source
from studio_app.projects.archive import encode
from studio_app.projects.service import ProjectService


def test_sources_and_identifiers_survive_roundtrip(tmp_path):
    projects = ProjectService(tmp_path)
    model = projects.add_template("Persistent")[0]
    projects.source(model["id"], "PersistentCTRL/GGIO1.AnIn1.mag.f",
                    {"mode": "sine", "offset": 110, "amplitude": 5, "period": 2})
    before = projects.snapshot()
    projects.restore(projects.export())
    after = projects.snapshot()
    assert after["id"] == before["id"]
    assert after["ieds"][0]["id"] == before["ieds"][0]["id"]
    assert after["sources"] == before["sources"]


def test_invalid_diagram_does_not_replace_project(tmp_path):
    projects = ProjectService(tmp_path)
    projects.add_template("KeepMe")
    original = copy.deepcopy(projects.project)
    invalid = copy.deepcopy(original)
    invalid["diagram"]["edges"] = [{"id": "bad", "source": "missing", "target": "missing"}]
    with pytest.raises(StudioError, match="missing equipment"):
        projects.restore(encode(invalid))
    assert projects.project == original


@pytest.mark.parametrize("value", [math.nan, math.inf, -math.inf])
def test_source_rejects_nonfinite_parameters(value):
    with pytest.raises(ValueError, match="finite"):
        validate_source({"kind": "FLOAT32", "fc": "MX"}, {"mode": "sine", "amplitude": value})


def test_duplicate_ied_identifier_is_rejected(tmp_path):
    projects = ProjectService(tmp_path)
    projects.add_template("First")
    projects.add_template("Second")
    invalid = copy.deepcopy(projects.project)
    invalid["ieds"][1]["id"] = invalid["ieds"][0]["id"]
    with pytest.raises(StudioError, match="duplicate"):
        projects.restore(encode(invalid))
