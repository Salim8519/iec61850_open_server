"""Decoding rules for the IP Helper adapter listing and the shared normalisation."""
from studio_app.networking import adapters, ip_helper


def test_status_and_speed_use_get_netadapter_words():
    assert ip_helper.status_word(1) == "Up"
    assert ip_helper.status_word(2) == ip_helper.status_word(7) == "Disconnected"
    assert ip_helper.status_word(5) == "Down"
    assert ip_helper.speed_words(10 ** 9) == "1 Gbps"
    assert ip_helper.speed_words(100 * 10 ** 6) == "100 Mbps"
    assert ip_helper.speed_words(2 ** 64 - 1) == ip_helper.speed_words(0) == "0 bps"
    assert ip_helper.mac_words(bytes([4, 66, 26, 134, 85, 145, 0, 0]), 6) == "04-42-1A-86-55-91"


def test_loopback_tunnels_and_hidden_miniports_are_not_listed():
    assert ip_helper.is_listed(6, "Realtek Gaming GbE Family Controller")
    assert ip_helper.is_listed(71, "MediaTek Wi-Fi 6 MT7921 Wireless LAN Card")
    assert not ip_helper.is_listed(24, "Software Loopback Interface 1")
    assert not ip_helper.is_listed(131, "Teredo Tunneling Pseudo-Interface")
    assert not ip_helper.is_listed(71, "Microsoft Wi-Fi Direct Virtual Adapter #2")


def test_native_records_are_normalised_with_ethernet_first():
    raw = [{"guid": "{b}", "name": "Wi-Fi", "description": "Wireless LAN", "status": "Up", "addresses": []},
           {"guid": "{c}", "name": "VMnet1", "description": "VMware Virtual Ethernet", "status": "Up", "addresses": []},
           {"guid": "{a}", "name": "Ethernet", "description": "Realtek GbE", "status": "Disconnected",
            "addresses": [{"ip": "192.168.40.111", "prefix": 24, "origin": "Manual", "skip_as_source": False}]}]
    listed = adapters.list_adapters(native=lambda: raw, run=lambda script: "[]")
    assert [item["name"] for item in listed] == ["Ethernet", "Wi-Fi", "VMnet1"]
    assert listed[0]["guid"] == "{A}" and listed[0]["addresses"][0]["prefix"] == 24


def test_powershell_is_only_a_fallback():
    def broken():
        raise OSError(50, "not supported")
    fallback = '[{"guid": "x", "name": "Ethernet", "description": "NIC", "status": "Up", "addresses": {"ip": "10.0.0.5", "prefix": 8}}]'
    listed = adapters.list_adapters(native=broken, run=lambda script: fallback)
    assert listed[0]["guid"] == "{X}" and listed[0]["addresses"][0]["ip"] == "10.0.0.5"
