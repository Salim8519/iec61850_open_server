"""List every GOOSE block an SCL document declares, sendable or not, with the reason.

`publisher_catalogue` keeps only blocks the engine can publish. The link map also
needs the others, so a subscriber expecting a block that the file switches off is
shown as "not sent (disabled in file)" instead of an unexplained missing link.
"""
from .communication import read_communication
from .goose_controls import gocb_reference
from .goose_links import publisher_catalogue
from .goose_vendor import ge_function, ge_label, vendor_attributes


def _status(element, datasets, address, sendable):
    if sendable:
        return "sendable"
    if element.get("type", "GOOSE") != "GOOSE":
        return "not_goose"
    if ge_function(element) == "disabled":
        return "disabled_in_file"
    if element.get("datSet", "") not in datasets:
        return "no_dataset"
    if address is None:
        return "no_address"
    return "not_sendable"


def _conf_rev(element):
    try:
        return int(element.get("confRev", "0"))
    except ValueError:
        return None


def block_inventory(root, catalogue=None):
    """{(ied, ldInst, cbName): block} for every GSEControl, keyed like `publisher_catalogue`."""
    catalogue = publisher_catalogue(root) if catalogue is None else catalogue
    blocks = {}
    for ied in root.findall("IED"):
        name = ied.get("name", "")
        communication, _ = read_communication(root, name)
        addresses = {(item["ld_inst"], item["cb_name"]): item for item in communication["gse"]}
        for ld in ied.findall("AccessPoint/Server/LDevice"):
            ln0 = ld.find("LN0")
            if ln0 is None:
                continue
            inst = ld.get("inst", "")
            domain = ld.get("ldName") or name + inst
            datasets = {item.get("name", ""): item for item in ln0.findall("DataSet")}
            for element in ln0.findall("GSEControl"):
                key = (name, inst, element.get("name", ""))
                address = addresses.get((inst, key[2]))
                known = catalogue.get(key)
                dataset = element.get("datSet", "")
                fcdas = [dict(item.attrib) for item in datasets[dataset].findall("FCDA")] if dataset in datasets else []
                blocks[key] = {
                    "publisher": name, "ld_inst": inst, "name": key[2],
                    "gocb_ref": gocb_reference(domain, key[2]),
                    "go_id": element.get("appID") or gocb_reference(domain, key[2]),
                    "dataset": dataset, "dataset_ref": f"{domain}/LLN0${dataset}" if dataset else "",
                    "conf_rev": _conf_rev(element),
                    "app_id": address["app_id"] if address else None, "mac": address["mac"] if address else "",
                    "vlan_id": address["vlan_id"] if address else None,
                    "status": _status(element, datasets, address, known is not None),
                    "label": ge_label(element), "vendor": vendor_attributes(element), "fcdas": fcdas}
    return blocks
