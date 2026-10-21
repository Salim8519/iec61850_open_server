"""Building blocks for corpus cases: the sample IED text and small, reviewable edits."""
from pathlib import Path

REPO = Path(__file__).resolve().parents[4]
PUBLIC = {"open_substation": "scd/open_substation.scd", "model_substation": "scd/model_substation.scd",
          "protection_relay": "scd/protection_relay.scd", "demo": "marketing/videos/demo/scadabench_demo.scd"}
NAME = "Corpus"
COMMUNICATION = ('<Communication><SubNetwork name="StationBus" type="8-MMS"><ConnectedAP iedName="Corpus" '
                 'apName="AP1"><Address><P type="IP">{ip}</P><P type="IP-SUBNET">255.255.255.0</P></Address>'
                 '{extra}</ConnectedAP></SubNetwork></Communication>')


def template(name=NAME):
    from studio_app.scl.templates import make_template
    return make_template(name).decode("utf-8")


def public(alias):
    return (REPO / PUBLIC[alias]).read_bytes()


def edit(text, old, new, count=1):
    """Replace `old` (which must exist) `count` times; a missing anchor is a generator bug."""
    if old not in text:
        raise ValueError(f"corpus anchor not found: {old[:60]!r}")
    return text.replace(old, new, count)


def with_communication(text, ip="192.168.10.21", extra=""):
    return edit(text, "<IED ", COMMUNICATION.format(ip=ip, extra=extra) + "<IED ")


def status_value(text, kind, value):
    """The INT32 stVal of the INS status type, re-typed to `kind` with initial `value`."""
    old = '<DA name="stVal" bType="INT32" fc="ST" dchg="true"><Val>1</Val></DA>'
    return edit(text, old, f'<DA name="stVal" bType="{kind}" fc="ST" dchg="true"><Val>{value}</Val></DA>')


def float_value(text, value):
    return edit(text, '<BDA name="f" bType="FLOAT32"><Val>0</Val></BDA>',
                f'<BDA name="f" bType="FLOAT32"><Val>{value}</Val></BDA>')


def utf8(text):
    return text.encode("utf-8")


class Case:
    def __init__(self, case, purpose, base, data, generated=False):
        self.case, self.purpose, self.base, self.data, self.generated = case, purpose, base, data, generated
