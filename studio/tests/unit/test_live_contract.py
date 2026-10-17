"""studio_live has a fixed, read-only operation table (LIVE-01)."""
import re
from pathlib import Path

NATIVE = Path(__file__).resolve().parents[2] / "native"
ALLOWED = {"hello", "npcap_status", "adapters", "goose_start", "goose_stop", "connect", "discover", "read",
           "rcb_status", "rcb_reserve", "rcb_release", "disconnect", "sv_start", "sv_stop",
           # IED Explorer / Find IEDs: directory, type-description and MMS Identify services only.
           "explore", "identify"}
FORBIDDEN_CALLS = ("ControlObjectClient", "IedConnection_writeObject", "IedConnection_writeBooleanValue",
                   "IedConnection_writeFloatValue", "IedConnection_writeVisibleStringValue",
                   "IedConnection_selectActiveSG", "IedConnection_setFile", "IedConnection_deleteFile",
                   "GoosePublisher_publish", "IedConnection_createDataSet", "IedConnection_deleteDataSet")


def test_operation_table_is_read_only():
    main = (NATIVE / "live/main.cpp").read_text(encoding="utf-8")
    operations = set(re.findall(r'\{"([a-z_]+)", \[', main))
    assert operations == ALLOWED
    for source in (NATIVE / "live").glob("*.cpp"):
        text = source.read_text(encoding="utf-8")
        for call in FORBIDDEN_CALLS:
            assert call not in text, f"{source.name} must not call {call}"
    reports = (NATIVE / "live/reports.cpp").read_text(encoding="utf-8")
    elements = set(re.findall(r"RCB_ELEMENT_[A-Z_]+", reports))
    assert elements == {"RCB_ELEMENT_RESV", "RCB_ELEMENT_RPT_ENA", "RCB_ELEMENT_GI"}


def test_health_reads_are_read_only():
    import ast
    backend = NATIVE.parent / "backend/studio_app/acquisition"
    used = set()
    for name in ("health_checks.py", "real_session.py"):
        tree = ast.parse((backend / name).read_text(encoding="utf-8"))
        for node in ast.walk(tree):
            if isinstance(node, ast.Call) and getattr(node.func, "attr", "") == "request" and node.args:
                if isinstance(node.args[0], ast.Constant):
                    used.add(node.args[0].value)
    # Health checks only use reads; reservation stays the LIVE-01 exception of the session itself.
    health = ast.parse((backend / "health_checks.py").read_text(encoding="utf-8"))
    health_ops = {node.args[0].value for node in ast.walk(health) if isinstance(node, ast.Call)
                  and getattr(node.func, "attr", "") == "request" and node.args and isinstance(node.args[0], ast.Constant)}
    assert health_ops == {"read"}
    assert used <= ALLOWED
    test_operation_table_is_read_only()


def test_mode_status_is_read_only():
    import ast
    backend = NATIVE.parent / "backend/studio_app"
    checks = ast.parse((backend / "acquisition/health_checks.py").read_text(encoding="utf-8"))
    ops = {node.args[0].value for node in ast.walk(checks) if isinstance(node, ast.Call)
           and getattr(node.func, "attr", "") == "request" and node.args and isinstance(node.args[0], ast.Constant)}
    assert ops == {"read"}
    apply = (backend / "simulation/test_mode_apply.py").read_text(encoding="utf-8")
    assert 'ied.get("role") != "simulated"' in apply and "sample_writer.apply" in apply
    assert "live" not in apply.split("import")[1].split("\n")[0]


def test_explorer_operations_use_only_read_services():
    allowed = {"IedConnection_create", "IedConnection_destroy", "IedConnection_setConnectTimeout",
               "IedConnection_setRequestTimeout", "IedConnection_connect", "IedConnection_close",
               "IedConnection_getMmsConnection", "MmsConnection_identify"}
    for name in ("explore.cpp", "identify.cpp"):
        text = (NATIVE / "live" / name).read_text(encoding="utf-8")
        assert set(re.findall(r"\b(?:IedConnection|MmsConnection)_[A-Za-z]+", text)) <= allowed, name
    browse = (NATIVE / "common/browse.cpp").read_text(encoding="utf-8")
    calls = set(re.findall(r"\bIedConnection_[A-Za-z]+", browse))
    assert all(call.startswith("IedConnection_get") for call in calls), calls
