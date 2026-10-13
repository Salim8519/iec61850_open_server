"""Verified scalar instance edits in a preserved copy of an original SCL document."""
from datetime import datetime, timezone
from lxml import etree
from studio_app.contracts.errors import StudioError
from studio_app.domain.signal import validate_value
from .importer import import_scl
from .schema_validation import validate_schema

NS = "{http://www.iec.ch/61850/2003/SCL}"


def export_ied(source, name, overrides):
    """Keep all source IEDs; edit only the named server's explicit scalar values."""
    if not isinstance(overrides, dict) or len(overrides) > 50000:
        raise StudioError("SCL value overrides must be an object of at most 50,000 references")
    original = import_scl(source, selected_names=[name])[0]
    _require_valid(original["validation"])
    raw = source.encode("utf-8") if isinstance(source, str) else source
    parser = etree.XMLParser(resolve_entities=False, no_network=True, load_dtd=False)
    tree = etree.fromstring(raw, parser).getroottree()
    root = tree.getroot()
    ied = next(item for item in root.findall(NS + "IED") if item.get("name") == name)
    nodes = _logical_nodes(ied, name)
    signals = {item["ref"]: item for item in original["signals"]}
    expected = {ref: item["value"] for ref, item in signals.items()}
    for reference, value in overrides.items():
        signal = signals.get(reference)
        if signal is None:
            raise StudioError(f"Unknown export reference: {reference}")
        try:
            value = validate_value(signal, value)
        except ValueError as error:
            raise StudioError(f"{reference}: {error}") from error
        text = _value_text(signal, value)
        domain, path = reference.split("/", 1)
        parts = path.split(".")
        if any("(" in part for part in parts):
            raise StudioError(f"Per-index SCL export is not supported yet: {reference}")
        parent = nodes[(domain, parts[0])]
        for index, part in enumerate(parts[1:]):
            tag = "DOI" if index == 0 else ("DAI" if index == len(parts) - 2 else "SDI")
            parent = _instance(parent, tag, part)
        values = parent.findall(NS + "Val")
        if len(values) > 1 or any(item.get("sGroup") is not None for item in values):
            raise StudioError(f"Setting-group values need an explicit group selection: {reference}")
        target = values[0] if values else etree.SubElement(parent, NS + "Val")
        target.text = text
        expected[reference] = value
    payload = etree.tostring(tree, encoding="UTF-8", xml_declaration=True)
    _require_valid(validate_schema(payload))
    restored = import_scl(payload, selected_names=[name])[0]
    actual = {item["ref"]: (item["kind"], item["fc"], item["value"])
              for item in restored["signals"]}
    wanted = {ref: (item["kind"], item["fc"], expected[ref]) for ref, item in signals.items()}
    if actual != wanted:
        raise StudioError("Export failed signal type/reference/value round-trip verification")
    if any(original[key] != restored[key] for key in ("datasets", "reports")):
        raise StudioError("Export changed dataset or report definitions unexpectedly")
    return payload


def _require_valid(validation):
    if validation["status"] != "valid":
        raise StudioError("Verified SCL export requires a supported, schema-valid document: "
                          + validation["errors"][0]["message"])


def _logical_nodes(ied, name):
    nodes = {}
    for device in ied.findall(f"{NS}AccessPoint/{NS}Server/{NS}LDevice"):
        domain = device.get("ldName") or name + device.get("inst")
        for node in device:
            if node.tag not in {NS + "LN", NS + "LN0"}:
                continue
            logical_name = "LLN0" if node.tag == NS + "LN0" else (
                node.get("prefix", "") + node.get("lnClass", "") + node.get("inst", ""))
            nodes[(domain, logical_name)] = node
    return nodes


def _instance(parent, tag, name):
    matches = [child for child in parent.findall(NS + tag) if child.get("name") == name]
    if len(matches) > 1 or any(child.get("ix") is not None for child in matches):
        raise StudioError(f"Ambiguous or indexed SCL instance: {name}")
    if matches:
        return matches[0]
    child = etree.Element(NS + tag, name=name)
    if tag == "DOI":
        # DOI precedes the later LN/LLN0 content, including GOOSE control blocks.
        later = {NS + key for key in ("Inputs", "Log", "GSEControl", "SampledValueControl",
                                     "SettingControl", "SCLControl")}
        position = next((index for index, item in enumerate(parent) if item.tag in later), len(parent))
        parent.insert(position, child)
    else:
        parent.append(child)
    return child


def _value_text(signal, value):
    kind = signal["kind"]
    if kind == "BOOLEAN":
        return "true" if value else "false"
    if kind == "Enum":
        label = next((label for label, ordinal in signal["enums"].items() if ordinal == value), None)
        if label is None:
            raise StudioError(f"Export requires a declared enumeration value: {signal['ref']}")
        return label
    if kind == "Timestamp":
        return datetime.fromtimestamp(value / 1000, timezone.utc).isoformat(timespec="milliseconds")
    if kind.startswith(("INT", "FLOAT")) or "String" in kind or kind in {"Currency", "ObjRef"}:
        return str(value)
    raise StudioError(f"SCL value encoding for {kind} is not supported yet: {signal['ref']}")
