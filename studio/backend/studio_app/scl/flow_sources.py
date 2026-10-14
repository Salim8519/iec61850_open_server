"""Collect everything an SCL document declares about signal flow between IEDs."""
from .document import parse_document
from .goose_links import goose_subscriptions, publisher_catalogue
from .report_clients import report_blocks


def read_flow(source):
    """Publishers, subscriptions and report clients across all IEDs in one file."""
    root = parse_document(source)
    catalogue = publisher_catalogue(root)
    subscriptions = {}
    for ied in root.findall("IED"):
        subscriptions[ied.get("name", "")] = goose_subscriptions(root, ied, catalogue)
    return {"ieds": [{"name": ied.get("name", ""), "desc": ied.get("desc", ""),
                      "manufacturer": ied.get("manufacturer", "")} for ied in root.findall("IED")],
            "gocbs": [{key: value for key, value in control.items() if key != "fcdas"}
                      | {"members": [_fcda(item) for item in control["fcdas"]]}
                      for control in catalogue.values()],
            "subscriptions": subscriptions, "reports": report_blocks(root)}


def _fcda(attributes):
    node = attributes.get("prefix", "") + attributes.get("lnClass", "") + attributes.get("lnInst", "")
    path = ".".join(part for part in (attributes.get("doName", ""), attributes.get("daName", "")) if part)
    return f"{attributes.get('ldInst', '')}/{node}.{path} [{attributes.get('fc', '')}]"
