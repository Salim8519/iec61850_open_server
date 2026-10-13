"""Compare a real IED's discovered data model with its SCD engineering model."""
from .live_value import describe_mms, type_matches

HINTS = {
    "missing_on_device": "The SCD has this point but the device does not. The device may run a "
                         "different configuration (CID) than the SCD expects.",
    "extra_on_device": "The device has this point but the SCD does not. Usually harmless, but it can "
                       "mean the SCD is older than the device configuration.",
    "type_mismatch": "Both have the point but with different data types, so clients may misread it.",
}


# Control-block service attributes (GoCB, RCB, logs, SV) are not engineering data points.
CONTROL_BLOCK_FCS = {"GO", "GS", "MS", "US", "RP", "BR", "LG", "SV"}


def _device_index(discovery):
    leaves, datasets, reports = {}, {}, set()
    for device in discovery.get("devices", []):
        for node in device.get("nodes", []):
            owner = f"{device['name']}/{node['name']}"
            for leaf in node.get("leaves", []):
                if leaf["fc"] not in CONTROL_BLOCK_FCS:
                    leaves[(leaf["ref"], leaf["fc"])] = leaf.get("mms_type")
            for name, entry in node.get("dataset_members", {}).items():
                datasets[f"{owner}${name}"] = [m.split("[")[0] for m in entry.get("members", [])]
            for kind, names in (("BR", node.get("buffered_reports", [])), ("RP", node.get("unbuffered_reports", []))):
                reports.update(f"{owner}.{kind}.{name}" for name in names)
    return leaves, datasets, reports


def compare(ied, discovery):
    """Classify every reference; returns summary counts and detailed rows."""
    leaves, device_datasets, device_reports = _device_index(discovery)
    rows, seen = [], set()
    for signal in ied["signals"]:
        key = (signal["ref"], signal["fc"])
        seen.add(key)
        if key not in leaves:
            category, device_type = "missing_on_device", None
        elif not type_matches(signal["kind"], leaves[key]):
            category, device_type = "type_mismatch", describe_mms(leaves[key])
        else:
            category, device_type = "match", describe_mms(leaves[key])
        rows.append({"ref": signal["ref"], "fc": signal["fc"], "category": category,
                     "scd_type": signal["kind"], "device_type": device_type, "hint": HINTS.get(category, "")})
    for (ref, fc), mms_type in sorted(leaves.items()):
        if (ref, fc) not in seen:
            rows.append({"ref": ref, "fc": fc, "category": "extra_on_device", "scd_type": None,
                         "device_type": describe_mms(mms_type), "hint": HINTS["extra_on_device"]})
    datasets = []
    for dataset in ied.get("datasets", []):
        ref = dataset.get("ref")
        device = device_datasets.get(ref)
        if device is None:
            datasets.append({"ref": ref, "status": "missing_on_device", "missing": [], "extra": []})
            continue
        expected = set(dataset.get("fcda_refs", []))
        missing = sorted(expected - set(device))
        extra = sorted(set(device) - expected)
        datasets.append({"ref": ref, "status": "match" if not (missing or extra) else "different",
                         "missing": missing, "extra": extra})
    reports = [{"ref": report["ref"], "status": "match" if report["ref"] in device_reports else "missing_on_device"}
               for report in ied.get("reports", [])]
    summary = {name: sum(1 for row in rows if row["category"] == name)
               for name in ("match", "missing_on_device", "extra_on_device", "type_mismatch")}
    summary["datasets_different"] = sum(1 for item in datasets if item["status"] != "match")
    summary["reports_missing"] = sum(1 for item in reports if item["status"] != "match")
    return {"summary": summary, "signals": rows, "datasets": datasets, "reports": reports}
