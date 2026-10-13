from pathlib import Path

from studio_app.projects.service import ProjectService


def test_removal_keeps_drawing_and_creates_restorable_backup(tmp_path):
    projects = ProjectService(tmp_path)
    first = projects.add_template("First")[0]
    second = projects.add_template("Second")[0]
    reference = "FirstCTRL/GGIO1.AnIn1.mag.f"
    projects.source(first["id"], reference, {"mode": "sine"})
    projects.diagram({"nodes": [{"id": "breaker", "position": {"x": 20, "y": 30},
                      "data": {"label": "CB1", "kind": "CB", "state_mapping": {"mode": "direct",
                               "source": {"ied_id": first["id"],
                                          "ref": "FirstCTRL/XCBR1.Pos.stVal"}}}}], "edges": []})
    projects.scenarios([{"id": "test", "name": "Test", "enabled": False, "steps": [
        {"id": "first", "at": 0, "ied_id": first["id"], "ref": reference, "value": 12},
        {"id": "second", "at": 1, "ied_id": second["id"],
         "ref": "SecondCTRL/GGIO1.AnIn1.mag.f", "value": 24}]}])
    preview = projects.removal_preview(first["id"])
    assert preview["bindings"] == 1 and preview["sources"] == 1 and preview["scenario_steps"] == 1
    result = projects.remove_ied(first["id"])
    assert [ied["name"] for ied in projects.project["ieds"]] == ["Second"]
    assert projects.project["diagram"]["nodes"][0]["data"]["state_mapping"] is None
    assert projects.project["sources"] == []
    assert len(projects.project["scenarios"][0]["steps"]) == 1
    recovery = Path(result["recovery_file"])
    assert recovery.is_relative_to(tmp_path / "recovery")
    projects.restore(recovery.read_bytes())
    assert len(projects.project["ieds"]) == 2
    assert len(projects.project["scenarios"][0]["steps"]) == 2
    mapping = projects.project["diagram"]["nodes"][0]["data"]["state_mapping"]
    assert mapping["source"]["ied_id"] == first["id"]
