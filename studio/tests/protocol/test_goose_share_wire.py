"""Inside never transmits; Share sends each IED with its own MAC after the check (GOOSE-18, GOOSE-20).

Needs Npcap and STUDIO_GOOSE_ADAPTER (a lab adapter GUID). The receiving side is npcap_peer,
an independent Npcap handle that shares no code with the engine.
"""
import time

import pytest
from engine_wire_rig import SHARE_CHECK_SECONDS, WireRig, from_source
from goose_engine_rig import READY
from npcap_peer import ADAPTER, AVAILABLE, Peer, goose, mac_bytes

pytestmark = [pytest.mark.skipif(not READY, reason="Native build and GOOSE engine required"),
              pytest.mark.skipif(not (AVAILABLE and ADAPTER), reason="Set STUDIO_GOOSE_ADAPTER and install Npcap")]
IEDS = {"IED_A": ["IED_ALD0/LLN0$GO$trip"], "IED_B": ["IED_BLD0/LLN0$GO$trip"]}
DEVICE = mac_bytes("00-30-A7-10-20-30")


def publish_all(rig, count):
    for _ in range(count):
        for ied, refs in IEDS.items():
            rig.publish(goose(rig.macs[ied], refs[0]))
        time.sleep(0.01)


@pytest.fixture
def rig_and_peer():
    rig, peer = WireRig(IEDS), Peer()
    yield rig, peer
    rig.close()
    peer.close()


def test_inside_listens_to_real_devices_but_never_sends(rig_and_peer):
    rig, peer = rig_and_peer
    assert rig.uplink(False)["share"] == "off"
    assert rig.event("goose_uplink", state="open"), "the engine did not open the adapter to listen"
    peer.capture(0.3)
    publish_all(rig, 50)
    heard = peer.capture(1.0)
    for mac in rig.macs.values():
        assert from_source(heard, mac) == [], "Inside mode put a simulated frame on the adapter"
    peer.send(goose(DEVICE, "RELAY1LD0/LLN0$GO$block"))
    delivered = rig.inside(1.0)
    assert any(origin == 1 and frame[6:12] == DEVICE for origin, frame in delivered)
    assert sum(1 for origin, _ in delivered if origin == 0) >= 100  # the IEDs still hear each other
    assert rig.engine.request("stats")["sent"] == 0


def test_share_waits_for_the_check_then_sends_each_ied_with_its_own_mac(rig_and_peer):
    rig, peer = rig_and_peer
    assert len(set(rig.macs.values())) == 2
    assert all(mac[0] & 0x02 and not mac[0] & 0x01 for mac in rig.macs.values())  # locally administered unicast
    started = time.monotonic()
    rig.uplink(True)
    assert rig.event("goose_share", state="checking")
    peer.capture(0.2)
    publish_all(rig, 20)
    early = peer.capture(0.5)
    assert all(from_source(early, mac) == [] for mac in rig.macs.values()), "sent during the network check"
    assert rig.event("goose_share", seconds=SHARE_CHECK_SECONDS + 5, state="sharing")
    assert time.monotonic() - started >= SHARE_CHECK_SECONDS - 0.2
    publish_all(rig, 30)
    heard = peer.capture(1.0)
    for ied, mac in rig.macs.items():
        frames = from_source(heard, mac)
        assert len(frames) >= 25, f"{ied}: only {len(frames)} of 30 frames reached the adapter"
        assert IEDS[ied][0].encode() in frames[0]
    rig.uplink(False)
    assert rig.event("goose_share", state="off")
    peer.capture(0.3)
    publish_all(rig, 20)
    assert all(from_source(peer.capture(0.5), mac) == [] for mac in rig.macs.values())
