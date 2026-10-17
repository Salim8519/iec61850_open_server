"""Build the public demo station used in tutorial videos: marketing/videos/demo/scadabench_demo.scd.

Source: scd/model_substation.scd, the generic OpenSCD example (IEDs FEED1, FEED2, BUS1, BUS2,
TR1, TR2; manufacturer "OpenSCD"). No vendor or customer engineering data is used anywhere in
the videos. Changes: only the 50 kV voltage level is kept, and every breaker and disconnector
gets LNode links to its IED (XCBR/XSWI position, CSWI control), so the SLD shows live positions.
"""
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SOURCE = ROOT / "scd" / "model_substation.scd"
TARGET = Path(__file__).resolve().parent / "demo" / "scadabench_demo.scd"
SCL = "http://www.iec.ch/61850/2003/SCL"
Q = "{%s}" % SCL

# Bay -> IED, and equipment -> (switch LN instance, CSWI instance).
BAYS = {"Bay_Feed1": "FEED1", "Bay_Feed2": "FEED2", "Bay_PT1": "TR1", "Bay_PT2": "TR2"}
LINKS = {
    "CBR_F1": ("XCBR", "1", "1"), "DIS_F1_B1": ("XSWI", "2", "2"), "DIS_F1_B2": ("XSWI", "3", "3"),
    "DIS_F1": ("XSWI", "4", "4"),
    "CBR_F2": ("XCBR", "1", "1"), "DIS_F2_B1": ("XSWI", "2", "2"), "DIS_F2_B2": ("XSWI", "3", "3"),
    "DIS_F2": ("XSWI", "4", "4"),
    "CBR_PT1": ("XCBR", "1", "1"), "DIS_PT1_B1": ("XSWI", "2", "2"), "DIS_PT1_B2": ("XSWI", "3", "3"),
    "CBR_PT2": ("XCBR", "1", "1"), "DIS_PT2_B1": ("XSWI", "2", "2"), "DIS_PT2_B2": ("XSWI", "3", "3"),
}


def lnode(ied: str, ln_class: str, inst: str) -> ET.Element:
    return ET.Element(Q + "LNode", {"iedName": ied, "ldInst": "LD1", "prefix": "", "lnClass": ln_class,
                                    "lnInst": inst})


def build() -> Path:
    ET.register_namespace("", SCL)
    ET.register_namespace("esld", "https://transpower.co.nz/SCL/SSD/SLD/v0")
    tree = ET.parse(SOURCE)
    root = tree.getroot()
    header = root.find(Q + "Header")
    header.set("id", "ScadaBench demo station")
    header.set("toolID", "ScadaBench demo (from the OpenSCD example)")
    for substation in root.findall(Q + "Substation"):
        substation.set("desc", "ScadaBench demo: 2 incoming 50 kV feeds, 2 busbars, 2 transformer bays")
        for level in list(substation.findall(Q + "VoltageLevel")):
            if not any(bay.get("name") in BAYS for bay in level.findall(Q + "Bay")):
                substation.remove(level)
        for transformer in substation.findall(Q + "PowerTransformer"):
            for winding in transformer.findall(Q + "TransformerWinding")[1:]:
                transformer.remove(winding)
    for bay in root.iter(Q + "Bay"):
        ied = BAYS.get(bay.get("name"))
        for equipment in bay.findall(Q + "ConductingEquipment"):
            link = LINKS.get(equipment.get("name")) if ied else None
            if link:
                for old in equipment.findall(Q + "LNode"):
                    equipment.remove(old)
                equipment.append(lnode(ied, link[0], link[1]))
                equipment.append(lnode(ied, "CSWI", link[2]))
    TARGET.parent.mkdir(parents=True, exist_ok=True)
    tree.write(TARGET, encoding="UTF-8", xml_declaration=True)
    return TARGET


if __name__ == "__main__":
    print(build())
