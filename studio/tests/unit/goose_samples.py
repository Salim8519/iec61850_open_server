"""Small synthetic SCL documents for GOOSE link-map tests (no customer data)."""

GE = "http://www.gedigitalenergy.com/multilin"


def gse(ied, cb, mac, app_id, ld="LD0"):
    return (f'<ConnectedAP iedName="{ied}" apName="AP1"><GSE ldInst="{ld}" cbName="{cb}"><Address>'
            f'<P type="MAC-Address">{mac}</P><P type="APPID">{app_id}</P><P type="VLAN-ID">000</P>'
            f'<P type="VLAN-PRIORITY">4</P></Address></GSE></ConnectedAP>')


def publisher(name, blocks, ld="LD0"):
    """blocks: [(cb name, dataset or '', ge Function or None)]"""
    controls, datasets = [], []
    for cb, dataset, function in blocks:
        vendor = f' ge:Function="{function}" ge:Label="Tx {cb}"' if function else ""
        attribute = f' datSet="{dataset}"' if dataset else ""
        controls.append(f'<GSEControl name="{cb}" type="GOOSE" appID="{name}_{cb}" confRev="1"{attribute}{vendor}/>')
        if dataset:
            datasets.append(f'<DataSet name="{dataset}"><FCDA ldInst="{ld}" lnClass="GGIO" lnInst="1" '
                            f'doName="Ind1" daName="stVal" fc="ST"/><FCDA ldInst="{ld}" lnClass="GGIO" '
                            f'lnInst="1" doName="Ind2" daName="stVal" fc="ST"/></DataSet>')
    return (f'<IED name="{name}"><AccessPoint name="AP1"><Server><LDevice inst="{ld}">'
            f'<LN0 lnClass="LLN0" inst="" lnType="L">{"".join(datasets)}{"".join(controls)}</LN0>'
            f'</LDevice></Server></AccessPoint></IED>')


def subscriber(name, extrefs, ld="LD0"):
    return (f'<IED name="{name}"><AccessPoint name="AP1"><Server><LDevice inst="{ld}">'
            f'<LN0 lnClass="LLN0" inst="" lnType="L"/><LN lnClass="GGIO" inst="3" lnType="G"><Inputs>'
            f'{"".join(extrefs)}</Inputs></LN></LDevice></Server></AccessPoint></IED>')


def named_input(publisher_name, cb, do="Ind1", int_addr="GGIO3.Ind1.stVal", ld="LD0"):
    return (f'<ExtRef iedName="{publisher_name}" ldInst="{ld}" lnClass="GGIO" lnInst="1" doName="{do}" '
            f'daName="stVal" serviceType="GOOSE" srcLDInst="{ld}" srcLNClass="LLN0" srcCBName="{cb}" '
            f'intAddr="{int_addr}" desc="input {do}"/>')


def packed_input(publisher_name, mac="01:0C:CD:01:00:00"):
    return (f'<ExtRef iedName="{publisher_name}" ldInst="CON" lnClass="GGIO" lnInst="1" doName="Ind1" '
            f'daName="stVal" intAddr="CON|{publisher_name}|1|{mac}|Bool|0,0x0,,Ind1.stVal,0"/>')


def document(ieds, addresses):
    return (f'<?xml version="1.0" encoding="UTF-8"?><SCL xmlns="http://www.iec.ch/61850/2003/SCL" '
            f'xmlns:ge="{GE}" version="2007" revision="B"><Header id="t"/><Communication>'
            f'<SubNetwork name="N1">{"".join(addresses)}</SubNetwork></Communication>{"".join(ieds)}</SCL>')
