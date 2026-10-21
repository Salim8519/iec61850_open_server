"""Synthetic benchmark IEDs: the built-in sample IED widened to a realistic multi-LN shape.

`points` scalar attributes are spread over GGIO logical nodes of 50 measured values each
(mag.f, q, t per value), the shape capacity-audit.md asks for instead of one huge LN.
No customer engineering data.
"""
from xml.etree import ElementTree as ET

NS = "{http://www.iec.ch/61850/2003/SCL}"
PER_LN = 50
CTL_MODEL = None  # --ctl-model: Pos ctlModel ordinal (3 direct-enhanced = template default, 4 sbo-enhanced)


def make_scl(name, points):
    from studio_app.scl.templates import make_template
    ET.register_namespace("", NS[1:-1])
    root = ET.fromstring(make_template(name))
    types = root.find(NS + "DataTypeTemplates")
    device = root.find(f".//{NS}LDevice")
    if CTL_MODEL is not None:
        for value in root.findall(f".//{NS}DOType[@id='StudioDPC']/{NS}DA[@name='ctlModel']/{NS}Val"):
            value.text = str(CTL_MODEL)
    values = max(1, points // 3)
    lns = (values + PER_LN - 1) // PER_LN
    for ln_index in range(lns):
        type_id = f"BenchGGIO{ln_index}"
        node = ET.Element(NS + "LNodeType", {"id": type_id, "lnClass": "GGIO"})
        count = min(PER_LN, values - ln_index * PER_LN)
        for number in range(count):
            ET.SubElement(node, NS + "DO", {"name": f"AnIn{number + 1}", "type": "StudioMV"})
        types.insert(0, node)
        ET.SubElement(device, NS + "LN", {"lnClass": "GGIO", "inst": str(ln_index + 10), "lnType": type_id})
    return ET.tostring(root, encoding="utf-8", xml_declaration=True)
