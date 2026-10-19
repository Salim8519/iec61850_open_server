"""XML features: DTDs and entities, comments, PIs, CDATA, namespaces, Private, editions."""
import re

from parity.corpus.base import Case, edit, template, utf8

DECLARATION = "<?xml version='1.0' encoding='utf-8'?>"
ROOT = '<SCL xmlns="http://www.iec.ch/61850/2003/SCL" version="2007" revision="B" release="4">'


def _doctype(text, doctype):
    return edit(text, DECLARATION + "\n", DECLARATION + "\n" + doctype + "\n")


def _prefixed(text):
    text = re.sub(r"<(/?)([A-Za-z])", r"<\1scl:\2", text)
    return text.replace('<scl:SCL xmlns="', '<scl:SCL xmlns:scl="', 1)


def _root(text, attributes):
    return edit(text, ROOT, f'<SCL xmlns="http://www.iec.ch/61850/2003/SCL"{attributes}>')


def _ied(text, extra):
    return edit(text, '<IED name="Corpus"', f'<IED name="Corpus"{extra}')


def cases():
    text = template()
    desc = 'desc="Simulation template; not a manufacturer relay model"'
    entity = edit(text, desc, 'desc="&e;"')
    items = [
        ("doctype_plain", "<!DOCTYPE SCL> without an internal subset", _doctype(text, "<!DOCTYPE SCL>")),
        ("entity_internal", "internal entity (defusedxml EntitiesForbidden)",
         _doctype(entity, '<!DOCTYPE SCL [<!ENTITY e "x">]>')),
        ("entity_external", "external SYSTEM entity",
         _doctype(entity, '<!DOCTYPE SCL [<!ENTITY e SYSTEM "file:///C:/Windows/win.ini">]>')),
        ("entity_laughs", "nested entity expansion (billion laughs, small)",
         _doctype(entity, '<!DOCTYPE SCL [<!ENTITY a "lol"><!ENTITY b "&a;&a;&a;"><!ENTITY e "&b;&b;&b;">]>')),
        ("entity_parameter", "parameter entity", _doctype(text, '<!DOCTYPE SCL [<!ENTITY % p "x"> %p;]>')),
        ("xinclude", "XInclude element kept as data, not processed",
         edit(text, "<AccessPoint", '<xi:include xmlns:xi="http://www.w3.org/2001/XInclude" href="x.xml"/><AccessPoint')),
        ("comment_before_root", "comment before the root", edit(text, ROOT, "<!-- public corpus -->" + ROOT)),
        ("comment_in_ied", "comment inside IED", edit(text, "<AccessPoint", "<!-- in IED --><AccessPoint")),
        ("comment_in_templates", "comment inside DataTypeTemplates",
         edit(text, "<DataTypeTemplates>", "<DataTypeTemplates><!-- types -->")),
        ("comment_between_val", "comment between Val elements",
         edit(text, "<Val>ScadaBench</Val>", "<!-- a --><Val>ScadaBench</Val><!-- b -->")),
        ("comment_dashes", "comment with dash-adjacent text", edit(text, "<AccessPoint", "<!-- a - b -  --><AccessPoint")),
        ("comment_non_ascii", "non-ASCII comment", edit(text, "<AccessPoint", "<!-- \u00c4\u00f6 \u0645\u0631\u062d\u0628\u0627 --><AccessPoint")),
        ("processing_instruction", "PI inside IED", edit(text, "<AccessPoint", "<?vendor keep this?><AccessPoint")),
        ("cdata_val", "CDATA inside Val", edit(text, "<Val>ScadaBench</Val>", "<Val><![CDATA[ScadaBench]]></Val>")),
        ("space_preserve", "xml:space preserve on Val",
         edit(text, "<Val>ScadaBench</Val>", '<Val xml:space="preserve">  ScadaBench  </Val>')),
        ("escaped_desc", "escaped characters in desc", edit(text, desc, 'desc="&lt;&amp;&gt;&quot;&apos;"')),
        ("scl_prefix", "scl: prefix on every element", _prefixed(text)),
        ("default_ns_redeclared", "default namespace redeclared on IED",
         _ied(text, ' xmlns="http://www.iec.ch/61850/2003/SCL"')),
        ("foreign_attributes", "vendor attributes (ge:Function, esld:x)",
         _ied(text, ' xmlns:ge="urn:ge" ge:Function="Feeder" xmlns:esld="https://transpower.co.nz/SCL/SSD/SLD/v0" esld:x="1.5"')),
        ("private_foreign", "Private element with foreign children",
         edit(text, "<AccessPoint", '<Private type="vendor-settings"><v:item xmlns:v="urn:v" a="1">data</v:item></Private><AccessPoint')),
        ("version_absent", "no version attributes", _root(text, "")),
        ("version_2003", "2003 edition", _root(text, ' version="2003"')),
        ("version_2007a", "2007 revision A", _root(text, ' version="2007" revision="A"')),
        ("version_2007b", "2007 revision B without release", _root(text, ' version="2007" revision="B"')),
        ("version_unknown", "unknown edition", _root(text, ' version="2099" revision="Z"')),
    ]
    for case, purpose, value in items:
        yield Case(case, purpose, "template", utf8(value))
