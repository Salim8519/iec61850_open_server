"""Network mode, per-IED settings, test mode, endpoint conflicts and GOOSE reach vectors."""

GUID = "{0123ABCD-0123-4567-89AB-0123456789AB}"
A, B, C = "a" * 32, "b" * 32, "c" * 32


def ied(key, name, ip, **extra):
    return {"id": key, "name": name, "ip": ip, "port": 102, "role": "simulated", "live": None, "lab_ip": "",
            "goose_publish": True, "test_mode": {"devices": {}, "simulation": False}, **extra}


def project(network=None, *ieds):
    return {"id": "f" * 32, "ieds": list(ieds) or [ied(A, "A", "127.50.0.2"), ied(B, "B", "127.50.0.3")],
            "network": network or {"mode": "loopback", "adapter_guid": "", "adapter_label": "", "nominal_hz": 50,
                                   "goose_reach": "inside", "goose_uplink_guid": "", "goose_share_guid": ""}}


NETWORKS = [None, {}, [], 0, "", [1], "x", 5, {"mode": "lab"}, {"mode": "lab", "adapter_guid": GUID},
            {"mode": "lab", "adapter_guid": GUID.lower()}, {"mode": "lab", "adapter_guid": GUID + "\n"},
            {"mode": "loopback", "adapter_guid": "x"}, {"mode": "LAB"}, {"mode": ["lab"]}, {"mode": None},
            {"nominal_hz": 60}, {"nominal_hz": 60.0}, {"nominal_hz": "50"}, {"nominal_hz": True}, {"nominal_hz": [50]},
            {"nominal_hz": 55}, {"goose_reach": "outside"}, {"goose_reach": ["inside"]}, {"goose_reach": None},
            {"goose_uplink_guid": GUID.lower()}, {"goose_share_guid": "nope"}, {"goose_uplink_guid": 5},
            {"adapter_label": "x" * 150}, {"adapter_label": 12.5}, {"adapter_label": None}, {"adapter_label": [1]},
            {"adapter_guid": "{0123ABCD-0123-4567-89AB-0123456789ﬀ}"}, {"adapter_guid": True},
            {"mode": "lab", "adapter_guid": GUID, "nominal_hz": 60, "goose_reach": "outside", "extra": 1}]
LIVES = [None, {}, [], "x", [1], {"access": "polling"}, {"access": "x"}, {"access": ["polling"]},
         {"poll_ms": 249}, {"poll_ms": 250}, {"poll_ms": 60001}, {"poll_ms": True}, {"poll_ms": 1000.0},
         {"ip": "192.168.1.20"}, {"ip": "127.0.0.1"}, {"ip": "0.0.0.0"}, {"ip": "224.0.0.1"}, {"ip": "239.255.255.255"},
         {"ip": "240.0.0.1"}, {"ip": "255.255.255.255"}, {"ip": "1.2.3"}, {"ip": "01.2.3.4"}, {"ip": "1.2.3.256"},
         {"ip": " 1.2.3.4"}, {"ip": "1.2.3.4/24"}, {"ip": "١.2.3.4"}, {"ip": "1..3.4"}, {"ip": 5}, {"ip": None},
         {"ip": ""}, {"ip": False}, {"ip": 0}, {"port": 0}, {"port": 65535}, {"port": "102"}, {"port": True},
         {"port": 10 ** 20}, {"ip": "x", "port": 0}]
TEST_MODES = [None, {}, [], "x", [1], 5, {"devices": None}, {"devices": []}, {"devices": [1]}, {"devices": "x"},
              {"devices": {"LD": 1}}, {"devices": {"LD": 5.0}}, {"devices": {"LD": 6}}, {"devices": {"LD": True}},
              {"devices": {"LD": "1"}}, {"devices": {"LD": [1]}}, {"devices": {"LD": 2.5}},
              {"devices": {str(i): 1 for i in range(101)}}, {"simulation": 1}, {"simulation": True},
              {"devices": {"A": 3, "B": 4}, "simulation": False, "x": 1}]
SETTINGS = [{}, {"role": "real"}, {"role": "off"}, {"role": "x"}, {"role": ["real"]}, {"role": None},
            {"goose_publish": 0}, {"goose_publish": False}, {"lab_ip": "10.0.0.5"}, {"lab_ip": "127.0.0.1"},
            {"lab_ip": None}, {"lab_ip": 7}, {"live": {"ip": "1.2.3.4", "access": "polling", "poll_ms": 500}},
            {"live": [1]}, {"test_mode": [1]}, {"test_mode": {"devices": {"LD": 9}}},
            {"role": "real", "lab_ip": "10.0.0.1", "goose_publish": False, "extra": True}]


def cases(fixtures):
    out = []

    def add(name, fn, **args):
        out.append({"name": name, "fn": fn, "args": args, "ieds": []})

    for index, value in enumerate(NETWORKS):
        add(f"network {index}", "validate_network", value=value)
    for index, value in enumerate(LIVES):
        add(f"live {index}", "validate_live", value=value)
    for index, value in enumerate(TEST_MODES):
        add(f"test mode {index}", "validate_test_mode", value=value)
    for index, value in enumerate(SETTINGS):
        add(f"ied settings {index}", "validate_ied_settings", value=value)
    lab = {"mode": "lab", "adapter_guid": GUID, "adapter_label": "Eth", "nominal_hz": 60, "goose_reach": "outside",
           "goose_uplink_guid": GUID, "goose_share_guid": ""}
    for index, value in enumerate([{}, None, {"nominal_hz": 60}, {"mode": "lab", "adapter_guid": GUID},
                                   {"mode": "loopback"}, {"mode": "x"}, ["nominal_hz"], ["mode"], "nominal_hz",
                                   [[1]], 5, {"nominal_hz": 61}, {"goose_reach": "inside"}]):
        add(f"update network {index}", "update_network", project=project(lab), value=value)
    clash = project(lab, ied(A, "A", "127.50.0.2", lab_ip="10.0.0.5"), ied(B, "B", "127.50.0.3", lab_ip="10.0.0.5"))
    add("update network clash", "update_network", project=clash, value={"mode": "lab", "adapter_guid": GUID})
    add("update network no clash", "update_network", project=clash, value={"mode": "loopback"})
    changes = [None, {}, [], {"name": "x"}, {"role": "real"}, {"role": "x"}, {"port": 103}, {"port": "103"},
               {"port": 0}, {"port": True}, {"lab_ip": "10.0.0.9"}, {"lab_ip": "bad"}, {"live": {"ip": "1.2.3.4"}},
               {"live": None}, {"live": [1]}, {"live": {"port": 0}}, {"goose_publish": "yes"},
               {"role": "simulated", "port": 102}, {"ip": "1.1.1.1"}]
    for index, value in enumerate(changes):
        add(f"update ied {index}", "update_ied", project=project(), key=A, changes=value)
    add("update ied unknown", "update_ied", project=project(), key="zz", changes={"role": "off"})
    add("update ied port clash", "update_ied", project=project(None, ied(A, "A", "127.50.0.2"),
                                                                   ied(B, "B", "127.50.0.2", port=103)),
        key=A, changes={"port": 103})
    real = project(None, ied(A, "A", "127.50.0.2", role="real", live={"ip": "", "port": 102, "access": "polling",
                                                                     "poll_ms": 1000}, scl_address={"ip": "10.1.1.1"}),
                   ied(B, "B", "127.50.0.3", role="real", live={"ip": "10.1.1.1", "port": 102}))
    add("update ied real clash", "update_ied", project=real, key=A, changes={"role": "real"})
    groups = [[ied(A, "A", "127.50.0.2"), ied(B, "B", "127.50.0.2"), ied(C, "C", "127.50.0.2")],
              [ied(A, "A", "127.50.0.2", port=102.0), ied(B, "", "127.50.0.2"), ied(C, "C", "127.50.0.2")],
              [ied(A, "A", "x", role="off"), ied(B, "B", "x", role="off")],
              [ied(A, "A", "1", role="real"), ied(B, "B", "1", role="real", scl_address={"ip": "9.9.9.9"})],
              [{"name": "A", "ip": "i"}, {"name": "B", "ip": "i", "port": 102}],
              [{"name": "A", "ip": [1]}], [ied(A, "A", "127.50.0.2", lab_ip="10.0.0.1"), ied(B, "B", "10.0.0.1")]]
    for index, ieds in enumerate(groups):
        for mode in ("loopback", "lab"):
            add(f"conflicts {index} {mode}", "endpoint_conflicts", ieds=ieds, network={"mode": mode})
    for index, (reach, uplink, share) in enumerate((("outside", GUID, GUID), ("inside", None, ""), ("x", "", ""),
                                                    ("outside", "bad", ""), (["inside"], "", ""))):
        add(f"save reach {index}", "save_reach", project=project(lab), reach=reach, uplink=uplink, share=share)
    return out
