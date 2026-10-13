"""Synthetic 20-IED/50,000-leaf fixture; contains no customer engineering data."""
from xml.etree import ElementTree as ET
from studio_app.scl.templates import make_template
from studio_app.scl.importer import import_scl

NS = "{http://www.iec.ch/61850/2003/SCL}"


def make_ied(index):
    name = f"Load{index:02d}"
    root = ET.fromstring(make_template(name))
    types = root.find(NS + "DataTypeTemplates")
    analog = next(item for item in types if item.tag == NS + "LNodeType" and item.get("id") == "StudioAnalog")
    for number in range(796):
        ET.SubElement(analog, NS + "DO", {"name": f"LoadValue{number}", "type": "StudioMV"})
    ET.SubElement(analog, NS + "DO", {"name": "LoadFlag", "type": "LoadFlagType"})
    flag = ET.SubElement(types, NS + "DOType", {"id": "LoadFlagType", "cdc": "SPS"})
    ET.SubElement(flag, NS + "DA", {"name": "stVal", "bType": "BOOLEAN", "fc": "ST", "dchg": "true"})
    model = import_scl(ET.tostring(root, encoding="utf-8"), name + ".cid")[0]
    assert len(model["signals"]) == 2500, "Fixture must have exactly 2500 scalar attributes per IED"
    model.update(ip=f"127.51.0.{index + 2}", port=18120)
    return model


def populate(projects, count=20, port=18120):
    ieds = [make_ied(index) for index in range(count)]
    for ied in ieds:
        ied["port"] = port
    sources = []
    for ied in ieds:
        points = [signal for signal in ied["signals"] if signal["kind"] == "FLOAT32"][:5]
        for signal in points:
            sources.append({"ied_id": ied["id"], "ref": signal["ref"], "spec": {
                "mode": "ramp", "offset": 100, "amplitude": 20, "period": 5,
                "seed": 1, "overridden": False}})
    projects.project.update(ieds=ieds, sources=sources)
    return ieds
