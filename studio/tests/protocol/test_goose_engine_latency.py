"""Ten simulated IEDs through the GOOSE engine: every frame arrives, quickly (ADR-0073 benchmark).

Measures bus -> engine -> inbox time on loopback. The p50/p99 are printed for the change record.
"""
import statistics
import time

import pytest
from engine_wire_rig import WireRig
from goose_engine_rig import READY
from npcap_peer import goose

pytestmark = pytest.mark.skipif(not READY, reason="Native build and GOOSE engine required")
IEDS = {f"IED{index:02d}": [f"IED{index:02d}LD0/LLN0$GO$trip"] for index in range(1, 11)}
ROUNDS = 300


def test_ten_ieds_reach_the_inbox_without_loss(capsys):
    rig = WireRig(IEDS)
    try:
        rig.inbox.settimeout(0.5)
        delays = []
        for round_number in range(ROUNDS):
            for ied, refs in IEDS.items():
                started = time.perf_counter()
                rig.publish(goose(rig.macs[ied], refs[0], app_id=0x1000 + round_number % 256))
                data = rig.inbox.recv(65536)
                delays.append((time.perf_counter() - started) * 1000)
                assert data[0] == 0 and data[7:13] == rig.macs[ied]
        stats = rig.engine.request("stats")
    finally:
        rig.close()
    delays.sort()
    p50, p99 = statistics.median(delays), delays[int(len(delays) * 0.99) - 1]
    with capsys.disabled():
        print(f"\n10 IEDs x {ROUNDS} rounds: p50 {p50:.3f} ms, p99 {p99:.3f} ms, max {delays[-1]:.3f} ms")
    assert stats["delivered_app"] == len(IEDS) * ROUNDS and stats["bus_rejected"] == 0
    assert p99 <= 3.0, f"p99 {p99:.2f} ms is above the 3 ms target"
