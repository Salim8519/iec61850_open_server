"""Sampled Values supervision rules (SV-05, SV-06, SV-09)."""
from studio_app.domain.sv_supervision import STREAM_LIMIT, assess, watch_list

STREAM = {"svcb_ref": "MU1LD0/LLN0.MSVCB01", "sv_id": "MU1", "conf_rev": 1, "smp_rate": 80, "smp_mod": 0,
          "mac": "01-0C-CD-04-00-01", "app_id": 0x4000, "layout": "9-2LE", "channels": ["TCTR1.AmpSv"] * 8,
          "scale": [0.001] * 8}


def watch(nominal=50, **changes):
    return watch_list([{"name": "MU1", "role": "real", "sv": {"publishes": [{**STREAM, **changes}]}}], nominal)[0][0]


def event(**changes):
    return {"state": "receiving", "asdus": 4000, "rate": 4000.0, "conf_rev": 1, "smp_synch": 2, "lost": 0,
            "invalid_quality": 0, "data_size": 64, "simulate": False, **changes}


def test_rate_mismatch():
    assert watch()["expected_rate"] == 4000 and watch(60)["expected_rate"] == 4800
    assert watch(smp_mod=1, smp_rate=4800)["expected_rate"] == 4800
    assert assess(watch(), event()) == {"state": "receiving", "problems": []}
    assert assess(watch(), event(rate=3990.0))["problems"] == []
    assert assess(watch(60), event())["problems"] == ["rate_mismatch"]
    assert assess(watch(), event(conf_rev=2, rate=2000.0))["problems"] == ["conf_rev_mismatch", "rate_mismatch"]


def test_unsynchronised_stream():
    assert assess(watch(), event(smp_synch=0)) == {"state": "problem", "problems": ["not_synchronised"]}
    assert assess(watch(), event(smp_synch=1))["problems"] == []


def test_loss_quality_and_simulation_are_reported_when_new():
    assert assess(watch(), event(lost=5), {"lost": 2})["problems"] == ["sample_loss"]
    assert assess(watch(), event(lost=5), {"lost": 5})["problems"] == []
    assert assess(watch(), event(invalid_quality=3))["problems"] == ["invalid_quality"]
    assert assess(watch(), event(simulate=True))["problems"] == ["simulation"]
    assert assess(watch(), {"state": "waiting", "asdus": 0}) == {"state": "waiting", "problems": []}


def test_stream_limit_is_reported():
    ieds = [{"name": f"MU{i}", "role": "real",
             "sv": {"publishes": [{**STREAM, "svcb_ref": f"MU{i}LD0/LLN0.MSVCB01"}]}} for i in range(20)]
    supervised, extra = watch_list(ieds, 50)
    assert len(supervised) == STREAM_LIMIT == 16
    assert [item["publisher"] for item in extra] == ["MU16", "MU17", "MU18", "MU19"]
    assert watch_list([{**ieds[0], "role": "off"}], 50) == ([], [])
