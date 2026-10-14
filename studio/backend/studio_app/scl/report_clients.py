"""Read report control blocks with their declared clients (RptEnabled/ClientLN)."""
from .reports import usable_reports


def _ln_name(element):
    if element.tag == "LN0":
        return "LLN0"
    return element.get("prefix", "") + element.get("lnClass", "") + element.get("inst", "")


def _member(fcda):
    node = fcda.get("prefix", "") + fcda.get("lnClass", "") + fcda.get("lnInst", "")
    path = ".".join(part for part in (fcda.get("doName", ""), fcda.get("daName", "")) if part)
    return f"{fcda.get('ldInst', '')}/{node}.{path} [{fcda.get('fc', '')}]"


def report_blocks(root):
    """Every named ReportControl in the document, with dataset members and clients."""
    blocks = []
    for ied in root.findall("IED"):
        name = ied.get("name", "")
        for device in ied.findall("AccessPoint/Server/LDevice"):
            domain = device.get("ldName") or name + device.get("inst", "")
            for ln in list(device):
                if ln.tag not in {"LN", "LN0"}:
                    continue
                datasets = {item.get("name"): item for item in ln.findall("DataSet")}
                for report in usable_reports(ln):
                    buffered = report.get("buffered") == "true"
                    dataset = report.get("datSet") or ""
                    found = datasets.get(dataset)
                    enabled = report.find("RptEnabled")
                    clients = [] if enabled is None else [
                        {key: client.get(key, "") for key in ("iedName", "ldInst", "prefix", "lnClass", "lnInst")}
                        for client in enabled.findall("ClientLN")]
                    blocks.append({
                        "publisher": name, "name": report.get("name"),
                        "ref": f"{domain}/{_ln_name(ln)}.{'BR' if buffered else 'RP'}.{report.get('name')}",
                        "buffered": buffered, "dataset": dataset, "dataset_found": found is not None,
                        "members": [_member(item) for item in found.findall("FCDA")] if found is not None else [],
                        "clients": clients})
    return blocks
