"""Sampled Values interpretation and nominal frequency (SV-07, SV-08)."""
import pytest
from studio_app.domain.network_settings import validate_network
from studio_app.domain.sv_supervision import native_stream, watch_list
from test_sv_supervision import STREAM


def stream(nominal=50, **changes):
    return watch_list([{"name": "MU1", "role": "real", "sv": {"publishes": [{**STREAM, **changes}]}}], nominal)[0][0]


def test_unknown_layout_is_not_interpreted():
    raw = native_stream(stream(layout="raw", channels=[], scale=[]))
    assert raw["layout"] == "raw" and raw["channels"] == [] and raw["scale"] == []
    seconds = native_stream(stream(smp_mod=2, smp_rate=1))
    assert seconds["layout"] == "raw" and seconds["per_cycle"] == 0
    known = native_stream(stream())
    assert known["layout"] == "9-2LE" and known["per_cycle"] == 80 and known["wrap"] == 4000


def test_nominal_frequency_is_explicit():
    assert validate_network({})["nominal_hz"] == 50
    assert validate_network({"nominal_hz": 60})["nominal_hz"] == 60
    for wrong in (55, "50", True, None):
        with pytest.raises(ValueError, match="50 Hz or 60 Hz"):
            validate_network({"nominal_hz": wrong})
    assert native_stream(stream(60))["wrap"] == 4800
    assert native_stream(stream(60, smp_mod=1, smp_rate=4800))["per_cycle"] == 80


def test_frequency_survives_mode_changes():
    from studio_app.projects.ied_settings import _merged_network
    lab = {"mode": "lab", "adapter_guid": "{15A6105F-A715-444F-9D7C-AA259A2EED8A}", "adapter_label": "Lab",
           "nominal_hz": 60}
    assert validate_network(_merged_network(lab, {"mode": "loopback"})) == {
        "mode": "loopback", "adapter_guid": "", "adapter_label": "", "nominal_hz": 60}
    assert validate_network(_merged_network(lab, {"nominal_hz": 50}))["mode"] == "lab"
