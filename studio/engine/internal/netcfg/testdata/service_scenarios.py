"""Scenario definitions for make_service.py (lab-network service and gateway firewall flows).

Each scenario has IEDs, the fake Windows state ("env") and steps. Steps:
  ["preview", guid, strategy, manual]  ["apply", fingerprint, confirm]  ["adapters", refresh]
  ["cleanup_preview"]  ["cleanup", fingerprint, confirm]  ["gw_preview"]  ["gw_apply", fingerprint, confirm]
  ["set", key, value]  (change the fake Windows state)  ["tick", seconds]  (advance the clock)
"$fp" means the fingerprint of the last preview of the same kind ("$gw", "$clean").
"""
from vector_common import GUID, OTHER_GUID, adapter

BAY = {"id": "bay", "name": "Bay", "scl_address": {"ip": "192.168.50.21", "subnet": "255.255.255.0"}, "port": 102}
LAB = adapter()
WIFI = adapter(guid=OTHER_GUID, name="Wi-Fi", index=3, addresses=[("192.168.50.30", 24)], description="Wireless",
               kind="wifi")
BLOCK = {"name": r"TCP Query User{CD50883C-152F-4A3A-864C-00A52154808F}c:\studio\native\studio_gateway.exe",
         "display": "studio_gateway.exe"}
MANY = [BAY, {"id": "b2", "name": "Feeder 2", "scl_address": {"ip": "192.168.50.30", "subnet": "255.255.255.0"},
              "port": 2102},
        {"id": "b3", "name": "Feeder 3", "scl_address": None, "port": 102},
        {"id": "b4", "name": "Tie", "scl_address": {"ip": "192.168.50.10", "subnet": "255.255.255.0"}},
        {"id": "b5", "name": "Real one", "role": "real", "scl_address": {"ip": "192.168.50.40"}},
        {"id": "b6", "name": "Copy", "scl_address": {"ip": "192.168.50.21", "subnet": "255.255.0.0"}},
        {"id": "b7", "name": "Far", "scl_address": {"ip": "10.9.9.9", "subnet": "255.255.255.0"}}]

DEFAULT_ENV = {"adapters": [LAB], "rules": [], "probe": [], "elevate": "ok", "gateway_rules": [], "blocking": [],
               "state": {"gateway": [], "disabled": []}}

SCENARIOS = {
    "apply_flow": {"ieds": [BAY], "steps": [
        ["preview", GUID, "adapter", None], ["apply", "$fp", None], ["apply", "0" * 64, True],
        ["apply", "$fp", True], ["set", "rules", "$rule"],
        ["set", "adapters", [adapter(addresses=[("192.168.50.10", 24), ("192.168.50.21", 24)])]], ["tick", 11],
        ["cleanup_preview"], ["cleanup", "0" * 64, True], ["cleanup", "$clean", "yes"],
        ["cleanup", "$clean", True], ["cleanup_preview"], ["cleanup", "$clean", True]]},
    "repick": {"ieds": [{**BAY, "scl_address": None}], "env": {"probe": ["192.168.50.21", "192.168.50.22"]}, "steps": [
        ["preview", GUID, "adapter", None],
        ["preview", GUID, "adapter", {"bay": {"ip": "192.168.50.21", "prefix": 24}}], ["apply", "$fp", True],
        ["set", "probe", ["192.168.50.%d" % n for n in range(21, 40)]], ["preview", GUID, "adapter", {}]]},
    "scd_many": {"ieds": MANY, "env": {"adapters": [LAB, WIFI], "probe": ["192.168.50.21"]}, "steps": [
        ["preview", GUID, "scd", None], ["preview", GUID.lower(), "adapter", None],
        ["preview", GUID, "scd", {"b3": {"ip": "192.168.50.77", "prefix": 24}, "b7": {"ip": "10.9.9.9", "prefix": 33},
                                  "zz": {"ip": "1.1.1.1"}, "b2": {}}],
        ["preview", GUID, "scd", {"b3": {"ip": "192.168.50.77", "prefix": "255.255.255.0"}}],
        ["preview", GUID, "bogus", None], ["preview", None, "scd", None], ["preview", OTHER_GUID, "scd", None]]},
    "nothing_to_publish": {"ieds": [{**BAY, "role": "real"}, {**BAY, "id": "x", "role": "off"}], "steps": [
        ["preview", GUID, "scd", None]]},
    "elevation_modes": {"ieds": [BAY, {**BAY, "id": "b2", "name": "B2", "scl_address": None, "port": 2102}], "steps": [
        ["preview", GUID, "adapter", None], ["set", "elevate", "cancel"], ["apply", "$fp", True],
        ["set", "elevate", "refused"], ["apply", "$fp", True], ["set", "elevate", "none"], ["apply", "$fp", True],
        ["set", "elevate", "runtime"], ["apply", "$fp", True], ["set", "elevate", "no_ok"], ["apply", "$fp", True],
        ["set", "elevate", "partial"], ["apply", "$fp", True], ["set", "elevate", "fail"], ["apply", "$fp", True],
        ["cleanup_preview"], ["set", "elevate", "ok"], ["apply", "$fp", True], ["cleanup_preview"]]},
    "network_changed": {"ieds": [BAY], "steps": [
        ["preview", GUID, "adapter", None],
        ["set", "adapters", [adapter(addresses=[("192.168.50.10", 24), ("192.168.50.21", 24)])]],
        ["apply", "$fp", True], ["preview", GUID, "adapter", None], ["apply", "$fp", True]]},
    "existing_rule": {"ieds": [BAY], "env": {"adapters": [adapter(addresses=[("192.168.50.10", 24), ("192.168.50.21", 24)])]},
                      "steps": [["preview", GUID, "scd", None], ["set", "rules", "$rule"], ["preview", GUID, "scd", None],
                                ["apply", "$fp", True]]},
    "cache": {"ieds": [BAY], "steps": [
        ["adapters", False], ["set", "adapters", [WIFI]], ["tick", 5], ["adapters", False], ["preview", GUID, "scd", None],
        ["tick", 6], ["adapters", False], ["preview", GUID, "scd", None], ["adapters", True]]},
    "gateway": {"ieds": [BAY], "env": {"blocking": [BLOCK]}, "steps": [
        ["gw_preview"], ["gw_apply", "$gw", False], ["set", "blocking", []], ["gw_apply", "$gw", True],
        ["gw_preview"], ["gw_apply", "$gw", True], ["gw_apply", "nope", True], ["set", "blocking", [BLOCK]],
        ["gw_preview"], ["gw_apply", "$gw", True], ["set", "gateway_rules", "$gwrule"], ["gw_preview"],
        ["preview", GUID, "adapter", None], ["apply", "$fp", True], ["set", "state", {"gateway": ["$gwrule"], "disabled": [BLOCK["name"]]}],
        ["cleanup_preview"], ["set", "elevate", "fail"], ["cleanup", "$clean", True], ["cleanup_preview"],
        ["set", "elevate", "ok"], ["cleanup", "$clean", True], ["cleanup_preview"]]},
    "gateway_errors": {"ieds": [BAY], "steps": [
        ["set", "gateway_rules", "!RuntimeError:Windows Firewall could not be read: Access is denied."], ["gw_preview"],
        ["set", "gateway_rules", "!ValueError:bad json"], ["gw_preview"],
        ["set", "gateway_rules", "!KeyError:name"], ["gw_preview"]]},
}
