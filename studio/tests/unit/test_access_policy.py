"""Free report blocks only, polling otherwise (LIVE-02)."""
from studio_app.domain.access_policy import choose_access

IED = {"reports": [
    {"ref": "D/LLN0.RP.Events01", "dataset_ref": "D/LLN0$Events"},
    {"ref": "D/LLN0.RP.Events02", "dataset_ref": "D/LLN0$Events"},
    {"ref": "D/LLN0.BR.Meas01", "dataset_ref": "D/LLN0$Meas"},
]}


def test_busy_blocks_fall_back_to_polling():
    status = {"D/LLN0.RP.Events01": {"readable": True, "free": False, "enabled": True},
              "D/LLN0.RP.Events02": {"readable": True, "free": True},
              "D/LLN0.BR.Meas01": {"readable": True, "free": False, "owned": True}}
    result = choose_access(IED, status)
    assert result["reports"] == ["D/LLN0.RP.Events02"]
    assert result["polling_datasets"] == ["D/LLN0$Meas"]
    assert any("Events02" in note and "free" in note for note in result["notes"])
    assert any("in use by other clients" in note and "Meas" in note for note in result["notes"])


def test_unreadable_blocks_are_never_used():
    result = choose_access(IED, {})
    assert result["reports"] == [] and len(result["polling_datasets"]) == 2
    assert all("did not answer" in note for note in result["notes"])


def test_polling_mode_never_touches_blocks():
    status = {ref: {"readable": True, "free": True} for ref in
              ("D/LLN0.RP.Events01", "D/LLN0.RP.Events02", "D/LLN0.BR.Meas01")}
    result = choose_access(IED, status, "polling")
    assert result["reports"] == [] and "will not use any report block" in result["notes"][0]
