"""Autonomous tools (owner decision 2026-10-17): settings are remembered, Explorer watches
key points by itself, and the gateway proposes its points, starts Run and resumes."""
from pathlib import Path
from types import SimpleNamespace

from studio_app.domain.explorer_watch import key_nodes, key_points, merge, node_class
from studio_app.gateway.manager import GatewayManager
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template
from studio_app.tools.tool_memory import ToolMemory


def test_tool_memory_survives_damage_and_keeps_every_tool(tmp_path):
    memory = ToolMemory(tmp_path)
    assert memory.get()["explorer"]["port"] == 102
    memory.update("explorer", host="10.0.0.5", interval_ms=2000,
                  watch_by_host={"10.0.0.5:102": [{"ref": "LD0/XCBR1.Pos.stVal", "fc": "ST"}]})
    memory.update("find", range="10.0.0.0/24", choice="Ethernet 2")
    again = ToolMemory(tmp_path).get()
    assert again["explorer"]["host"] == "10.0.0.5" and again["explorer"]["interval_ms"] == 2000
    assert again["explorer"]["watch_by_host"]["10.0.0.5:102"][0]["fc"] == "ST"
    assert again["find"] == {"range": "10.0.0.0/24", "port": 102, "choice": "Ethernet 2"}
    (tmp_path / "tool-settings.json").write_text("{broken", encoding="utf-8")
    assert ToolMemory(tmp_path).get()["explorer"]["host"] == ""


def test_explorer_picks_key_nodes_and_points():
    assert node_class("QA1XCBR1") == "XCBR" and node_class("LLN0") == "LLN0" and node_class("Foo") == ""
    assert key_nodes("LD0", ["LLN0", "PDIS1", "QA1XCBR1", "MMXU1"]) == ["LD0/LLN0", "LD0/QA1XCBR1", "LD0/MMXU1"]
    attribute = lambda ref, fc, path: {"ref": ref, "fc": fc, "path": path, "type": "x"}
    breaker = {"name": "QA1XCBR1", "objects": [
        {"name": "Pos", "attributes": [attribute("LD0/QA1XCBR1.Pos.stVal", "ST", "stVal"),
                                       attribute("LD0/QA1XCBR1.Pos.ctlModel", "CF", "ctlModel")]},
        {"name": "OpCnt", "attributes": [attribute("LD0/QA1XCBR1.OpCnt.stVal", "ST", "stVal")]}]}
    assert [point["ref"] for point in key_points(breaker)] == ["LD0/QA1XCBR1.Pos.stVal"]
    many = [{"ref": f"r{n % 50}", "fc": "ST"} for n in range(100)]
    assert len(merge(many)) == 40


class FakeProjects:
    def __init__(self, ieds, gateway=None):
        self.project = {"ieds": ieds, "gateway": gateway}

    def gateway(self, config):
        self.project["gateway"] = config


def manager_for(ieds, gateway=None):
    coordinator = SimpleNamespace(state="stopped", index={}, emit=lambda *a, **k: None,
                                  start=lambda: setattr(coordinator, "state", "running"))
    return GatewayManager(coordinator, FakeProjects(ieds, gateway), Path("missing/studio_ied.exe"))


def test_gateway_prepares_points_and_run_by_itself():
    ied = import_scl(make_template("BayA"))[0]
    ied.update(id="a", role="simulated")
    manager = manager_for([ied])
    config = manager._prepare()
    assert config["points"]["iec104"] and manager.coordinator.state == "running"
    assert not config.get("auto_start")  # remembered only after a successful start


def test_monitor_only_follows_role_and_real_command_setting():
    real = {"id": "b", "role": "real"}
    manager = manager_for([real, {"id": "c", "role": "off"}])
    assert manager._monitor_only({"ied": "b"}, {"real_commands": True}) is False
    assert manager._monitor_only({"ied": "b"}, {"real_commands": False}) is True
    assert manager._monitor_only({"ied": "c"}, {}) is True


def test_autostart_only_when_remembered_and_never_raises():
    manager = manager_for([], {"protocol": "iec104", "listen": "this_pc", "settings": {}, "points": {},
                               "auto_start": True})
    manager.autostart()  # the gateway program is missing here: reported, not raised
    assert manager.state == "failed" and "Automatic start failed" in manager.message
    quiet = manager_for([])
    quiet.autostart()
    assert quiet.state == "stopped"
