"""Goldens for scl/goose_sources.read_goose_sources and goose_inputs._with_members.

  python studio/engine/internal/scl/testdata/goose_sources_gen.py   -> golden/goose_sources.json

Inputs: every public fixture and corpus case that decodes as UTF-8 (alone, with all IEDs
loaded and with only the first), pairs of GOOSE cases, and synthetic multi-file stations
(publishers and subscribers in different files, an IED copied into two files, remote and
packed vendor inputs). Each entry: {name, documents: [{case | text, loaded}], result | error,
members: [{doc, ied, gocb_ref, result | error}]}. "blocks" is written as [[key, block], ...]
because JSON has no tuple keys. Deterministic; json.dumps defaults, ASCII.
"""
import json
import pathsetup

pathsetup.install()

from golden_common import capture  # noqa: E402
from golden_gen import FIXTURES, CORPUS_PREFIX  # noqa: E402
from studio_app.acquisition.goose_inputs import _with_members  # noqa: E402
from studio_app.scl.document import parse_document  # noqa: E402
from studio_app.scl.goose_sources import read_goose_sources  # noqa: E402

NS = 'xmlns="http://www.iec.ch/61850/2003/SCL" xmlns:ge="http://www.gedigitalenergy.com/multilin"'
FCDA = '<FCDA ldInst="LD0" lnClass="GGIO" lnInst="1" doName="Ind{n}" daName="stVal" fc="ST"/>'


def publisher(name, mac_tail, extra=""):
    return (f'<IED name="{name}" desc="pub {name}" manufacturer="ACME" type="P1"><AccessPoint name="AP1"><Server>'
            f'<LDevice inst="LD0"><LN0 lnClass="LLN0" inst="" lnType="L"><DataSet name="DS1">'
            f'{FCDA.format(n=1)}{FCDA.format(n=2)}</DataSet><DataSet name="DS1">{FCDA.format(n=3)}</DataSet>'
            f'<GSEControl name="GC1" appID="{name}_GC1" confRev=" 7 " datSet="DS1" ge:Label="L1"/>'
            f'<GSEControl name="GC2" type="GSSE" datSet="DS1"/>'
            f'<GSEControl name="GC3" confRev="x" datSet="DS9" ge:Function=" DISABLED "/>'
            f'<GSEControl name="GC4" datSet="DS1" confRev="99999999999999999999"/>{extra}</LN0>'
            f'</LDevice></Server></AccessPoint></IED>'), (
            f'<ConnectedAP iedName="{name}" apName="AP1"><GSE ldInst="LD0" cbName="GC1"><Address>'
            f'<P type="MAC-Address">01-0C-CD-01-00-{mac_tail}</P><P type="APPID">00{mac_tail}</P>'
            f'<P type="VLAN-ID">005</P><P type="VLAN-PRIORITY">4</P></Address></GSE></ConnectedAP>')


def subscriber(name, inputs):
    return (f'<IED name="{name}"><AccessPoint name="AP1"><Server><LDevice inst="LD0">'
            f'<LN0 lnClass="LLN0" inst="" lnType="L"/><LN lnClass="LGOS" inst="1" lnType="G"><DOI name="GoCBRef">'
            f'<DAI name="setSrcRef"><Val> PUB1LD0/LLN0$GO$GC1 </Val></DAI></DOI></LN>'
            f'<LN lnClass="GGIO" inst="3" lnType="G"><Inputs>{inputs}</Inputs></LN></LDevice></Server>'
            f'</AccessPoint></IED>')


def extref(ied, n, **extra):
    attributes = "".join(f' {key}="{value}"' for key, value in extra.items())
    return (f'<ExtRef iedName="{ied}" ldInst="LD0" lnClass="GGIO" lnInst="1" doName="Ind{n}" daName="stVal"'
            f'{attributes}/>')


def document(ieds, access=""):
    return (f'<?xml version="1.0" encoding="UTF-8"?><SCL {NS} version="2007" revision="B"><Header id="t"/>'
            f'<Communication><SubNetwork name="N1">{access}</SubNetwork></Communication>{ieds}</SCL>')


SUB_INPUTS = "".join([
    extref("PUB1", 1, serviceType="GOOSE", srcCBName="GC1", intAddr="in1", desc="d1"),
    extref("PUB1", 2, desc="data match"),
    extref("PUB1", 3, srcCBName="GC9", srcLDInst="LD0"),
    extref("PUB2", 1, srcCBName="GC1"),
    extref("REMOTE1", 1, serviceType="GOOSE", srcCBName="X", srcLDInst="LDX"),
    extref("REMOTE2", 1, intAddr="LD0|REMOTE2|1|01:0c:cd:01:00:0a|Bool|x"),
    extref("REMOTE3", 1, intAddr="LD0|OTHER|1|01:0C:CD:01:00:0B|Bool"),
    extref("REMOTE4", 1, serviceType="Report", srcCBName="R"),
    extref("REMOTE5", 2, serviceType="GOOSE"),
    '<ExtRef intAddr="spare" desc="spare input"/>',
])


def synthetic():
    pub1, access1 = publisher("PUB1", "01")
    pub2, access2 = publisher("PUB2", "02")
    sub = subscriber("SUB1", SUB_INPUTS)
    pubs = document(pub1 + pub2, access1 + access2)
    subs = document(sub)
    both = document(sub + pub1, access1)
    return [
        ("split", [(pubs, ["PUB1", "PUB2"]), (subs, ["SUB1"])]),
        ("split none loaded", [(subs, []), (pubs, [])]),
        ("copy loaded from second", [(both, ["SUB1"]), (pubs, ["PUB1"])]),
        ("copy loaded from first", [(both, ["SUB1", "PUB1"]), (pubs, ["PUB2"])]),
        ("subscriber alone", [(subs, ["SUB1"])]),
        ("same file twice", [(pubs, ["PUB1"]), (pubs, ["PUB2"])]),
        ("broken second", [(pubs, ["PUB1"]), ("<SCL", ["X"])]),
        ("empty", []),
    ]


def names_of(text):
    ok, root = capture(parse_document, text)
    return [ied.get("name") for ied in root.findall("IED")] if ok else []


def members(documents):
    found = []
    for index, (text, _) in enumerate(documents):
        for name in names_of(text):
            ok, result = capture(read_goose_sources, [(text, [name])])
            refs = [item["gocb_ref"] for item in result["subscriptions"].get(name, [])] if ok else []
            for ref in refs + ["NOPE/LLN0$GO$X"]:
                ok, value = capture(_with_members, {"source": text, "name": name}, {"gocb_ref": ref})
                found.append({"doc": index, "ied": name, "gocb_ref": ref, "result" if ok else "error": value})
    return found


def entry(name, documents, refs):
    ok, result = capture(read_goose_sources, documents)
    if ok:
        result["blocks"] = [[list(key), block] for key, block in result["blocks"].items()]
    return {"name": name, "documents": refs, "result" if ok else "error": result, "members": members(documents)}


def corpus():
    cases = list(FIXTURES)
    manifest = json.loads((pathsetup.CORPUS / "manifest.json").read_bytes())
    cases += [(item["case"], CORPUS_PREFIX + item["case"] + ".scd") for item in manifest]
    texts = {}
    for case, path in cases:
        try:
            texts[case] = ((pathsetup.REPO / path).read_bytes().decode("utf-8-sig"), path)
        except UnicodeDecodeError:
            continue
    return texts


def main():
    entries, texts = [], corpus()
    for case, (text, path) in texts.items():
        names = names_of(text)
        entries.append(entry(case + " all", [(text, names)], [{"case": path, "loaded": names}]))
        entries.append(entry(case + " first", [(text, names[:1])], [{"case": path, "loaded": names[:1]}]))
    goose = [case for case in texts if case.startswith(("goose_", "gse_ge", "ext_", "fixture_"))]
    for first in goose:
        for second in goose:
            if first < second:
                pair = [(texts[first][0], names_of(texts[first][0])[:1]), (texts[second][0], names_of(texts[second][0]))]
                refs = [{"case": texts[c][1], "loaded": loaded} for c, (_, loaded) in zip((first, second), pair)]
                entries.append(entry(f"{first} + {second}", pair, refs))
    for name, documents in synthetic():
        entries.append(entry(name, documents, [{"text": text, "loaded": loaded} for text, loaded in documents]))
    data = json.dumps(entries).encode("ascii")
    pathsetup.write_bytes(pathsetup.GOLDEN / "goose_sources.json", data)
    print(len(entries), "entries,", sum(len(item["members"]) for item in entries), "member lookups")


if __name__ == "__main__":
    main()
