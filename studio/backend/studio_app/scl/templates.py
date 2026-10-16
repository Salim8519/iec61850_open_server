"""Create a documented laboratory IED template as standard SCL XML."""
from xml.etree.ElementTree import Element, SubElement, tostring
from .type_resolver import identifier

NS = "http://www.iec.ch/61850/2003/SCL"


def child(parent, tag, **attributes):
    return SubElement(parent, tag, {key: str(value) for key, value in attributes.items()})


def da(parent, name, kind, fc=None, value=None, **attributes):
    item = child(parent, "DA" if fc else "BDA", name=name, bType=kind,
                 **({"fc": fc} if fc else {}), **attributes)
    if value is not None:
        child(item, "Val").text = str(value)
    return item


def make_template(name="BayController"):
    identifier(name, "IED name")
    root = Element("SCL", {"xmlns": NS, "version": "2007", "revision": "B", "release": "4"})
    child(root, "Header", id=name, version="1", revision="0")
    ied = child(root, "IED", name=name, manufacturer="ScadaBench", type="Lab bay controller",
                configVersion="1", desc="Simulation template; not a manufacturer relay model")
    server = child(child(ied, "AccessPoint", name="AP1"), "Server")
    child(server, "Authentication", none="true")
    ld = child(server, "LDevice", inst="CTRL")
    ln0 = child(ld, "LN0", lnClass="LLN0", inst="", lnType="StudioLLN0")
    ds = child(ln0, "DataSet", name="Events")
    for cls, inst, obj, attr, fc in [("XCBR", "1", "Pos", "stVal", "ST"),
                                     ("XSWI", "1", "Pos", "stVal", "ST"),
                                     ("XSWI", "2", "Pos", "stVal", "ST"),
                                     ("GGIO", "1", "AnIn1", "mag.f", "MX"),
                                     ("GGIO", "1", "SPCSO1", "stVal", "ST")]:
        child(ds, "FCDA", ldInst="CTRL", lnClass=cls, lnInst=inst, doName=obj, daName=attr, fc=fc)
    for buffered in (False, True):
        rc = child(ln0, "ReportControl", name="Buffered" if buffered else "EventsRCB",
                   rptID=name + ("_BR" if buffered else "_RP"), datSet="Events", confRev="1",
                   buffered=str(buffered).lower(), indexed="false", bufTime="0", intgPd="1000")
        child(rc, "TrgOps", dchg="true", qchg="true", dupd="false", period="true", gi="true")
        child(rc, "OptFields", seqNum="true", timeStamp="true", reasonCode="true",
              dataSet="true", dataRef="false", bufOvfl="true", entryID="true", configRef="true")
    child(ld, "LN", lnClass="LPHD", inst="1", lnType="StudioLPHD")
    for cls, inst, ln_type in [("XCBR", "1", "StudioCB"), ("XSWI", "1", "StudioSwitch"),
                                ("XSWI", "2", "StudioSwitch"), ("GGIO", "1", "StudioAnalog")]:
        ln = child(ld, "LN", lnClass=cls, inst=inst, lnType=ln_type)
        if cls in {"XCBR", "XSWI"}:
            pos = child(ln, "DOI", name="Pos")
            child(child(pos, "DAI", name="stVal"), "Val").text = "1"
    types = child(root, "DataTypeTemplates")
    for key, cls, objects in [
        ("StudioLLN0", "LLN0", [("Beh", "StudioStatus"), ("Health", "StudioStatus"), ("NamPlt", "StudioName")]),
        ("StudioLPHD", "LPHD", [("PhyHealth", "StudioStatus")]),
        ("StudioCB", "XCBR", [("Pos", "StudioDPC"), ("Loc", "StudioBoolean")]),
        ("StudioSwitch", "XSWI", [("Pos", "StudioDPC"), ("Loc", "StudioBoolean")]),
        ("StudioAnalog", "GGIO", [("AnIn1", "StudioMV"), ("AnIn2", "StudioMV"),
                                    ("SPCSO1", "StudioSPC")]),
    ]:
        ln_type = child(types, "LNodeType", id=key, lnClass=cls)
        for obj, type_name in objects:
            child(ln_type, "DO", name=obj, type=type_name)
    status = child(types, "DOType", id="StudioStatus", cdc="INS")
    da(status, "stVal", "INT32", "ST", 1, dchg="true")
    da(status, "q", "Quality", "ST", qchg="true")
    da(status, "t", "Timestamp", "ST")
    boolean = child(types, "DOType", id="StudioBoolean", cdc="SPS")
    da(boolean, "stVal", "BOOLEAN", "ST", "false", dchg="true")
    da(boolean, "q", "Quality", "ST", qchg="true")
    da(boolean, "t", "Timestamp", "ST")
    plate = child(types, "DOType", id="StudioName", cdc="LPL")
    for field in ("vendor", "swRev", "d"):
        da(plate, field, "VisString255", "DC", "ScadaBench")
    pos = child(types, "DOType", id="StudioDPC", cdc="DPC")
    da(pos, "stVal", "Dbpos", "ST", dchg="true")
    da(pos, "q", "Quality", "ST", qchg="true")
    da(pos, "t", "Timestamp", "ST")
    da(pos, "ctlModel", "Enum", "CF", 3, type="StudioControlModel")
    da(pos, "sboTimeout", "INT32U", "CF", 10000)
    da(pos, "sboClass", "Enum", "CF", 0, type="StudioSboClass")
    da(pos, "SBO", "VisString129", "CO")
    for field in ("Oper", "SBOw"):
        da(pos, field, "Struct", "CO", type="StudioOperate")
    da(pos, "Cancel", "Struct", "CO", type="StudioCancel")
    single = child(types, "DOType", id="StudioSPC", cdc="SPC")
    da(single, "stVal", "BOOLEAN", "ST", "false", dchg="true")
    da(single, "q", "Quality", "ST", qchg="true")
    da(single, "t", "Timestamp", "ST")
    da(single, "ctlModel", "Enum", "CF", 3, type="StudioControlModel")
    da(single, "sboTimeout", "INT32U", "CF", 10000)
    da(single, "sboClass", "Enum", "CF", 0, type="StudioSboClass")
    da(single, "SBO", "VisString129", "CO")
    for field in ("Oper", "SBOw"):
        da(single, field, "Struct", "CO", type="StudioOperate")
    da(single, "Cancel", "Struct", "CO", type="StudioCancel")
    analog = child(types, "DOType", id="StudioMV", cdc="MV")
    da(analog, "mag", "Struct", "MX", type="StudioMagnitude", dchg="true")
    da(analog, "q", "Quality", "MX", qchg="true")
    da(analog, "t", "Timestamp", "MX")
    magnitude = child(types, "DAType", id="StudioMagnitude")
    da(magnitude, "f", "FLOAT32", value=0)
    operate = child(types, "DAType", id="StudioOperate")
    da(operate, "ctlVal", "BOOLEAN")
    da(operate, "origin", "Struct", type="StudioOrigin")
    da(operate, "ctlNum", "INT8U")
    da(operate, "T", "Timestamp")
    da(operate, "Test", "BOOLEAN")
    da(operate, "Check", "Check")
    cancel = child(types, "DAType", id="StudioCancel")
    for field, kind in (("ctlVal", "BOOLEAN"), ("origin", "Struct"),
                        ("ctlNum", "INT8U"), ("T", "Timestamp"), ("Test", "BOOLEAN")):
        da(cancel, field, kind, **({"type": "StudioOrigin"} if field == "origin" else {}))
    origin = child(types, "DAType", id="StudioOrigin")
    da(origin, "orCat", "Enum", type="StudioOriginCategory")
    da(origin, "orIdent", "Octet64")
    for type_id, values in [("StudioControlModel", ["status-only", "direct-normal", "sbo-normal", "direct-enhanced", "sbo-enhanced"]),
                            ("StudioSboClass", ["operate-once", "operate-many"]),
                            ("StudioOriginCategory", ["not-supported", "bay-control", "station-control", "remote-control"] )]:
        enum = child(types, "EnumType", id=type_id)
        for index, label in enumerate(values):
            child(enum, "EnumVal", ord=index).text = label
    return tostring(root, encoding="utf-8", xml_declaration=True)
