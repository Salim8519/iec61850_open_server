"""Sampled Values from an independent test publisher, received by studio_live (SV-03/04/11)."""
import os
import time
from pathlib import Path

import pytest
from studio_app.runtime.worker_process import WorkerProcess

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
ADAPTER = os.environ.get("STUDIO_GOOSE_ADAPTER", "")
pytestmark = [pytest.mark.skipif(not (BIN / "studio_sv_test_publisher.exe").exists(), reason="Native build required"),
              pytest.mark.skipif(not ADAPTER, reason="Set STUDIO_GOOSE_ADAPTER to a lab adapter GUID")]
CHANNELS = ["IA", "IB", "IC", "IN", "VA", "VB", "VC", "VN"]
STREAM = {"key": "s1", "app_id": 0x4001, "mac": "01-0C-CD-04-00-21", "layout": "9-2LE", "per_cycle": 80,
          "wrap": 4000, "channels": CHANNELS, "scale": [0.001] * 4 + [0.01] * 4}


def listen(seconds, *extra, **publisher):
    """Run studio_live on STREAM with one test publisher (plus optional extra publishers)."""
    events = []
    live = WorkerProcess(BIN / "studio_live.exe", events.append)
    sources = [WorkerProcess(BIN / "studio_sv_test_publisher.exe", lambda event: None) for _ in range(1 + len(extra))]
    try:
        live.request("sv_start", timeout=30, adapter_guid=ADAPTER, streams=[STREAM])
        start = time.monotonic()
        sources[0].request("start", timeout=30, adapter_guid=ADAPTER, mac=STREAM["mac"], app_id=STREAM["app_id"],
                           sv_id="MU01", **publisher)
        for source, settings in zip(sources[1:], extra):
            source.request("start", timeout=30, adapter_guid=ADAPTER, **settings)
        time.sleep(seconds)
        return [event for event in events if event.get("event") == "sv_rx"], events, time.monotonic() - start
    finally:
        for source in sources:
            source.close()
        live.close()


def test_independent_publisher_is_received():
    reports, _, _ = listen(3, amps=1.0, volts=63.5)
    last = reports[-1]
    assert last["state"] == "receiving" and last["sv_id"] == "MU01" and last["smp_synch"] == 2
    # The test publisher paces itself with Windows sleeps; its rate varies a few percent
    # under load. The 2 % rate rule is tested in test_sv_supervision.py.
    assert 3600 < last["rate"] < 4400 and last["lost"] == 0 and last["data_size"] == 64
    values = {item["name"]: item for item in last["values"]}
    assert values["IA"]["rms"] == pytest.approx(1.0, abs=0.005)
    assert values["VA"]["rms"] == pytest.approx(63.5, abs=0.1)
    assert values["IB"]["angle"] - values["IA"]["angle"] == pytest.approx(-120.0, abs=1.0)


def test_event_rate_is_decimated():
    reports, everything, elapsed = listen(3)
    assert len(everything) <= 2 * elapsed + 4
    assert reports[-1]["asdus"] > 4000  # thousands of frames, only a handful of events


def test_sample_loss_counts_gaps_and_wrap():
    reports, _, _ = listen(3, skip_every=400)
    last = reports[-1]
    # 3 s at 4000/s crosses the smpCnt wrap (4000) at least twice; a wrap is not loss or a restart.
    assert last["restarts"] == 0
    assert last["lost"] == pytest.approx(last["asdus"] / 399, abs=3)


def test_simulate_bit_and_unknown_stream_are_seen():
    other = {"mac": "01-0C-CD-04-00-22", "app_id": 0x4002, "sv_id": "OTHER"}
    reports, events, _ = listen(3, other, simulate=True)
    assert reports[-1]["simulate"] is True and reports[-1]["state"] == "receiving"
    unknown = [event for event in events if event.get("event") == "sv_unknown"]
    assert any(item["sv_id"] == "OTHER" and item["app_id"] == 0x4002 for item in unknown)
