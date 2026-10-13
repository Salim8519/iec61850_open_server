"""SCD versus device model comparison categories (LIVE-06)."""
from studio_app.domain.model_compare import compare

IED = {
    "signals": [{"ref": "D/XCBR1.Pos.stVal", "fc": "ST", "kind": "Dbpos"},
                {"ref": "D/MMXU1.TotW.mag.f", "fc": "MX", "kind": "FLOAT32"},
                {"ref": "D/GGIO1.Ind1.stVal", "fc": "ST", "kind": "BOOLEAN"}],
    "datasets": [{"ref": "D/LLN0$Events", "fcda_refs": ["D/XCBR1.Pos.stVal", "D/GGIO1.Ind1.stVal"]},
                 {"ref": "D/LLN0$Gone", "fcda_refs": []}],
    "reports": [{"ref": "D/LLN0.RP.Events01"}, {"ref": "D/LLN0.BR.Missing01"}],
}
DISCOVERY = {"devices": [{"name": "D", "nodes": [
    {"name": "XCBR1", "leaves": [{"ref": "D/XCBR1.Pos.stVal", "fc": "ST", "mms_type": 3}]},
    {"name": "MMXU1", "leaves": [{"ref": "D/MMXU1.TotW.mag.f", "fc": "MX", "mms_type": 4}]},
    {"name": "GGIO1", "leaves": [{"ref": "D/GGIO1.Ind2.stVal", "fc": "ST", "mms_type": 2}]},
    {"name": "LLN0", "leaves": [], "dataset_members": {
        "Events": {"members": ["D/XCBR1.Pos.stVal[ST]", "D/GGIO1.Ind2.stVal[ST]"]}},
     "unbuffered_reports": ["Events01"], "buffered_reports": []},
]}]}


def test_signal_categories():
    result = compare(IED, DISCOVERY)
    categories = {row["ref"]: row["category"] for row in result["signals"]}
    assert categories == {"D/XCBR1.Pos.stVal": "match", "D/MMXU1.TotW.mag.f": "type_mismatch",
                          "D/GGIO1.Ind1.stVal": "missing_on_device", "D/GGIO1.Ind2.stVal": "extra_on_device"}
    assert result["summary"] == {"match": 1, "missing_on_device": 1, "extra_on_device": 1, "type_mismatch": 1,
                                 "datasets_different": 2, "reports_missing": 1}
    mismatch = next(row for row in result["signals"] if row["category"] == "type_mismatch")
    assert mismatch["device_type"] == "integer" and mismatch["hint"]


def test_dataset_and_report_differences():
    result = compare(IED, DISCOVERY)
    events = next(item for item in result["datasets"] if item["ref"] == "D/LLN0$Events")
    assert events["status"] == "different"
    assert events["missing"] == ["D/GGIO1.Ind1.stVal"] and events["extra"] == ["D/GGIO1.Ind2.stVal"]
    assert next(item for item in result["datasets"] if item["ref"] == "D/LLN0$Gone")["status"] == "missing_on_device"
    assert {item["ref"]: item["status"] for item in result["reports"]} == {
        "D/LLN0.RP.Events01": "match", "D/LLN0.BR.Missing01": "missing_on_device"}
