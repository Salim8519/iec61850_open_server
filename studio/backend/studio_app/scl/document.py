"""Read an SCL document safely and list IEDs without compiling their models."""
from defusedxml import ElementTree
from studio_app.contracts.errors import StudioError
from .type_resolver import identifier

MAX_XML = 20 * 1024 * 1024


def parse_document(source):
    if isinstance(source, str):
        source = source.encode("utf-8")
    if len(source) > MAX_XML:
        raise StudioError("SCL exceeds the 20 MB import limit")
    try:
        root = ElementTree.fromstring(source)
    except Exception as error:
        raise StudioError(f"Invalid or unsafe XML: {error}") from error
    for element in root.iter():
        element.tag = element.tag.rsplit("}", 1)[-1]
    if root.tag != "SCL":
        raise StudioError("Expected an SCL root element")
    names = [identifier(ied.get("name"), "IED name") for ied in root.findall("IED")]
    if len(names) != len(set(names)):
        raise StudioError("Duplicate IED name in SCL")
    return root


def preview_ieds(source):
    root = parse_document(source)
    result = []
    for ied in root.findall("IED"):
        servers = ied.findall("AccessPoint/Server")
        reason = "" if len(servers) == 1 else (
            "No server model in this IED" if not servers else "Multiple servers are not supported yet")
        result.append({"name": ied.get("name"), "description": ied.get("desc", ""),
                       "manufacturer": ied.get("manufacturer", ""), "type": ied.get("type", ""),
                       "logical_devices": len(ied.findall("AccessPoint/Server/LDevice")),
                       "unavailable_reason": reason})
    if not result:
        raise StudioError("No IEDs found in SCL")
    return result
