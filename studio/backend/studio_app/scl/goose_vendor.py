"""Read vendor GOOSE hints from SCL without inventing data.

Two vendor conventions are understood, both only as hints (change 0094):
- GE Multilin writes `{http://www.gedigitalenergy.com/multilin}Function="Enabled|Disabled"`
  (plus Label and TAL) on GSEControl. A Disabled block is never sent by the device.
- GE C650 packs the publisher into ExtRef intAddr, for example
  `CON|BCU_120|1|01:0C:CD:01:00:00|Bool|...`, and leaves srcCBName out.
"""
import re

GE_MULTILIN = "http://www.gedigitalenergy.com/multilin"
_MAC = re.compile(r"^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$")
_NAME = re.compile(r"^[A-Za-z][A-Za-z0-9_]{0,63}$")


def vendor_attributes(element):
    """Every namespaced attribute of an element as {namespace: {name: value}}, unchanged."""
    result = {}
    for key, value in element.attrib.items():
        if key.startswith("{") and "}" in key:
            namespace, name = key[1:].split("}", 1)
            result.setdefault(namespace, {})[name] = value
    return result


def ge_function(element):
    """'enabled', 'disabled' or None when the GE Function attribute is absent or unreadable."""
    value = (element.get(f"{{{GE_MULTILIN}}}Function") or "").strip().lower()
    return value if value in {"enabled", "disabled"} else None


def ge_label(element):
    return element.get(f"{{{GE_MULTILIN}}}Label") or ""


def packed_input_hint(int_addr):
    """Publisher name, MAC and value type from a packed vendor intAddr, or None.

    Only the strict shape `LD|IED|index|MAC|type[|...]` is accepted; anything else
    is an ordinary intAddr and nothing is read from it.
    """
    parts = (int_addr or "").split("|")
    if len(parts) < 5 or not _NAME.match(parts[1]) or not _MAC.match(parts[3]):
        return None
    return {"ld_inst": parts[0], "publisher": parts[1], "index": parts[2],
            "mac": parts[3].replace(":", "-").upper(), "value_type": parts[4], "vendor": "packed intAddr"}
