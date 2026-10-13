"""Lab address proposals and plain conflict hints (NET-04)."""
import pytest
from studio_app.networking.address_plan import plan_addresses, validate_manual

LAB = {"guid": "{11111111-1111-1111-1111-111111111111}", "name": "Lab NIC", "index": 7,
       "addresses": [{"ip": "192.168.50.10", "prefix": 24}]}
WIFI = {"guid": "{22222222-2222-2222-2222-222222222222}", "name": "Wi-Fi", "index": 3,
        "addresses": [{"ip": "192.168.50.30", "prefix": 24}]}


def ied(name, ip=None, subnet="255.255.255.0"):
    return {"id": name, "name": name, "scl_address": {"ip": ip, "subnet": subnet} if ip else None}


def test_scd_address_preferred_inside_subnet():
    rows = plan_addresses([ied("A", "192.168.50.21"), ied("B", "10.0.0.5")], LAB, [LAB], "adapter")
    assert rows[0]["proposed"] == "192.168.50.21" and rows[0]["status"] == "ok"
    assert rows[1]["proposed"].startswith("192.168.50.") and rows[1]["proposed"] not in {"192.168.50.10", "192.168.50.21"}
    scd = plan_addresses([ied("A", "10.0.0.5", "255.255.0.0")], LAB, [LAB], "scd")[0]
    assert scd["proposed"] == "10.0.0.5" and scd["prefix"] == 16


def test_conflicts_are_detected_with_hints():
    rows = plan_addresses([ied("Own", "192.168.50.10"), ied("Other", "192.168.50.30"),
                           ied("First", "10.0.0.5"), ied("Second", "10.0.0.5"), ied("Missing")],
                          LAB, [LAB, WIFI], "scd")
    statuses = {row["name"]: row["status"] for row in rows}
    assert statuses == {"Own": "already_on_adapter", "Other": "used_by_other_adapter", "First": "ok",
                        "Second": "duplicate", "Missing": "missing_scd_address"}
    assert all(row["hint"] for row in rows if row["status"] != "ok")
    assert "Wi-Fi" in rows[1]["hint"] and "First" in rows[3]["hint"]


def test_manual_addresses_are_validated():
    assert validate_manual("192.168.1.21", 24) == ("192.168.1.21", 24)
    for address, prefix in (("169.254.1.1", 16), ("192.168.1.0", 24), ("224.0.0.1", 24), ("x", 24)):
        with pytest.raises(ValueError):
            validate_manual(address, prefix)
