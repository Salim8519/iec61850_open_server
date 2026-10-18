"""Merge the GOOSE declarations of every SCL file stored with the project."""
from .document import parse_document
from .goose_inventory import block_inventory
from .goose_links import goose_subscriptions
from .goose_remote import remote_inputs


def _owners(parsed):
    """IED name -> (document index, element). Copies of one IED in several files are read once:
    from the file it was loaded from, otherwise from the first file that contains it."""
    owners = {}
    for prefer_loaded in (True, False):
        for index, (root, loaded) in enumerate(parsed):
            for ied in root.findall("IED"):
                name = ied.get("name", "")
                if name in loaded or not prefer_loaded:
                    owners.setdefault(name, (index, ied))
    return owners


def read_goose_sources(documents):
    """`documents` is [(SCL text, IED names loaded from it)]; returns merged declarations.

    {ieds: {name: {desc, manufacturer, type, loaded}}, blocks: {key: block},
     subscriptions: {name: [...]}, problems: {name: [...]}, remote: {name: [...]}}
    """
    parsed = [(parse_document(text), set(loaded)) for text, loaded in documents]
    owners = _owners(parsed)
    inventories = [block_inventory(root) for root, _ in parsed]
    blocks = {}
    for index, inventory in enumerate(inventories):
        for key, block in inventory.items():
            if owners[key[0]][0] == index:
                blocks[key] = block
    loaded = set().union(*(names for _, names in parsed)) if parsed else set()
    known = set(owners)
    result = {"ieds": {}, "blocks": blocks, "subscriptions": {}, "problems": {}, "remote": {}}
    for name, (index, ied) in sorted(owners.items()):
        result["ieds"][name] = {"desc": ied.get("desc", ""), "manufacturer": ied.get("manufacturer", ""),
                                "type": ied.get("type", ""), "loaded": name in loaded}
        subscriptions, problems = goose_subscriptions(parsed[index][0], ied, blocks)
        remote = remote_inputs(ied, known)
        # Inputs from a publisher outside every file are listed as remote, not as unresolved problems.
        outside = {item["publisher"] for item in remote}
        result["subscriptions"][name] = subscriptions
        result["problems"][name] = [message for message in problems if not any(
            f"GOOSE block {other}/" in message or f"Input from {other}/" in message for other in outside)]
        result["remote"][name] = remote
    return result
