"""Station health GOOSE checks (HEALTH-05, HEALTH-07)."""
import re
from pathlib import Path

from studio_app.domain.goose_consistency import device_gocb_checks, unknown_publishers, wire_check
from studio_app.acquisition.goose_monitor import UNKNOWN_FIELDS

NATIVE = Path(__file__).resolve().parents[2] / "native/live"
CONTROL = {"gocb_ref": "P1LD0/LLN0$GO$gcb1", "go_id": "P1-gcb1", "dataset_ref": "P1LD0/LLN0$DS1", "conf_rev": 3}
IED = {"goose": {"publishes": [CONTROL]}}


def reads(**values):
    return {"values": {f"P1LD0/LLN0.gcb1.{key}": value for key, value in values.items()}, "errors": {},
            "checked_at": "2026-10-14T10:00:00+00:00", "source": "mms_read"}


def test_device_gocb_attributes_are_compared():
    ok = device_gocb_checks(IED, reads(GoEna=True, GoID="P1-gcb1", DatSet="P1LD0/LLN0.DS1", ConfRev=3))
    assert [row["result"] for row in ok] == ["ok"]
    bad = device_gocb_checks(IED, reads(GoEna=True, GoID="other", DatSet="P1LD0/LLN0$DS2", ConfRev=4))[0]
    assert bad["result"] == "problem" and bad["source"] == "mms_read" and bad["observed_at"]
    for text in ("other", "P1-gcb1", "P1LD0/LLN0$DS2", "P1LD0/LLN0$DS1", "confRev is 4", "3 in the SCD"):
        assert text in bad["message"], text
    off = device_gocb_checks(IED, reads(GoEna=False, GoID="P1-gcb1", DatSet="P1LD0/LLN0$DS1", ConfRev=3))[0]
    assert off["result"] == "warning" and "switched off" in off["message"]
    assert device_gocb_checks(IED, reads())[0]["result"] == "warning"


def test_unknown_publisher_details_and_cap():
    native = (NATIVE / "goose_traffic.cpp").read_text(encoding="utf-8")
    unknown_event = native[native.index('"goose_unknown"}'):]
    sent = set(re.findall(r'\{"([a-z_]+)", ', unknown_event[:unknown_event.index(");")]))
    assert {"app_id", "mac", "src_mac", "vlan_id", "go_id", "conf_rev", "st_num", "dataset", "gocb_ref"} <= sent
    assert set(UNKNOWN_FIELDS) - {"test"} <= sent | {"gocb_ref"}
    assert '"goose_unknown_cap"' in native
    rows, note = unknown_publishers({"unknown": [{"gocb_ref": "X/LLN0$GO$a", "app_id": 1}], "unknown_limit": 500})
    assert rows[0]["gocb_ref"] == "X/LLN0$GO$a" and "500" in note
    assert unknown_publishers({"unknown": []}) == ([], "")


def test_wire_states_and_listener_off():
    watch = {"gocb_ref": "A", "state": "receiving", "frames": 5, "observed_at": "t"}
    assert wire_check(watch, {"state": "listening"})["result"] == "ok"
    assert wire_check({**watch, "received_test": True}, {"state": "listening"})["result"] == "warning"
    assert wire_check({**watch, "state": "timeout", "problem": "timeout"}, {"state": "listening"})["result"] == "problem"
    off = wire_check(watch, {"state": "needs_npcap", "message": "Install Npcap"})
    assert off["result"] == "not_checked" and off["message"] == "Install Npcap"
