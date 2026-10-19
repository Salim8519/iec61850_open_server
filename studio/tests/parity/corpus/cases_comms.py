"""Communication, control blocks, inputs and the Substation section."""
from parity.corpus.base import Case, edit, template, utf8, with_communication

GSE_CONTROL = '<GSEControl name="Gse1" datSet="Events" appID="Corpus_G1" confRev="1" type="GOOSE" />'
SMV_CONTROL = ('<SampledValueControl name="Smv1" datSet="Events" smvID="Corpus_SV1" smpRate="80" nofASDU="1" '
               'confRev="1" multicast="true"><SmvOpts refreshTime="true" sampleRate="true" /></SampledValueControl>')
SUBSTATION = ('<Substation name="S1" desc="corpus"><VoltageLevel name="E1"><Bay name="Q 01">'
              '<ConductingEquipment name="QA1" type="CBR"><Terminal name="T1" connectivityNode="S1/E1/Q 01/L 1" '
              'substationName="S1" voltageLevelName="E1" bayName="Q 01" cNodeName="L 1" />'
              '<Terminal name="T2" cNodeName="grounded" />{lnode}</ConductingEquipment>'
              '<ConnectivityNode name="L 1" pathName="S1/E1/Q 01/L 1" /></Bay></VoltageLevel>{extra}</Substation>')


def _gse(mac="01-0C-CD-01-00-01", appid="0001", vlan="000", priority="4"):
    return (f'<GSE ldInst="CTRL" cbName="Gse1"><Address><P type="MAC-Address">{mac}</P><P type="APPID">{appid}</P>'
            f'<P type="VLAN-ID">{vlan}</P><P type="VLAN-PRIORITY">{priority}</P></Address></GSE>')


def _goose(text, control=GSE_CONTROL, **gse):
    return with_communication(edit(text, "</DataSet>", "</DataSet>" + control), extra=_gse(**gse))


def _substation(text, lnode="", extra="", attributes=""):
    body = SUBSTATION.format(lnode=lnode, extra=extra)
    if attributes:
        body = body.replace('<ConductingEquipment name="QA1" type="CBR">',
                            f'<ConductingEquipment name="QA1" type="CBR"{attributes}>')
    return edit(text, "<IED ", body + "<IED ")


def _extref(text, ref):
    return edit(text, '<LN lnClass="GGIO" inst="1" lnType="StudioAnalog" />',
                f'<LN lnClass="GGIO" inst="1" lnType="StudioAnalog"><Inputs>{ref}</Inputs></LN>')


def cases():
    text = template()
    esld = ' xmlns:esld="https://transpower.co.nz/SCL/SSD/SLD/v0" esld:x="12.5" esld:y="-3.25"'
    items = [
        ("comm_basic", "ConnectedAP with an IP address", with_communication(text)),
        ("comm_no_ip", "ConnectedAP without IP", with_communication(text).replace('<P type="IP">192.168.10.21</P>', "")),
        ("comm_ipv6", "IPv6 address", with_communication(text, ip="fe80::1")),
        ("comm_bad_ip", "address that is not an IP", with_communication(text, ip="300.1.1.1")),
        ("goose_basic", "GSEControl with GSE address", _goose(text)),
        ("goose_mac_lower", "MAC address lower case", _goose(text, mac="01-0c-cd-01-00-01")),
        ("goose_mac_colon", "MAC address with colons", _goose(text, mac="01:0C:CD:01:00:01")),
        ("goose_appid_0x", "APPID with 0x", _goose(text, appid="0x0001")),
        ("goose_appid_decimal", "APPID decimal-looking", _goose(text, appid="4096")),
        ("goose_appid_bad", "APPID not hex", _goose(text, appid="ZZZZ")),
        ("goose_vlan_max", "VLAN-ID FFF", _goose(text, vlan="FFF")),
        ("goose_vlan_over", "VLAN-ID over 12 bits", _goose(text, vlan="1000")),
        ("goose_priority_8", "VLAN-PRIORITY 8", _goose(text, priority="8")),
        ("goose_no_dataset", "GSEControl without datSet",
         _goose(text, control=GSE_CONTROL.replace(' datSet="Events"', ""))),
        ("sv_control", "SampledValueControl", edit(text, "</DataSet>", "</DataSet>" + SMV_CONTROL)),
        ("report_unnamed", "ReportControl without a name", edit(text, 'ReportControl name="EventsRCB"', "ReportControl")),
        ("report_max_zero", "RptEnabled max 0", edit(text, '<TrgOps dchg="true" qchg="true" dupd="false" period="true" gi="true" />',
                                                     '<TrgOps dchg="true" qchg="true" dupd="false" period="true" gi="true" /><RptEnabled max="0" />')),
        ("report_indexed", "indexed report", edit(text, 'indexed="false"', 'indexed="true"')),
        ("extref_basic", "ExtRef to an IED outside the file",
         _extref(text, '<ExtRef iedName="Other" ldInst="LD0" lnClass="PTRC" lnInst="1" doName="Tr" daName="general" '
                       'intAddr="in1" serviceType="GOOSE" />')),
        ("extref_pipe", "intAddr with a pipe", _extref(text, '<ExtRef intAddr="vendor|in|1" />')),
        ("extref_no_ied", "ExtRef without iedName", _extref(text, '<ExtRef ldInst="LD0" lnClass="PTRC" lnInst="1" doName="Tr" />')),
        ("two_access_points", "second AccessPoint with a Server",
         edit(text, "</AccessPoint>", '</AccessPoint><AccessPoint name="AP2"><Server><Authentication /><LDevice inst="B">'
                                      '<LN0 lnClass="LLN0" inst="" lnType="StudioLLN0" /></LDevice></Server></AccessPoint>')),
        ("server_at", "AccessPoint with ServerAt", edit(text, "</AccessPoint>", '</AccessPoint><AccessPoint name="AP3">'
                                                                              '<ServerAt apName="AP1" /></AccessPoint>')),
        ("substation_basic", "Substation with node names containing spaces", _substation(text)),
        ("substation_lnode_absent", "LNode to an absent IED",
         _substation(text, lnode='<LNode iedName="Ghost" ldInst="CTRL" lnClass="XCBR" lnInst="1" />')),
        ("substation_lnode_present", "LNode to the IED in the file",
         _substation(text, lnode='<LNode iedName="Corpus" ldInst="CTRL" lnClass="XCBR" lnInst="1" />')),
        ("substation_esld", "esld x/y floats and negatives", _substation(text, attributes=esld)),
        ("substation_transformer", "power transformer with windings",
         _substation(text, extra='<PowerTransformer name="T1" type="PTR"><TransformerWinding name="W1" type="PTW" />'
                                 '<TransformerWinding name="W2" type="PTW" /></PowerTransformer>')),
        ("substation_two", "two Substation sections",
         _substation(_substation(text).replace('<Substation name="S1"', '<Substation name="S2"', 1))),
    ]
    for case, purpose, value in items:
        yield Case(case, purpose, "template", utf8(value))
