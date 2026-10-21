"""Smart guard on a real adapter: conflicts, reflections, no wire-to-wire (GOOSE-21, GOOSE-22).

Needs Npcap and STUDIO_GOOSE_ADAPTER. npcap_peer plays the real device with the same gocbRef.
"""
import time

import pytest
from engine_wire_rig import WireRig, from_source
from goose_engine_rig import READY
from npcap_peer import ADAPTER, AVAILABLE, Peer, goose, mac_bytes

pytestmark = [pytest.mark.skipif(not READY, reason="Native build and GOOSE engine required"),
              pytest.mark.skipif(not (AVAILABLE and ADAPTER), reason="Set STUDIO_GOOSE_ADAPTER and install Npcap")]
REF = "SIM1LD0/LLN0$GO$gcb1"
DEVICE = mac_bytes("00-30-A7-10-20-30")


@pytest.fixture
def shared():
    rig, peer = WireRig({"SIM1": [REF]}), Peer()
    assert rig.wait_sharing(), "sharing did not start"
    yield rig, peer
    rig.close()
    peer.close()


def send_simulated(rig, count):
    for _ in range(count):
        rig.publish(goose(rig.macs["SIM1"], REF))
        time.sleep(0.01)


def sent_outside(rig, peer, count=20):
    peer.capture(0.2)
    send_simulated(rig, count)
    return len(from_source(peer.capture(0.6), rig.macs["SIM1"]))


def test_conflict_holds_the_ied_until_stop_or_continue(shared):
    rig, peer = shared
    assert sent_outside(rig, peer) >= 15
    rig.inside(0.3)
    peer.send(goose(DEVICE, REF))
    conflict = rig.event("goose_conflict", ied_id="SIM1")
    assert conflict and conflict["gocb_ref"] == REF and conflict["src_mac"] == "00-30-A7-10-20-30"
    assert not [frame for origin, frame in rig.inside(0.5) if origin == 1], "the real copy was delivered inside"
    assert rig.engine.request("stats")["gates"]["SIM1"] == "held"
    assert sent_outside(rig, peer) == 0, "a held IED still went outside"
    assert rig.engine.request("gate", ied_id="SIM1", choice="stop")["gate"] == "blocked"
    assert sent_outside(rig, peer) == 0
    assert rig.engine.request("gate", ied_id="SIM1", choice="continue")["gate"] == "forced"
    assert sent_outside(rig, peer) >= 15
    peer.send(goose(DEVICE, REF))
    assert not [frame for origin, frame in rig.inside(0.5) if origin == 1]


def test_inside_traffic_continues_while_held(shared):
    rig, peer = shared
    peer.send(goose(DEVICE, REF))
    assert rig.event("goose_conflict", ied_id="SIM1")
    rig.inside(0.2)
    send_simulated(rig, 20)
    assert sum(1 for origin, _ in rig.inside(0.6) if origin == 0) >= 15


def test_reflections_are_dropped_and_wire_frames_are_never_resent(shared):
    rig, peer = shared
    rig.inside(0.3)
    peer.send(goose(rig.macs["SIM1"], REF))  # our own frame coming back, e.g. from a looped switch
    assert rig.inside(0.5) == [], "a reflection of our own frame was delivered"
    assert rig.engine.request("stats")["refused"].get("reflection", 0) >= 1
    peer.capture(0.2)
    for _ in range(10):
        peer.send(goose(DEVICE, "RELAY1LD0/LLN0$GO$block"))
    heard = from_source(peer.capture(0.6), DEVICE)
    assert len(heard) <= 10, f"{len(heard)} copies of 10 device frames: the engine forwarded wire to wire"
    assert sum(1 for origin, frame in rig.inside(0.3) if origin == 1 and frame[6:12] == DEVICE) == 10
    assert rig.event("goose_conflict", seconds=0.3) is None
