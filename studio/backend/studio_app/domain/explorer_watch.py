"""Choose the key points IED Explorer watches by itself right after connecting.

A tester first wants positions, measurements, protection trips and device health, so
only those logical-node classes are opened, and a few points are taken from each.
"""
import re

KEY_CLASSES = ("XCBR", "XSWI", "CSWI", "MMXU", "MMXN", "MSQI", "PTRC", "LLN0", "LPHD")
MAX_NODES = 16
MAX_POINTS = 40
PER_NODE = {"MMXU": 8, "MMXN": 4, "MSQI": 4}
WANTED = {
    "XCBR": {"Pos": ("stVal", "q")}, "XSWI": {"Pos": ("stVal", "q")}, "CSWI": {"Pos": ("stVal",)},
    "PTRC": {"Tr": ("general",), "Op": ("general",), "Str": ("general",)},
    "LLN0": {"Beh": ("stVal",), "Health": ("stVal",)}, "LPHD": {"PhyHealth": ("stVal",)},
}


def node_class(name):
    """LN class of a logical node name: prefix + class (4 letters) + instance, or LLN0."""
    if name == "LLN0":
        return "LLN0"
    match = re.search(r"([A-Z]{4})\d*$", name)
    return match.group(1) if match else ""


def key_nodes(device, names):
    chosen = [f"{device}/{name}" for name in names if node_class(name) in KEY_CLASSES]
    return chosen[:MAX_NODES]


def key_points(view):
    """Watch items (ref, fc, type) from one node view of explorer_tree.node_view."""
    kind = node_class(view["name"])
    points = []
    if kind in PER_NODE:
        for item in view["objects"]:
            for attribute in item["attributes"]:
                if attribute["fc"] == "MX" and attribute["path"].endswith("mag.f"):
                    points.append(attribute)
        return [_item(point) for point in points[:PER_NODE[kind]]]
    wanted = WANTED.get(kind, {})
    for item in view["objects"]:
        paths = wanted.get(item["name"], ())
        for attribute in item["attributes"]:
            if attribute["fc"] == "ST" and attribute["path"] in paths:
                points.append(_item(attribute))
    return points


def _item(attribute):
    return {"ref": attribute["ref"], "fc": attribute["fc"], "type": attribute.get("type", "")}


def merge(points):
    seen, result = set(), []
    for point in points:
        key = (point["ref"], point["fc"])
        if key not in seen:
            seen.add(key)
            result.append(point)
    return result[:MAX_POINTS]
