"""Corpus cases about the Communication section, GOOSE and Sampled Values blocks."""
from corpus_base import (LN0_END, case, connected_ap, gse, gse_control, min_max, sub, template_text,
                         with_communication, with_goose, with_second_ied)

BASE = "make_template(BayController)"
GE = "http://www.gedigitalenergy.com/multilin"
ROOT = '<SCL xmlns="http://www.iec.ch/61850/2003/SCL"'


def _comm(*aps, text=None):
    return with_communication(text or template_text(), list(aps))


def _goose(**options):
    """Template with GC1; options go to the GSE address builder."""
    return with_goose(template_text(), gse_element=gse(**options))


def _control(control, gse_element=None):
    text = sub(template_text(), ROOT, f'{ROOT} xmlns:ge="{GE}"')
    return with_goose(text, control=control, gse_element=gse_element)


def _sv_dataset(fcdas):
    return '<DataSet name="SvData">' + "".join(fcdas) + "</DataSet>"


def _fcda(ln_class, inst, do="AmpSv", da=None):
    attribute = f' daName="{da}"' if da else ""
    return f'<FCDA ldInst="CTRL" lnClass="{ln_class}" lnInst="{inst}" doName="{do}"{attribute} fc="MX" />'


LE8 = [_fcda("TCTR", i) for i in range(1, 5)] + [_fcda("TVTR", i, "VolSv") for i in range(1, 5)]
LE16 = [item for i in range(1, 5) for item in (_fcda("TCTR", i, "AmpSv", "instMag.i"), _fcda("TCTR", i, "AmpSv", "q"))] + \
       [item for i in range(1, 5) for item in (_fcda("TVTR", i, "VolSv", "instMag.i"), _fcda("TVTR", i, "VolSv", "q"))]


def _sv(fcdas, control_attrs='smpRate="80" nofASDU="1" confRev="1"', address=True, smp_mod=""):
    text = template_text()
    control = (f'<SampledValueControl name="MSVCB01" smvID="BayController_SV" datSet="SvData" {control_attrs}'
               f'{smp_mod} multicast="true"><SmvOpts refreshTime="false" sampleSynchronized="true" /></SampledValueControl>')
    text = sub(text, '<DataSet name="Events">', _sv_dataset(fcdas) + '<DataSet name="Events">')
    text = sub(text, LN0_END, "</ReportControl>" + control + "</LN0>")
    smv = gse("MSVCB01", "01-0C-CD-04-00-01", "4001", "000", "4", tag="SMV") if address else ""
    return with_communication(text, [connected_ap(extra=smv)])


def comm_cases():
    two = with_second_ied(template_text())
    return [
        case("comm_ip_basic", "ConnectedAP with IP, subnet and gateway", BASE, _comm(connected_ap())),
        case("comm_no_ip", "ConnectedAP without an Address", BASE, _comm(connected_ap(ip=None))),
        case("comm_invalid_ip", "ConnectedAP with an invalid IPv4 address", BASE, _comm(connected_ap(ip="10.0.0.256"))),
        case("comm_ipv6", "ConnectedAP with an IPv6 address", BASE, _comm(connected_ap(ip="fe80::1"))),
        case("comm_ip_leading_zero", "IPv4 with a leading zero octet", BASE, _comm(connected_ap(ip="010.0.0.1"))),
        case("comm_ip_padded", "IPv4 with surrounding whitespace in the P text", BASE, _comm(connected_ap(ip=" 10.0.0.9 "))),
        case("comm_other_ied", "ConnectedAP only for an IED that is not in the file", BASE,
             _comm(connected_ap(ied="Elsewhere"))),
        case("comm_duplicate_ips", "two IEDs with the same IP", BASE,
             _comm(connected_ap(), connected_ap(ied="Bay_2"), text=two)),
        case("comm_two_connected_aps", "one IED on two ConnectedAPs: first IP wins", BASE,
             _comm(connected_ap(ip="10.0.0.5"), connected_ap(ip="10.0.0.6", ap="AP2"))),
        case("gse_mac_lower", "GSE MAC-Address in lower case", BASE, _goose(mac="01-0c-cd-01-00-0a")),
        case("gse_mac_colon", "GSE MAC-Address with colons (warning)", BASE, _goose(mac="01:0C:CD:01:00:01")),
        case("gse_mac_missing", "GSE without MAC-Address", BASE, _goose(mac=None)),
        case("gse_appid_0x", "APPID with 0x prefix", BASE, _goose(appid="0x3001")),
        case("gse_appid_decimal_looking", "APPID 12289 read as hex", BASE, _goose(appid="12289")),
        case("gse_appid_garbage", "APPID that is not hex", BASE, _goose(appid="zz")),
        case("gse_appid_too_big", "APPID above 0xFFFF", BASE, _goose(appid="10000")),
        case("gse_appid_missing", "GSE without APPID", BASE, _goose(appid=None)),
        case("gse_vlan_fff", "VLAN-ID FFF (maximum)", BASE, _goose(vlan="FFF")),
        case("gse_vlan_too_big", "VLAN-ID 1000 hex (out of range)", BASE, _goose(vlan="1000")),
        case("gse_vlan_empty", "VLAN-ID empty text", BASE, _goose(vlan="")),
        case("gse_priority_7", "VLAN-PRIORITY 7", BASE, _goose(prio="7")),
        case("gse_priority_8", "VLAN-PRIORITY 8 (out of range)", BASE, _goose(prio="8")),
        case("gse_priority_negative", "VLAN-PRIORITY -1", BASE, _goose(prio="-1")),
        case("gse_times", "MinTime 4 and MaxTime 1000", BASE, _goose(times=min_max())),
        case("gse_times_fraction", "MinTime 4.6 and MaxTime 2.5 (round half even)", BASE, _goose(times=min_max("4.6", "2.5"))),
        case("gse_times_bad", "MinTime negative and MaxTime text", BASE, _goose(times=min_max("-1", "abc"))),
        case("gse_no_address", "GSEControl without a GSE address", BASE,
             with_goose(template_text(), gse_element="")),
        case("gse_no_dataset", "GSEControl without datSet", BASE, _control(gse_control(dataset=None))),
        case("gse_unknown_dataset", "GSEControl datSet not in LN0", BASE, _control(gse_control(dataset="Nope"))),
        case("gse_type_gsse", "GSEControl type GSSE", BASE,
             _control(gse_control().replace('type="GOOSE"', 'type="GSSE"'))),
        case("gse_ge_disabled", "GSEControl with ge:Function=Disabled", BASE,
             _control(gse_control(extra=' ge:Function="Disabled" ge:Label="Tx GC1"'))),
        case("gse_ge_enabled", "GSEControl with ge:Function=Enabled", BASE,
             _control(gse_control(extra=' ge:Function=" ENABLED "'))),
        case("gse_bad_confrev", "GSEControl confRev not an integer", BASE,
             _control(gse_control().replace('confRev="1"', 'confRev="1.5"'))),
        case("gse_goid_space", "GSEControl appID with a space", BASE, _control(gse_control(app_id="Bay GC1"))),
        case("gse_goid_empty", "GSEControl appID empty: GoCB reference is used", BASE, _control(gse_control(app_id=""))),
        case("gse_fixed_offs", "GSEControl fixedOffs=true", BASE, _control(gse_control(extra=' fixedOffs="true"'))),
    ]


def sv_cases():
    return [
        case("sv_92le_8", "9-2LE dataset with 8 whole-object FCDAs", BASE, _sv(LE8)),
        case("sv_92le_16", "9-2LE dataset with 16 instMag.i/q FCDAs", BASE, _sv(LE16)),
        case("sv_raw", "SV dataset that is not a 9-2LE layout", BASE, _sv(LE8[:5])),
        case("sv_wrong_lnclass", "8 FCDAs but one is not TCTR/TVTR", BASE, _sv(LE8[:7] + [_fcda("GGIO", 1)])),
        case("sv_no_address", "SampledValueControl without SMV address", BASE, _sv(LE8, address=False)),
        case("sv_bad_rate", "smpRate not an integer", BASE, _sv(LE8, 'smpRate="4k" nofASDU="1" confRev="1"')),
        case("sv_smp_per_sec", "smpMod SmpPerSec with 8 ASDUs", BASE,
             _sv(LE8, 'smpRate="4800" nofASDU="8" confRev="2"', smp_mod=' smpMod="SmpPerSec"')),
    ]
