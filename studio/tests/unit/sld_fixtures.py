"""Small SCL Substation documents for SLD import tests."""


def terminal(node):
    if node == "grounded":
        return '<Terminal cNodeName="grounded" substationName="S" voltageLevelName="V" bayName="B"/>'
    return f'<Terminal connectivityNode="{node}"/>'


def equipment(name, kind, nodes, lnodes="", esld=None):
    position = f' esld:x="{esld[0]}" esld:y="{esld[1]}"' if esld else ""
    terminals = "".join(terminal(node) for node in nodes)
    return f'<ConductingEquipment name="{name}" type="{kind}"{position}>{lnodes}{terminals}</ConductingEquipment>'


def scl(bays, extra="", ieds=""):
    """bays: list of (bay name, equipment xml, connectivity node names)."""
    body = ""
    for bay, items, nodes in bays:
        cnodes = "".join(f'<ConnectivityNode name="{n}" pathName="S/V/{bay}/{n}"/>' for n in nodes)
        body += f'<Bay name="{bay}">{items}{cnodes}</Bay>'
    return (f'<SCL xmlns="http://www.iec.ch/61850/2003/SCL" '
            f'xmlns:esld="https://transpower.co.nz/SCL/SSD/SLD/v0">'
            f'<Substation name="S">{extra}<VoltageLevel name="V">{body}</VoltageLevel></Substation>'
            f'{ieds}</SCL>')


def feeder(esld=False):
    """Busbar BB/L, a feeder with DS, CB, CT, a VT tee and an earth switch."""
    def at(x, y):
        return (x, y) if esld else None
    items = (equipment("QB1", "DIS", ["S/V/BB/L", "S/V/F1/L1"], esld=at(2, 2))
             + equipment("QA1", "CBR", ["S/V/F1/L1", "S/V/F1/L2"], esld=at(2, 4))
             + equipment("BI1", "CTR", ["S/V/F1/L2", "S/V/F1/L3"], esld=at(2, 6))
             + equipment("BU1", "VTR", ["S/V/F1/L3"], esld=at(4, 8))
             + equipment("QC1", "DIS", ["S/V/F1/L3", "grounded"], esld=at(0, 8))
             + equipment("IFL1", "IFL", ["S/V/F1/L3"]))
    return scl([("BB", "", ["L"]), ("F1", items, ["L1", "L2", "L3"])],
               extra='<PowerTransformer name="T1" type="PTR"/>')
