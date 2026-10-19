"""Where GOOSE goes: adapter auto-pick, risk, holds and status (GOOSE-23, GOOSE-24, ADR-0073)."""
from studio_app.domain.goose_reach import advanced_choices, pick_uplink, reach_status, share_hold, share_risk
from studio_app.domain.network_settings import validate_network
from studio_app.projects.ied_settings import _merged_network

LAB = "{15A6105F-A715-444F-9D7C-AA259A2EED8A}"
CABLE = "{7B29C0B1-72E8-4C7D-9FAA-6A695893211D}"


def adapter(guid, name, kind="ethernet", status="Up", default_route=False, description=""):
    return {"guid": guid, "name": name, "kind": kind, "status": status, "default_route": default_route,
            "description": description}


ADAPTERS = [
    adapter("{00000000-0000-0000-0000-00000000000A}", "Wi-Fi", "wifi", default_route=True),
    adapter("{00000000-0000-0000-0000-00000000000B}", "Npcap Loopback Adapter", description="Npcap Loopback Adapter"),
    adapter("{00000000-0000-0000-0000-00000000000C}", "Ethernet 2", status="Disconnected"),
    adapter(CABLE, "Ethernet", default_route=True),
    adapter("{00000000-0000-0000-0000-00000000000D}", "Ethernet 3"),
    adapter(LAB, "VMware VMnet1", "virtual"),
    adapter("{00000000-0000-0000-0000-00000000000E}", "ZeroTier", "vpn"),
]


def test_pick_uplink_prefers_advanced_then_lab_then_connected_cable_with_default_route():
    assert pick_uplink(ADAPTERS, {"mode": "loopback"})["guid"] == CABLE
    assert pick_uplink(ADAPTERS, {"mode": "lab", "adapter_guid": LAB})["guid"] == LAB
    chosen = pick_uplink(ADAPTERS, {"mode": "lab", "adapter_guid": LAB, "goose_uplink_guid": ADAPTERS[0]["guid"]})
    assert chosen["guid"] == ADAPTERS[0]["guid"] and chosen["auto"] is False
    gone = pick_uplink(ADAPTERS, {"goose_uplink_guid": "{00000000-0000-0000-0000-0000000000FF}"})
    assert gone["guid"] == "" and "not on this PC" in gone["reason"]


def test_pick_uplink_never_auto_picks_wifi_vpn_virtual_loopback_or_unplugged():
    risky = [item for item in ADAPTERS if item["guid"] not in {CABLE, "{00000000-0000-0000-0000-00000000000D}"}]
    picked = pick_uplink(risky, {"mode": "loopback"})
    assert picked["guid"] == "" and "No connected wired network adapter" in picked["reason"]
    names = [choice["label"] for choice in advanced_choices(ADAPTERS)]
    assert "Npcap Loopback Adapter" not in names and "ZeroTier" not in names
    assert next(c for c in advanced_choices(ADAPTERS) if c["label"] == "Wi-Fi")["warning"]


def test_mode_change_keeps_reach():
    shared = validate_network({"mode": "lab", "adapter_guid": LAB, "goose_reach": "outside",
                               "goose_uplink_guid": CABLE, "goose_share_guid": CABLE})
    back = validate_network(_merged_network(shared, {"mode": "loopback"}))
    assert (back["goose_reach"], back["goose_uplink_guid"], back["goose_share_guid"]) == ("outside", CABLE, CABLE)
    assert validate_network({})["goose_reach"] == "inside"


def test_share_risk_is_high_when_real_devices_may_act_on_it():
    uplink = pick_uplink(ADAPTERS, {"mode": "lab", "adapter_guid": LAB})
    lab = share_risk({"mode": "lab", "adapter_guid": LAB}, uplink, [], False)
    assert lab["level"] == "high" and "real network addresses" in lab["reasons"][0]
    cable = pick_uplink(ADAPTERS, {"mode": "loopback"})
    assert share_risk({"mode": "loopback"}, cable, [], False) == {"level": "warning", "reasons": []}
    assert share_risk({"mode": "loopback"}, cable, [], True)["level"] == "high"
    assert share_risk({"mode": "loopback"}, cable, [{"role": "real"}], False)["level"] == "high"


def test_sharing_is_held_until_confirmed_on_the_adapter_in_use():
    uplink = pick_uplink(ADAPTERS, {"mode": "loopback"})
    network = {"goose_reach": "outside", "goose_share_guid": CABLE}
    assert share_hold(network, uplink, True) == ""
    assert "Npcap" in share_hold(network, uplink, False)
    moved = share_hold({**network, "goose_share_guid": LAB}, uplink, True)
    assert "confirm this adapter" in moved
    assert share_hold({"goose_reach": "inside"}, uplink, False) == ""


def test_status_line_names_inside_listening_checking_sharing_and_holds():
    uplink = pick_uplink(ADAPTERS, {"mode": "loopback"})
    assert reach_status({}, uplink, "", {"uplink": "closed"})[0] == "inside"
    assert reach_status({}, uplink, "", {"uplink": "open", "share": "off"})[0] == "listening"
    assert reach_status({}, uplink, "", {"uplink": "open", "share": "checking"})[0] == "checking"
    state, message = reach_status({}, uplink, "", {"uplink": "open", "share": "sharing"})
    assert state == "sharing" and "Ethernet" in message and "smart guard" in message
    assert reach_status({}, uplink, "Paused", {"share": "sharing"}) == ("share_held", "Paused")


def test_lab_projects_saved_before_reach_get_a_one_time_notice():
    from studio_app.projects.goose_reach import lab_notice
    lab = validate_network({"mode": "lab", "adapter_guid": LAB})
    assert lab_notice({"mode": "lab", "adapter_guid": LAB}, lab)
    assert not lab_notice({"mode": "lab", "adapter_guid": LAB, "goose_reach": "inside"}, lab)
    assert not lab_notice({"mode": "loopback"}, validate_network({}))
