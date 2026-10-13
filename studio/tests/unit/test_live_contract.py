"""studio_live has a fixed, read-only operation table (LIVE-01)."""
import re
from pathlib import Path

NATIVE = Path(__file__).resolve().parents[2] / "native"
ALLOWED = {"hello", "npcap_status", "adapters", "goose_start", "goose_stop", "connect", "discover", "read",
           "rcb_status", "rcb_reserve", "rcb_release", "disconnect"}
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
