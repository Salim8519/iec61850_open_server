"""Corpus cases about reports, ExtRef inputs, access points/servers and Substation sections."""
from corpus_base import (HEADER, IED, case, extref, gse, gse_control, sub, template_text, variant,
                         with_goose, with_inputs, with_second_ied)

BASE = "make_template(BayController)"
RCB = '<ReportControl name="EventsRCB" rptID="BayController_RP" datSet="Events" confRev="1" buffered="false" indexed="false"'
RCB_END = 'configRef="true" /></ReportControl><ReportControl name="Buffered"'
SERVER_END = "</LDevice></Server></AccessPoint>"


def _rcb(attributes, enabled=None):
    text = variant((RCB, f'<ReportControl name="EventsRCB" {attributes}'))
    if enabled is not None:
        text = sub(text, RCB_END, f'configRef="true" />{enabled}</ReportControl><ReportControl name="Buffered"')
    return text


STD = 'rptID="BayController_RP" datSet="Events" confRev="1" buffered="false"'


def report_cases():
    return [
        case("rcb_unnamed", "ReportControl with an empty name is skipped with a warning", BASE,
             variant(('<ReportControl name="EventsRCB"', '<ReportControl name="" rptID="X" datSet="Events" confRev="1" '
                      'buffered="false" indexed="false"><RptEnabled max="1" /></ReportControl><ReportControl name="EventsRCB"'))),
        case("rcb_name_missing", "ReportControl without a name attribute", BASE,
             _rcb(STD + ' indexed="false"').replace('<ReportControl name="EventsRCB" ', "<ReportControl ", 1)),
        case("rcb_rpt_enabled_0", "RptEnabled max=0", BASE, _rcb(STD + ' indexed="true"', '<RptEnabled max="0" />')),
        case("rcb_rpt_enabled_5", "RptEnabled max=5 indexed: five instances", BASE,
             _rcb(STD + ' indexed="true"', '<RptEnabled max="5" />')),
        case("rcb_rpt_enabled_65", "RptEnabled max=65 (above 64)", BASE, _rcb(STD + ' indexed="true"', '<RptEnabled max="65" />')),
        case("rcb_rpt_enabled_no_max", "RptEnabled without max", BASE, _rcb(STD + ' indexed="true"', "<RptEnabled />")),
        case("rcb_indexed_absent", "indexed absent: default true gives EventsRCB01", BASE, _rcb(STD)),
        case("rcb_indexed_false_max_3", "indexed=false with RptEnabled max=3: one instance", BASE,
             _rcb(STD + ' indexed="false"', '<RptEnabled max="3" />')),
        case("rcb_rptid_space", "rptID with a space", BASE,
             _rcb('rptID="Bay RP" datSet="Events" confRev="1" buffered="false" indexed="false"')),
        case("rcb_rptid_empty", "rptID empty: report name used", BASE,
             _rcb('rptID="" datSet="Events" confRev="1" buffered="false" indexed="false"')),
        case("rcb_dataset_missing", "datSet refers to a dataset that does not exist", BASE,
             _rcb('rptID="R" datSet="Nope" confRev="1" buffered="false" indexed="false"')),
        case("rcb_no_dataset", "ReportControl without datSet", BASE, _rcb('rptID="R" confRev="1" buffered="false" indexed="false"')),
        case("rcb_confrev_text", "confRev not an integer", BASE,
             _rcb('rptID="R" datSet="Events" confRev="one" buffered="false" indexed="false"')),
        case("rcb_buffered_true_text", "buffered='True' (only 'true' counts)", BASE,
             _rcb('rptID="R" datSet="Events" confRev="1" buffered="True" indexed="false"')),
        case("rcb_fcda_ix", "FCDA with ix and a daName", BASE,
             variant(('daName="mag.f" fc="MX" />', 'daName="mag.f" fc="MX" ix="2" />'))),
        case("rcb_fcda_other_ld", "FCDA pointing to another logical device", BASE,
             variant(('<FCDA ldInst="CTRL" lnClass="XSWI" lnInst="2"', '<FCDA ldInst="PROT" lnClass="XSWI" lnInst="2"'))),
    ]


def _lgos():
    return ('<LN lnClass="LGOS" inst="1" lnType="StudioLPHD"><DOI name="GoCBRef"><DAI name="setSrcRef">'
            '<Val>BayControllerCTRL/LLN0$GO$GC1</Val></DAI></DOI></LN>')


def _second_publisher(text):
    """Bay_2 also publishes GC1 with the same dataset, so data-matched inputs are ambiguous."""
    text = with_second_ied(text)
    return sub(text, "</ConnectedAP>", '</ConnectedAP><ConnectedAP iedName="Bay_2" apName="AP1">'
               + gse(mac="01-0C-CD-01-00-02", appid="3002") + "</ConnectedAP>")


def extref_cases():
    goose = with_goose(template_text())
    data = extref(cb=None, do="Pos", ln_class="XSWI")
    return [
        case("ext_named", "named ExtRef to the own GOOSE block", BASE, with_inputs(goose, [extref()])),
        case("ext_packed_intaddr", "ExtRef intAddr containing '|' (vendor packed form)", BASE,
             with_inputs(goose, [extref(cb=None, extra="").replace('intAddr="InPos"',
                                 'intAddr="CON|BayController|1|01:0C:CD:01:00:01|Bool|0,0x0,,Pos.stVal,0"')])),
        case("ext_missing_iedname", "ExtRef without iedName is ignored", BASE, with_inputs(goose, [extref(ied=None)])),
        case("ext_outside_file", "srcCBName pointing to an IED outside the file", BASE,
             with_inputs(goose, [extref(ied="Remote1", cb="GCX")])),
        case("ext_data_zero", "data-matched ExtRef with no candidate block", BASE,
             with_inputs(goose, [extref(cb=None, do="Loc", ln_class="GGIO")])),
        case("ext_data_one", "data-matched ExtRef with exactly one candidate", BASE, with_inputs(goose, [data])),
        case("ext_data_two", "data-matched ExtRef with two candidate blocks", BASE,
             with_inputs(sub(goose, "</ReportControl>" + gse_control() + "</LN0>",
                             "</ReportControl>" + gse_control() + gse_control("GC2") + "</LN0>")
                         .replace("</GSE>", "</GSE>" + gse("GC2", "01-0C-CD-01-00-03", "3003"), 1), [data])),
        case("ext_data_two_ieds", "data-matched ExtRef ambiguous across two publishing IEDs", BASE,
             with_inputs(_second_publisher(goose), [extref(ied="Bay_2", cb=None, do="Pos", ln_class="XSWI")])),
        case("ext_service_report", "ExtRef with serviceType Report is not a GOOSE input", BASE,
             with_inputs(goose, [extref(service="Report")])),
        case("ext_member_path", "ExtRef daName below the FCDA member", BASE,
             with_inputs(goose, [extref(do="AnIn1", da="mag.f", ln_class="GGIO")])),
        case("ext_lgos_setsrcref", "LGOS with setSrcRef naming the subscribed block", BASE,
             with_inputs(sub(goose, '<LN lnClass="LPHD" inst="1" lnType="StudioLPHD" />',
                             '<LN lnClass="LPHD" inst="1" lnType="StudioLPHD" />' + _lgos()), [extref()])),
    ]


def server_cases():
    text = template_text()
    second_ap = '<AccessPoint name="AP2"><Server><Authentication /><LDevice inst="PROT"><LN0 lnClass="LLN0" inst="" lnType="StudioLLN0" /></LDevice></Server></AccessPoint>'
    return [
        case("srv_two_access_points", "IED with two AccessPoints each holding a Server", BASE,
             sub(text, SERVER_END, SERVER_END + second_ap)),
        case("srv_no_server", "AccessPoint without Server (only IED)", BASE,
             text[:text.index("<Server>")] + "</AccessPoint></IED>" + text[text.index("</IED>") + 6:]),
        case("srv_server_at", "second AccessPoint with ServerAt", BASE,
             sub(text, SERVER_END, SERVER_END + '<AccessPoint name="AP2"><ServerAt apName="AP1" /></AccessPoint>')),
        case("srv_no_server_second_ied", "second IED without Server next to a normal one", BASE,
             with_second_ied(text, body='<IED name="Proxy"><AccessPoint name="AP1" /></IED>')),
        case("srv_no_access_point", "IED without any AccessPoint", BASE,
             with_second_ied(text, body='<IED name="Bare" />')),
        case("srv_empty_ldevice", "LDevice without LN0", BASE,
             sub(text, SERVER_END, '</LDevice><LDevice inst="EMPTY" /></Server></AccessPoint>')),
    ]


def _substation(body):
    return sub(template_text(), HEADER, HEADER + body)


def _bay(name, extra=""):
    return (f'<Bay name="{name}"><ConductingEquipment name="QA1" type="CBR">'
            f'<LNode iedName="{IED}" ldInst="CTRL" lnClass="XCBR" lnInst="1" />'
            f'<Terminal connectivityNode="S1/V1/{name}/L 1" substationName="S1" voltageLevelName="V1" '
            f'bayName="{name}" cNodeName="L 1" /></ConductingEquipment>'
            f'<ConnectivityNode name="L 1" pathName="S1/V1/{name}/L 1" />{extra}</Bay>')


def substation_cases():
    ns = '<SCL xmlns="http://www.iec.ch/61850/2003/SCL"'
    station = '<Substation name="S1"><VoltageLevel name="V1">' + _bay("B1") + "</VoltageLevel></Substation>"
    esld = station.replace('<ConductingEquipment name="QA1" type="CBR">',
                           '<ConductingEquipment name="QA1" type="CBR" esld:x="-1.25" esld:y="3.5e1">')
    return [
        case("sub_basic", "one Substation with a bay, ConnectivityNode name with a space", BASE, _substation(station)),
        case("sub_several", "two Substation sections", BASE, _substation(station + station.replace('"S1"', '"S2"', 1))),
        case("sub_esld", "esld:x/y floats and negatives on equipment", BASE,
             sub(_substation(esld), ns, ns + ' xmlns:esld="https://transpower.co.nz/SCL/SSD/SLD/v0"')),
        case("sub_lnode_absent_ied", "LNode pointing to an IED that is not in the file", BASE,
             _substation(station.replace(f'iedName="{IED}"', 'iedName="Ghost"'))),
        case("sub_grounded_transformer", "grounded terminal and a power transformer", BASE,
             _substation(station.replace("</ConductingEquipment>", '<Terminal cNodeName="grounded" substationName="S1" '
                                         'voltageLevelName="V1" bayName="B1" connectivityNode="S1/V1/B1/grounded" />'
                                         "</ConductingEquipment>", 1)
                         .replace("<VoltageLevel", '<PowerTransformer name="T1" type="PTR"><TransformerWinding '
                                  'name="W1" type="PTW" /></PowerTransformer><VoltageLevel', 1))),
    ]
