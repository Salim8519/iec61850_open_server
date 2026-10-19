"""Write generated/plan.json: adapter normalisation, IP Helper words, address plans and change plans.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/netcfg/testdata/make_plan.py
Inputs are fakes (no Windows call); outputs are the frozen Python results as exact JSON text.
"""
import random
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vector_common import GATEWAY, GUID, OTHER_GUID, PROGRAM, adapter, dumps, outcome, write  # noqa: E402
from studio_app.networking import adapters, ip_helper  # noqa: E402
from studio_app.networking.address_plan import on_link, plan_addresses, validate_manual  # noqa: E402
from studio_app.networking.change_plan import (apply_command, build_plan, describe, gateway_plan,  # noqa: E402
                                               remove_command)

RAWS = [
    [{"guid": "{b}", "name": "Wi-Fi", "description": "Wireless LAN", "status": "Up", "addresses": []},
     {"guid": "{c}", "name": "VMnet1", "description": "VMware Virtual Ethernet", "status": "Up", "addresses": []},
     {"guid": "{a}", "name": "Ethernet", "description": "Realtek GbE", "status": "Disconnected",
      "addresses": [{"ip": "192.168.40.111", "prefix": 24, "origin": "Manual", "skip_as_source": False}]}],
    {"guid": "x", "name": "Ethernet", "description": "NIC", "status": "Up", "addresses": {"ip": "10.0.0.5", "prefix": 8}},
    [{"guid": "1a2b", "name": n, "description": d, "status": s, "index": i, "mac": "00-11", "speed": "1 Gbps",
      "default_route": r, "if_type": 6, "addresses": [{"ip": "10.1.0.%d" % i, "prefix": 16, "skip_as_source": i % 2,
                                                       "origin": "Dhcp"}]}
     for i, (n, d, s, r) in enumerate([
         ("Loop", "KM-TEST Loopback Adapter", "Up", False), ("vEthernet (WSL)", "Hyper-V Virtual Ethernet", "Up", 0),
         ("ZeroTier", "ZeroTier Virtual Port", "Down", True), ("BT", "Bluetooth Device (PAN)", "Disconnected", 1),
         ("ethernet 2", "Intel(R) Ethernet", "Up", True), ("Ethernet", "Intel(R) Ethernet", "Up", False),
         ("WLAN", "802.11ac card", "Up", None), ("Tap", "TAP-Windows Adapter V9", "Up", []),
         ("Other", "Mystery", "Disconnected", False), ("alpha", "Realtek", "Disconnected", False)], start=1)],
    [], None, "", '{"guid": "{ab}", "name": "One"}', "[]",
]
SPEEDS = [0, 1, 999, 1000, 1500, 999999, 10 ** 6, 100 * 10 ** 6, 866700000, 10 ** 9, 1200000000, 2500000000,
          1234567890, 10 ** 15, 123456789012345678, 2 ** 63 - 1, 2 ** 63, 2 ** 64 - 1]
LISTED = [(6, "Realtek Gaming GbE Family Controller"), (71, "MediaTek Wi-Fi 6 MT7921 Wireless LAN Card"),
          (24, "Software Loopback Interface 1"), (131, "Teredo Tunneling Pseudo-Interface"),
          (71, "Microsoft Wi-Fi Direct Virtual Adapter #2"), (23, "WAN Miniport (PPTP)"),
          (6, "Microsoft Kernel Debug Network Adapter"), (6, None), (53, "")]
MANUAL = [("192.168.1.21", 24), ("169.254.1.1", 16), ("192.168.1.0", 24), ("224.0.0.1", 24), ("x", 24),
          ("192.168.1.255", 24), ("10.0.0.1", 8), ("10.0.0.1", 7), ("10.0.0.1", 30), ("10.0.0.2", 30),
          ("10.0.0.1", 31), ("127.0.0.5", 24), ("192.168.1.21", "24"), ("192.168.1.21", "255.255.255.0"),
          ("192.168.1.21", "0.0.0.255"), ("192.168.1.21", "255.0.255.0"), ("192.168.01.21", 24),
          ("192.168.1.21/24", 24), (None, 24), ("192.168.1.21", None), (" 192.168.1.21", 24),
          ("192.168.1.21", "024"), ("192.168.1.21", 24.0), (True, 24), (3232235797, 24), ("", ""),
          ("1.2.3.4", "+24"), ("1.2.3.4", "２４"), ("0.0.0.1", 8), ("255.255.255.254", 8), ("1.2.3.4", 0),
          ("192.168.1.21", "255.255.255.252"), ("192.168.1.21", "255.255.255.255"), ("1.2.3", 24),
          ("1.2.3.4.5", 24), ("1.2.3.256", 24), ("192.168.1.21", "33"), ("172.16.5.4", "000000000000012")]
SUBNETS = [None, "", "255.255.255.0", "255.255.0.0", "0.0.0.255", "24", "16", "255.0.255.0", "abc", "255.255.255.252"]


def random_ied(rng, index):
    roll = rng.random()
    if roll < 0.15:
        address = None
    elif roll < 0.6:
        address = {"ip": "192.168.50.%d" % rng.choice([10, 21, 22, 30, 1, 255, 0, rng.randint(2, 254)]),
                   "subnet": rng.choice(SUBNETS)}
    elif roll < 0.8:
        address = {"ip": "10.0.%d.%d" % (rng.randint(0, 3), rng.randint(1, 30)), "subnet": rng.choice(SUBNETS)}
    else:
        address = {"ip": rng.choice(["", "999.1.1.1", "172.16.0.5", "169.254.3.3"]), "subnet": rng.choice(SUBNETS)}
    return {"id": "ied-%d" % (index % 5), "name": "IED%d" % index, "scl_address": address}


def random_adapter(rng, guid, name):
    choices = [("192.168.50.10", 24), ("192.168.50.30", 24), ("10.0.0.1", 16), ("169.254.7.7", 16),
               ("172.16.0.1", 30), ("172.16.9.9", 31), ("192.168.50.21", 24), ("10.0.2.5", 22)]
    picked = rng.sample(choices, rng.randint(0, 3))
    return adapter(guid=guid, name=name, index=rng.randint(1, 30), addresses=picked)


def address_cases(rng):
    cases = []
    for _ in range(160):
        lab = random_adapter(rng, GUID, "Lab NIC")
        others = [lab] + [random_adapter(rng, "{%d}" % n, "Other %d" % n) for n in range(rng.randint(0, 2))]
        ieds = [random_ied(rng, n) for n in range(rng.randint(1, 6))]
        strategy = rng.choice(["scd", "adapter", "adapter", "bogus"])
        avoid = sorted({"192.168.50.%d" % rng.randint(18, 26) for _ in range(rng.randint(0, 4))})
        cases.append({"ieds": ieds, "adapter": lab, "all": others, "strategy": strategy, "avoid": avoid,
                      **outcome(lambda: plan_addresses(ieds, lab, others, strategy, set(avoid)))})
    return cases


def item_texts(items):
    return [{name: outcome(lambda: function(item), str) for name, function in (
        ("describe", describe), ("apply", apply_command), ("remove", remove_command))} for item in items]


def build_cases(rng):
    programs = [PROGRAM, r"C:\Program Files\O'Brien\studio_ied.exe", "C:\\Ström\\studio_ied.exe", "", 'C:\\a"b\\x.exe']
    cases = []
    for _ in range(80):
        lab = random_adapter(rng, rng.choice([GUID, OTHER_GUID]), rng.choice(["Lab NIC", "Ethernet 'Lab'"]))
        rows = [{"name": "IED%d" % n, "proposed": rng.choice(["", "192.168.50.10", "192.168.50.%d" % rng.randint(20, 40),
                                                              "10.0.0.%d" % rng.randint(2, 9)]),
                 "prefix": rng.choice([24, 16, None]),
                 "status": rng.choice(["ok", "ok", "already_on_adapter", "duplicate", "missing_scd_address"])}
                for n in range(rng.randint(0, 5))]
        ports = sorted({rng.choice([102, 102, 2102, 10102, 1]) for _ in range(rng.randint(1, 3))})
        program = rng.choice(programs)
        first = build_plan(lab, rows, set(ports), program)
        existing = [item["name"] for item in first["items"] if item["kind"] == "firewall" and rng.random() < 0.5]
        plan = build_plan(lab, rows, set(ports), program, existing)
        cases.append({"adapter": lab, "rows": rows, "ports": ports, "program": program, "existing": existing,
                      "out": dumps(plan), "texts": item_texts(plan["items"])})
    return cases


def gateway_cases():
    block = {"name": r"TCP Query User{CD50883C-152F-4A3A-864C-00A52154808F}c:\studio\native\studio_gateway.exe",
             "display": "studio_gateway.exe"}
    cases = []
    for program in (GATEWAY, GATEWAY.upper(), r"D:\Ä\studio_gateway.exe", "x'y"):
        name = gateway_plan(program)["items"][0]["name"]
        for existing, blocking in (([], []), ([name], []), ([], [block]), (["other"], [block, {"name": "UDP x"}])):
            plan = gateway_plan(program, existing, blocking)
            cases.append({"program": program, "existing": existing,
                          "blocking": [{"name": r["name"], "display": r.get("display", "")} for r in blocking],
                          "out": dumps(plan), "texts": item_texts(plan["items"])})
    return cases


def main():
    rng = random.Random(61850)
    write("plan.json", {
        "normalize": [{"raw": raw, **outcome(lambda: adapters.normalize(raw))} for raw in RAWS],
        "status": [[n, ip_helper.status_word(n)] for n in range(-1, 9)],
        "speed": [[str(n), ip_helper.speed_words(n)] for n in SPEEDS],
        "mac": [[list(raw), length, ip_helper.mac_words(raw, length)] for raw, length in (
            (bytes([4, 66, 26, 134, 85, 145, 0, 0]), 6), (bytes(8), 0),
            (bytes([255] * 8), 8), (bytes([1, 2, 3, 4, 5, 6, 7, 8]), 20))],
        "listed": [[t, d, ip_helper.is_listed(t, d)] for t, d in LISTED],
        "manual": [{"address": a, "prefix": p, **outcome(lambda: list(validate_manual(a, p)))} for a, p in MANUAL],
        "on_link": [{"address": a, "adapter": ad, **outcome(lambda: on_link(a, ad))} for a, ad in (
            ("192.168.50.77", adapter()), ("192.168.51.1", adapter()), ("169.254.1.1", adapter(addresses=[("169.254.7.7", 16)])),
            ("10.0.3.255", adapter(addresses=[("10.0.0.1", 22)])), ("x", adapter()), ("x", adapter(addresses=[])))],
        "address_plan": address_cases(rng),
        "build_plan": build_cases(rng),
        "gateway_plan": gateway_cases(),
    })


if __name__ == "__main__":
    main()
