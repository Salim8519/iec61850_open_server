"""Read the SCL Substation section into a flat, drawing-neutral topology."""
from .document import parse_document

ESLD = "{https://transpower.co.nz/SCL/SSD/SLD/v0}"
GROUNDED = "grounded"


def has_substation(source):
    return parse_document(source).find("Substation") is not None


def read_substation(source):
    """Return equipment, connectivity nodes, LNodes and logical-device names.

    Nothing is interpreted for drawing here; unsupported elements are listed so the
    caller can report them instead of dropping them.
    """
    root = parse_document(source)
    result = {"substations": [], "equipment": [], "nodes": {}, "other": [],
              "ld_domains": _ld_domains(root), "diagnostics": []}
    for substation in root.findall("Substation"):
        name = substation.get("name", "")
        result["substations"].append(name)
        for item in substation.findall("PowerTransformer"):
            _other(result, item, [name])
        for level_index, level in enumerate(substation.findall("VoltageLevel")):
            _voltage_level(result, level, name, level_index)
    for equipment in result["equipment"]:
        equipment["terminals"] = [_resolve(result, equipment, terminal)
                                  for terminal in equipment.pop("raw_terminals")]
    return result


def _voltage_level(result, level, substation, level_index):
    names = [substation, level.get("name", "")]
    for item in level.findall("PowerTransformer"):
        _other(result, item, names)
    for bay_index, bay in enumerate(level.findall("Bay")):
        bay_names = names + [bay.get("name", "")]
        for node in bay.findall("ConnectivityNode"):
            path = node.get("pathName") or "/".join(bay_names + [node.get("name", "")])
            result["nodes"][path] = {"path": path, "name": node.get("name", ""),
                                     "level": level_index, "bay": bay_index,
                                     "bay_name": bay_names[-1], "bus_y": _bus_y(node)}
        for item in bay.findall("ConductingEquipment"):
            result["equipment"].append({
                "path": "/".join(bay_names + [item.get("name", "")]),
                "name": item.get("name", ""), "type": item.get("type", ""),
                "level": level_index, "bay": bay_index, "bay_name": bay_names[-1],
                "esld": _position(item),
                "raw_terminals": [_terminal_path(terminal) for terminal in item.findall("Terminal")],
                "lnodes": [_lnode(node) for node in item.iter("LNode")
                           if _owner_is(item, node)]})
        for item in bay.findall("PowerTransformer") + bay.findall("GeneralEquipment"):
            _other(result, item, bay_names)


def _owner_is(equipment, lnode):
    """LNodes directly on the equipment, not on phase SubEquipment."""
    return lnode in equipment.findall("LNode")


def _other(result, item, names):
    result["other"].append({"path": "/".join(names + [item.get("name", "")]),
                            "element": item.tag, "type": item.get("type", "")})


def _terminal_path(terminal):
    if terminal.get("cNodeName") == GROUNDED or terminal.get("connectivityNode", "").endswith("/" + GROUNDED):
        return GROUNDED
    path = terminal.get("connectivityNode")
    if path:
        return path
    parts = [terminal.get(key) for key in ("substationName", "voltageLevelName", "bayName", "cNodeName")]
    return "/".join(parts) if all(parts) else None


def _resolve(result, equipment, path):
    if path == GROUNDED or path in result["nodes"]:
        return path
    result["diagnostics"].append(
        f"{equipment['path']}: terminal points to connectivity node "
        f"{path or '(missing)'} which is not declared; no wire is drawn for it")
    return None


def _lnode(node):
    return {key: node.get(key, "") for key in ("iedName", "ldInst", "prefix", "lnClass", "lnInst")}


def _number(value):
    try:
        return float(value)
    except (TypeError, ValueError):
        return None


def _position(item):
    x, y = _number(item.get(ESLD + "x")), _number(item.get(ESLD + "y"))
    return None if x is None or y is None else {"x": x, "y": y}


def _bus_y(node):
    """Height of a drawn busbar from esld vertices marked bus="true", when present."""
    heights = []
    for section in node.iter("Section"):
        if section.get("bus") != "true":
            continue
        heights.extend(_number(vertex.get(ESLD + "y")) for vertex in section.iter("Vertex"))
    heights = [value for value in heights if value is not None]
    return min(heights) if heights else None


def _ld_domains(root):
    domains = {}
    for ied in root.findall("IED"):
        name = ied.get("name", "")
        for device in ied.findall("AccessPoint/Server/LDevice"):
            inst = device.get("inst", "")
            domains[(name, inst)] = device.get("ldName") or name + inst
    return domains
