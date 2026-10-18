"""The 104 master also asks for integrated totals (counter interrogation) and words a negative
select confirmation plainly. Found against the FreyrSCADA 104 server simulator (change 0089):
its counter IOA 800 never appeared because only a station interrogation was sent."""
import pytest
from iec104_standin import C_CI, C_IC, Station104
from test_master_wire import BIN, Master

pytestmark = pytest.mark.skipif(not (BIN / "studio_master.exe").exists(), reason="Master build required")


def connect(port):
    master = Master()
    master.process.request("connect", protocol="iec104", settings={"host": "127.0.0.1", "port": port})
    assert master.wait_for(lambda: "open" in master.links)
    return master


def test_master_reads_counters_after_the_station_interrogation():
    station = Station104(12431)
    master = connect(12431)
    try:
        answer = master.process.request("interrogate", timeout=60)
        assert answer["confirmed"] is True and answer["counters"] == "confirmed"
        assert [kind for kind, _, _ in station.received][:2] == [C_IC, C_CI]
        assert master.wait_for(lambda: master.latest(800) is not None)  # values follow the confirmation
        counter = master.latest(800)
        assert counter and counter["group"] == "counter" and counter["value"] == 4242
        assert counter["cause"] == "counter interrogated"
        assert any("C_CI_NA_1 counter interrogation" in line for line in master.traffic)
    finally:
        master.close()
        station.close()


def test_a_device_without_counters_does_not_fail_discovery():
    station = Station104(12432, counters=False)
    master = connect(12432)
    try:
        answer = master.process.request("interrogate", timeout=60)
        assert answer["confirmed"] is True and answer["counters"].startswith("refused")
        assert master.wait_for(lambda: master.latest(100) is not None) and master.latest(100)["value"] == 1
        assert any(line.startswith("C_CI_NA_1 activation confirmed") and "(negative)" in line
                   for line in master.traffic), "the refusal names C_CI_NA_1, not 'other'"
    finally:
        master.close()
        station.close()


def test_a_refused_select_is_explained_not_called_confirmed():
    station = Station104(12433)
    master = connect(12433)
    try:
        refused = master.process.request("command", kind="double", address=2000, value=1, select=True)
        text = refused["steps"][0]["text"]
        assert not refused["ok"] and "negative confirmation" in text and "direct execute" in text
        assert "activation confirmed" not in text
        direct = master.process.request("command", kind="double", address=2000, value=1)
        assert direct["ok"] and direct["terminated"]
    finally:
        master.close()
        station.close()
