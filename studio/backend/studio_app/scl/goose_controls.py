"""Translate GSEControl blocks and their SCD addresses into native GoCB configuration."""
from .goose_vendor import ge_function


def gocb_reference(domain, name):
    return f"{domain}/LLN0$GO${name}"


def _usable_go_id(value):
    return bool(value) and len(value) <= 129 and not any(c.isspace() or c in "();{}" for c in value)


def goose_controls(ln0, domain, ld_inst, gse, datasets, ied_name, warnings):
    """Return publishable GoCBs for one LN0; unusable blocks become warnings."""
    controls = []
    addresses = {(item["ld_inst"], item["cb_name"]): item for item in gse}
    names = {item["name"]: item for item in datasets}
    for element in ln0.findall("GSEControl"):
        name = element.get("name", "")
        where = f"GOOSE block {ied_name}/{ld_inst}/{name}"
        if element.get("type", "GOOSE") != "GOOSE":
            warnings.append(f"{where} is {element.get('type')}; only GOOSE is published")
            continue
        if ge_function(element) == "disabled":
            warnings.append(f"{where} is switched off in the file (vendor Function=Disabled), "
                            "so it is not sent, like on the real device")
            continue
        dataset = element.get("datSet", "")
        if dataset not in names:
            warnings.append(f"{where} has no dataset in this logical device, so it is not published")
            continue
        address = addresses.get((ld_inst, name))
        if address is None:
            warnings.append(f"{where} has no network address (GSE) in the SCD Communication "
                            "section, so it is not published")
            continue
        reference = gocb_reference(domain, name)
        go_id = element.get("appID") or reference
        if not _usable_go_id(go_id):
            warnings.append(f"{where} has a goID the native engine cannot represent; not published")
            continue
        try:
            conf_rev = int(element.get("confRev", "0"))
        except ValueError:
            warnings.append(f"{where} has an unreadable confRev; not published")
            continue
        controls.append({**{key: address[key] for key in
                            ("mac", "app_id", "vlan_id", "vlan_priority", "min_time", "max_time")},
                         "name": name, "ld_inst": ld_inst, "gocb_ref": reference, "go_id": go_id,
                         "dataset": dataset, "dataset_ref": f"{domain}/LLN0${dataset}",
                         "conf_rev": conf_rev, "fixed_offs": element.get("fixedOffs") == "true",
                         "members": list(names[dataset]["members"])})
    return controls


def goose_cfg_lines(control):
    minimum = control["min_time"] if control["min_time"] is not None else -1
    maximum = control["max_time"] if control["max_time"] is not None else -1
    mac = control["mac"].replace("-", "").lower()
    return [f"GC({control['name']} {control['go_id']} {control['dataset']} {control['conf_rev']} "
            f"{int(control['fixed_offs'])} {minimum} {maximum}){{",
            f"PA({control['vlan_priority']} {control['vlan_id']} {control['app_id']} {mac});",
            "}"]
