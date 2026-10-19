"""Public base documents and text-substitution helpers for the synthetic SCL corpus."""
from studio_app.scl.templates import make_template

IED = "BayController"
HEADER = '<Header id="BayController" version="1" revision="0" />'
LN0_END = "</ReportControl></LN0>"
GGIO = '<LN lnClass="GGIO" inst="1" lnType="StudioAnalog" />'
TEMPLATES_END = "</DataTypeTemplates>"
IED_DESC = 'desc="Simulation template; not a manufacturer relay model"'


def template_text(name=IED):
    return make_template(name).decode("utf-8")


def sub(text, old, new, count=1):
    """Replace exactly `count` occurrences (0 = all); fail loudly when the anchor moved."""
    found = text.count(old)
    if found == 0 or (count and found < count):
        raise AssertionError(f"corpus anchor not found: {old[:60]!r}")
    return text.replace(old, new) if count == 0 else text.replace(old, new, count)


def variant(*pairs):
    """Template text with each (old, new) pair applied once, in order."""
    text = template_text()
    for old, new in pairs:
        text = sub(text, old, new)
    return text


def pretty(text):
    """One element per line (LF), so libxml2 line numbers become meaningful."""
    return text.replace("><", ">\n<")


def ied_element(text):
    start = text.index("<IED ")
    end = text.index("</IED>") + len("</IED>")
    return text[start:end]


def with_second_ied(text, name="Bay_2", body=None):
    """Append a renamed copy of the first IED (or `body`) directly after it."""
    element = ied_element(text)
    copy = body if body is not None else element.replace(f'name="{IED}"', f'name="{name}"', 1)
    return text.replace(element, element + copy, 1)


def with_communication(text, aps, subnetwork="LAN1"):
    """Insert a Communication section after the Header; aps are ConnectedAP strings."""
    section = f'<Communication><SubNetwork name="{subnetwork}" type="8-MMS">{"".join(aps)}</SubNetwork></Communication>'
    return sub(text, HEADER, HEADER + section)


def address(params):
    return "<Address>" + "".join(f'<P type="{key}">{value}</P>' for key, value in params) + "</Address>"


def connected_ap(ied=IED, ip="10.0.0.5", extra="", ap="AP1"):
    params = [("IP", ip), ("IP-SUBNET", "255.255.255.0"), ("IP-GATEWAY", "10.0.0.1")] if ip is not None else []
    body = address(params) if params else ""
    return f'<ConnectedAP iedName="{ied}" apName="{ap}">{body}{extra}</ConnectedAP>'


def gse(cb="GC1", mac="01-0C-CD-01-00-01", appid="3001", vlan="000", prio="4", times="", ld="CTRL", tag="GSE"):
    params = [("MAC-Address", mac), ("APPID", appid), ("VLAN-ID", vlan), ("VLAN-PRIORITY", prio)]
    params = [(key, value) for key, value in params if value is not None]
    return f'<{tag} ldInst="{ld}" cbName="{cb}">{address(params)}{times}</{tag}>'


def min_max(minimum="4", maximum="1000"):
    return (f'<MinTime unit="s" multiplier="m">{minimum}</MinTime>'
            f'<MaxTime unit="s" multiplier="m">{maximum}</MaxTime>')


def gse_control(name="GC1", dataset="Events", extra="", app_id=None):
    goid = f' appID="{app_id}"' if app_id is not None else f' appID="{IED}_{name}"'
    data = f' datSet="{dataset}"' if dataset is not None else ""
    return f'<GSEControl name="{name}" type="GOOSE"{goid} confRev="1"{data}{extra} />'


def with_goose(text, control=None, gse_element=None, ip="10.0.0.5"):
    """Template with one GSEControl in LN0 and its GSE address in Communication."""
    text = sub(text, LN0_END, "</ReportControl>" + (control or gse_control()) + "</LN0>")
    extra = gse() if gse_element is None else gse_element
    return with_communication(text, [connected_ap(ip=ip, extra=extra)])


def with_types(text, declarations):
    """Append DOType/DAType/EnumType declarations to DataTypeTemplates."""
    return sub(text, TEMPLATES_END, declarations + TEMPLATES_END)


def with_inputs(text, extrefs, ln=GGIO):
    """Give GGIO1 an Inputs section holding the ExtRef strings."""
    opened = ln.replace(" />", ">")
    return sub(text, ln, f"{opened}<Inputs>{''.join(extrefs)}</Inputs></LN>")


def extref(ied=IED, cb="GC1", do="Pos", da="stVal", ln_class="XCBR", inst="1", extra="", service="GOOSE"):
    named = f' srcLDInst="CTRL" srcLNClass="LLN0" srcCBName="{cb}"' if cb else ""
    who = f' iedName="{ied}"' if ied is not None else ""
    kind = f' serviceType="{service}"' if service else ""
    return (f'<ExtRef{who} ldInst="CTRL" lnClass="{ln_class}" lnInst="{inst}" doName="{do}" '
            f'daName="{da}"{kind}{named} intAddr="In{do}"{extra} />')


def encode(text):
    return text.encode("utf-8")


def case(name, purpose, base, data):
    if isinstance(data, str):
        data = encode(data)
    return {"case": name, "purpose": purpose, "base": base, "data": data}
