"""Shape one logical node read by IED Explorer into data objects, datasets and report blocks."""
from .explorer_values import type_name

# Control-block service attributes are shown as blocks, not as data objects.
BLOCK_FCS = {"RP", "BR", "GO", "GS", "MS", "US", "LG", "SV"}
FC_MEANING = {"ST": "status", "MX": "measurement", "CO": "control", "SP": "setting", "SG": "setting group",
              "SE": "editable setting", "CF": "configuration", "DC": "description", "EX": "extended definition",
              "SV": "substitution", "OR": "operate received", "BL": "blocking"}
# What a tester looks at first: live status and measurements, then settings and description, controls last.
FC_ORDER = {fc: rank for rank, fc in enumerate(("ST", "MX", "SP", "SG", "SE", "SV", "CF", "DC", "EX", "BL", "OR", "CO"))}


def _rank(fc):
    return FC_ORDER.get(fc, len(FC_ORDER))


def node_view(reference, answer):
    """reference is "LD/LN"; answer is the native explore result for that node."""
    objects = {}
    for leaf in answer.get("leaves", []):
        if leaf["fc"] in BLOCK_FCS:
            continue
        rest = leaf["ref"][len(reference) + 1:]
        name = rest.split(".", 1)[0]
        entry = objects.setdefault(name, {"name": name, "fcs": [], "attributes": []})
        if leaf["fc"] not in entry["fcs"]:
            entry["fcs"].append(leaf["fc"])
        entry["attributes"].append({"ref": leaf["ref"], "fc": leaf["fc"], "path": rest[len(name) + 1:] or name,
                                    "type": type_name(leaf["mms_type"])})
    for entry in objects.values():
        entry["fcs"].sort(key=_rank)
        entry["attributes"].sort(key=lambda item: _rank(item["fc"]))
    datasets = [{"name": name, "ref": f"{reference}${name}",
                 "members": (answer.get("dataset_members", {}).get(name) or {}).get("members", [])}
                for name in answer.get("datasets", [])]
    reports = [{"name": name, "ref": f"{reference}.{kind}.{name}", "buffered": kind == "BR"}
               for kind, key in (("BR", "buffered_reports"), ("RP", "unbuffered_reports"))
               for name in answer.get(key, [])]
    return {"ref": reference, "name": answer.get("name", reference.split("/")[-1]),
            "objects": sorted(objects.values(), key=lambda item: item["name"]),
            "datasets": datasets, "reports": reports,
            "leaf_count": sum(len(item["attributes"]) for item in objects.values())}
