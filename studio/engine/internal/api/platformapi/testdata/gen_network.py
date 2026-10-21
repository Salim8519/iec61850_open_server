"""Lab-network and gateway-firewall steps for http_gen.py (fake adapters and helper only).

"$fp", "$clean" and "$gw" are the fingerprints of the last plan, clean-up and gateway
previews; "$rules" and "$gwrules" the firewall rule names those previews listed. The
generator records the resolved values, so the Go replay sends literal requests.
"""
from gen_license import op, req

GUID = "{11111111-1111-1111-1111-111111111111}"
OTHER_GUID = "{22222222-2222-2222-2222-222222222222}"
PROGRAM = r"C:\Studio\native\studio_ied.exe"
GATEWAY = r"C:\Studio\native\studio_gateway.exe"
BLOCK = {"name": r"TCP Query User{CD50883C-152F-4A3A-864C-00A52154808F}c:\studio\native\studio_gateway.exe",
         "display": "studio_gateway.exe"}


def adapter(guid=GUID, name="Lab NIC", index=7, addresses=(("192.168.50.10", 24),), kind="ethernet"):
    return {"guid": guid, "name": name, "description": "USB Ethernet", "status": "Up", "index": index, "mac": "",
            "speed": "1 Gbps", "kind": kind, "default_route": False,
            "addresses": [{"ip": ip, "prefix": prefix, "skip_as_source": False, "origin": "Manual"}
                          for ip, prefix in addresses]}


LAB = adapter()
WIFI = adapter(guid=OTHER_GUID, name="Wi-Fi", index=3, addresses=[("10.0.0.5", 24)], kind="wifi")
BAY = {"id": "bay", "name": "Bay", "role": "simulated", "scl_address": {"ip": "192.168.50.21", "subnet": "255.255.255.0"},
       "ip": "127.50.0.1", "port": 102, "source": "<SCL/>"}
B2 = {"id": "b2", "name": "B2", "role": "simulated", "scl_address": None, "ip": "127.50.0.2", "port": 2102,
      "source": "<SCL/>"}
REAL = {"id": "r1", "name": "Real", "role": "real", "scl_address": {"ip": "192.168.50.40"}, "ip": "127.50.0.3",
        "port": 102, "source": "<SCL/>"}
DEFAULT_ENV = {"adapters": [LAB, WIFI], "rules": [], "probe": [], "elevate": "ok", "gateway_rules": [],
               "blocking": [], "state": {"gateway": [], "disabled": []}}


def net(key, value):
    return op("net", key=key, value=value)


def network_steps():
    put, preview, apply = "/network", "/network/plan/preview", "/network/plan/apply"
    clean, cleanup, firewall = "/network/cleanup/preview", "/network/cleanup/apply", "/network/gateway-firewall"
    return [
        req("GET", "/network"), req("GET", "/network/adapters"), req("GET", "/network/adapters?refresh=1"),
        req("GET", "/network/adapters?refresh=true"), req("GET", "/network/adapters?refresh=1&refresh=0"),
        req("PUT", put), req("PUT", put, {}), req("PUT", put, {"nominal_hz": 60}), req("PUT", put, {"nominal_hz": 55}),
        req("PUT", put, {"mode": "bogus"}), req("PUT", put, {"mode": "lab"}), req("PUT", put, [1]),
        req("PUT", put, "x"), req("PUT", put, 5), req("PUT", put, raw="{", ctype="application/json"),
        req("PUT", put, {"nominal_hz": 50}), req("GET", "/network"),
        req("POST", preview), req("POST", preview, {"adapter_guid": GUID}),
        op("ieds", value=[BAY, {**B2, "ip": BAY["ip"], "port": 102}, REAL]),
        req("PUT", put, {"nominal_hz": 60}), req("GET", "/network/adapters?refresh=%FF"),
        op("ieds", value=[BAY, B2, REAL]),
        req("PUT", put, {"mode": "lab", "adapter_guid": GUID, "adapter_label": "Lab NIC"}),
        req("PUT", put, {"goose_reach": "network"}), req("PUT", put, {"mode": "loopback", "nominal_hz": 60}),
        req("PUT", put, {"mode": "loopback", "nominal_hz": 50}), req("GET", "/network"),
        op("runtime", state="running"),
        req("PUT", put, {"nominal_hz": 60}), req("POST", preview, {"adapter_guid": GUID}),
        req("POST", apply, {"fingerprint": "0" * 64, "confirm": True}), req("POST", cleanup, {"confirm": True}),
        req("GET", clean), req("GET", firewall),
        op("runtime", state="stopped"),
        req("POST", preview, {"adapter_guid": OTHER_GUID, "strategy": "scd"}),
        req("POST", preview, {"adapter_guid": GUID, "strategy": 5}),
        req("POST", preview, {"adapter_guid": GUID, "strategy": [1]}),
        req("POST", preview, {"adapter_guid": None, "strategy": [1]}),
        req("POST", preview, {"adapter_guid": GUID, "strategy": "bogus"}),
        req("POST", preview, [1]), req("POST", preview, "x"), req("POST", preview, 0),
        req("POST", preview, {"adapter_guid": GUID, "manual": {"b2": {"ip": "192.168.50.77", "prefix": 24}}}),
        req("POST", preview, {"adapter_guid": GUID.lower(), "strategy": "adapter"}),
        req("POST", apply), req("POST", apply, {"fingerprint": "$fp"}),
        req("POST", apply, {"fingerprint": "$fp", "confirm": "yes"}),
        req("POST", apply, {"fingerprint": "0" * 64, "confirm": True}),
        net("elevate", "cancel"), req("POST", apply, {"fingerprint": "$fp", "confirm": True}),
        net("elevate", "runtime"), req("POST", apply, {"fingerprint": "$fp", "confirm": True}),
        net("elevate", "refused"), req("POST", apply, {"fingerprint": "$fp", "confirm": True}),
        net("elevate", "none"), req("POST", apply, {"fingerprint": "$fp", "confirm": True}),
        net("elevate", "fail"), req("POST", apply, {"fingerprint": "$fp", "confirm": True}),
        req("GET", "/network"), req("GET", clean),
        net("elevate", "ok"), req("POST", apply, {"fingerprint": "$fp", "confirm": True}),
        req("GET", "/network"), req("PUT", put, {"nominal_hz": 60}), req("GET", "/network"),
        net("rules", "$rules"),
        net("adapters", [adapter(addresses=[("192.168.50.10", 24), ("192.168.50.21", 24), ("192.168.50.22", 24)]),
                         WIFI]),
        op("tick", seconds=11),
        req("GET", "/network/adapters"), req("GET", clean), req("POST", cleanup),
        req("POST", cleanup, {"fingerprint": "0" * 64, "confirm": True}),
        req("POST", cleanup, {"fingerprint": "$clean", "confirm": 1}),
        req("POST", cleanup, [1]),
        req("POST", cleanup, {"fingerprint": "$clean", "confirm": True}), req("GET", "/network"), req("GET", clean),
        req("GET", firewall), req("POST", firewall), req("POST", firewall, {"fingerprint": "$gw"}),
        req("POST", firewall, {"fingerprint": "0" * 64, "confirm": True}), req("POST", firewall, "x"),
        net("elevate", "cancel"), req("POST", firewall, {"fingerprint": "$gw", "confirm": True}),
        net("elevate", "ok"), req("POST", firewall, {"fingerprint": "$gw", "confirm": True}),
        net("gateway_rules", "$gwrules"), net("blocking", [BLOCK]), req("GET", firewall),
        req("POST", firewall, {"fingerprint": "$gw", "confirm": True}),
        net("blocking", "!RuntimeError:netsh failed"), req("GET", firewall),
        net("gateway_rules", "!ValueError:bad rule list"), req("GET", firewall),
        net("adapters", "!RuntimeError:The adapter list could not be read"),
        req("GET", "/network/adapters?refresh=1"), req("POST", preview, {"adapter_guid": GUID}),
        net("adapters", "!KeyError:guid"), req("GET", "/network/adapters?refresh=1"),
        net("adapters", "!ValueError:odd adapter"), req("GET", "/network/adapters?refresh=1"),
    ]
