"""Corpus cases about XML features (DTDs, comments, namespaces) and SCL editions."""
import re
from corpus_base import HEADER, IED_DESC, case, ied_element, sub, template_text, variant

BASE = "make_template(BayController)"
ROOT = '<SCL xmlns="http://www.iec.ch/61850/2003/SCL" version="2007" revision="B" release="4">'
NS = "http://www.iec.ch/61850/2003/SCL"


def _before_root(text, prolog):
    return sub(text, ROOT, prolog + ROOT)


def _root(attributes):
    return variant((ROOT, f'<SCL xmlns="{NS}"{attributes}>'))


def _prefixed():
    text = template_text().replace(f'xmlns="{NS}"', f'xmlns:scl="{NS}"', 1)
    return re.sub(r"<(/?)([A-Za-z])", r"<\1scl:\2", text)


def _comments():
    text = template_text()
    text = _before_root(text, "<!-- before root: bay \u00c4 \u2603 -->\n")
    text = sub(text, '<AccessPoint name="AP1">', '<!-- inside IED - single-dash text --><AccessPoint name="AP1">')
    text = sub(text, "<DataTypeTemplates>", "<DataTypeTemplates><!--types\u00e9-->")
    text = sub(text, "<DAI name=\"stVal\"><Val>1</Val>", "<DAI name=\"stVal\"><!--a--><Val>1</Val><!--b-->")
    return sub(text, "</SCL>", "</SCL>\n<!-- after root -->")


def _entity(subset, use="&x;"):
    text = _before_root(template_text(), f"<!DOCTYPE SCL [{subset}]>\n")
    return sub(text, IED_DESC, f'desc="{use}"') if use else text


def _private(content):
    return variant(('<AccessPoint name="AP1">', content + '<AccessPoint name="AP1">'))


LAUGHS = ('<!ENTITY a "lol"><!ENTITY b "&a;&a;&a;&a;&a;&a;&a;&a;&a;&a;">'
          '<!ENTITY c "&b;&b;&b;&b;&b;&b;&b;&b;&b;&b;"><!ENTITY d "&c;&c;&c;&c;&c;&c;&c;&c;&c;&c;">')


def xml_cases():
    text = template_text()
    foreign = ('<Private type="vendor-x" source="tool"><v:Setting xmlns:v="urn:vendor" v:id="7">'
               '<v:Value>12</v:Value></v:Setting></Private>')
    return [
        case("xml_doctype_no_subset", "<!DOCTYPE SCL> without internal subset", BASE,
             _before_root(text, "<!DOCTYPE SCL>\n")),
        case("xml_doctype_empty_subset", "<!DOCTYPE SCL []> with an empty subset", BASE,
             _before_root(text, "<!DOCTYPE SCL []>\n")),
        case("xml_internal_entity", "internal entity declared and used in desc", BASE,
             _entity('<!ENTITY x "expanded">')),
        case("xml_internal_entity_unused", "internal entity declared, never used", BASE,
             _entity('<!ENTITY x "expanded">', use=None)),
        case("xml_external_entity", "external SYSTEM entity to a local file", BASE,
             _entity('<!ENTITY x SYSTEM "file:///c:/windows/win.ini">')),
        case("xml_billion_laughs", "nested entity expansion bomb", BASE, _entity(LAUGHS, "&d;")),
        case("xml_parameter_entity", "parameter entity in the internal subset", BASE,
             _entity('<!ENTITY % p "<!ENTITY x \'y\'>"> %p;')),
        case("xml_doctype_public", "DOCTYPE with a PUBLIC external id and no subset", BASE,
             _before_root(text, '<!DOCTYPE SCL PUBLIC "-//X//SCL//EN" "scl.dtd">\n')),
        case("xml_undefined_entity", "undefined entity reference without a DOCTYPE", BASE,
             sub(text, IED_DESC, 'desc="&nbsp;"')),
        case("xml_xinclude", "XInclude element kept as data inside the IED", BASE,
             _private('<xi:include xmlns:xi="http://www.w3.org/2001/XInclude" href="other.xml" />')),
        case("xml_comments_everywhere", "comments before root, in IED, types, around Val, after root", BASE,
             _comments()),
        case("xml_comment_double_hyphen", "comment containing -- (not well-formed)", BASE,
             sub(text, '<AccessPoint name="AP1">', '<!-- a -- b --><AccessPoint name="AP1">')),
        case("xml_comment_before_declaration", "comment placed before the XML declaration", BASE,
             "<!-- first -->" + text),
        case("xml_processing_instructions", "PIs before the root and inside the IED", BASE,
             _private("<?vendor-tool keep this?>").replace(ROOT, "<?app data?>\n" + ROOT, 1)),
        case("xml_pi_xml_reserved", "processing instruction named xml-stylesheet before root", BASE,
             _before_root(text, '<?xml-stylesheet type="text/xsl" href="s.xsl"?>\n')),
        case("xml_cdata_val", "CDATA section inside Val texts", BASE,
             sub(sub(text, "<Val>1</Val></DAI>", "<Val><![CDATA[1]]></Val></DAI>"),
                 "<Val>ScadaBench</Val>", "<Val><![CDATA[Scada<&>Bench]]></Val>")),
        case("xml_space_preserve", "xml:space=preserve on a Val with padding", BASE,
             sub(text, "<Val>1</Val></DAI>", '<Val xml:space="preserve">  1  </Val></DAI>')),
        case("xml_escaped_desc", "predefined entities in desc", BASE,
             sub(text, IED_DESC, 'desc="&lt;&amp;&gt;&quot;&apos; done"')),
        case("xml_charrefs_desc", "decimal and hex character references in desc", BASE,
             sub(text, IED_DESC, 'desc="&#65;&#x42;&#x10FFFF;"')),
        case("xml_prefix_everywhere", "scl: prefix on every element", BASE, _prefixed()),
        case("xml_default_ns_redeclared", "default namespace redeclared on a Private child", BASE,
             _private('<Private type="x"><Foo xmlns="urn:other"><Bar a="1" /></Foo></Private>')),
        case("xml_ied_foreign_namespace", "the IED element moved to a foreign default namespace", BASE,
             sub(text, '<IED name="BayController"', '<IED xmlns="urn:other" name="BayController"')),
        case("xml_no_namespace", "SCL root without the IEC namespace", BASE,
             sub(text, f' xmlns="{NS}"', "")),
        case("xml_foreign_attributes", "foreign namespaced attributes on IED, LN and DA", BASE,
             variant((ROOT, ROOT.replace(">", ' xmlns:ge="http://www.gedigitalenergy.com/multilin" '
                                         'xmlns:esld="https://transpower.co.nz/SCL/SSD/SLD/v0">')),
                     ('<IED name="BayController"', '<IED esld:x="1.5" ge:Function="Enabled" name="BayController"'),
                     ('<LN lnClass="XCBR"', '<LN esld:y="-2" lnClass="XCBR"'))),
        case("xml_private_foreign", "Private with type and foreign children in IED and LN", BASE,
             sub(_private(foreign), '<LN lnClass="LPHD" inst="1" lnType="StudioLPHD" />',
                 f'<LN lnClass="LPHD" inst="1" lnType="StudioLPHD">{foreign}</LN>')),
        case("xml_unbound_prefix", "attribute with an undeclared prefix", BASE,
             sub(text, '<IED name="BayController"', '<IED esld:x="1" name="BayController"')),
    ]


def edition_cases():
    return [
        case("ed_version_absent", "no version: edition 1, SCL2003 schema", BASE, _root("")),
        case("ed_2003", "version=2003", BASE, _root(' version="2003"')),
        case("ed_2007", "version=2007 without revision: unsupported", BASE, _root(' version="2007"')),
        case("ed_2007A", "2007 revision A: unsupported", BASE, _root(' version="2007" revision="A"')),
        case("ed_2007B", "2007 revision B without release", BASE, _root(' version="2007" revision="B"')),
        case("ed_2007B_release_empty", "2007B with release=''", BASE,
             _root(' version="2007" revision="B" release=""')),
        case("ed_2007B_release_3", "2007B release 3: unsupported", BASE,
             _root(' version="2007" revision="B" release="3"')),
        case("ed_2007C", "2007 revision C: unsupported", BASE, _root(' version="2007" revision="C" release="5"')),
        case("ed_unknown", "unknown version string", BASE, _root(' version="2099" revision="Z"')),
        case("ed_2007B_release_04", "2007B release '04': not the string '4', unsupported", BASE,
             _root(' version="2007" revision="B" release="04"')),
        case("ed_header_missing_id", "schema-invalid: Header without id", BASE,
             variant((HEADER, '<Header version="1" revision="0" />'))),
        case("ed_elements_out_of_order", "schema-invalid: DataTypeTemplates before IED", BASE,
             _swap_types_and_ied()),
        case("ed_unknown_element", "schema-invalid: unknown element in the IED", BASE,
             _private("<Bogus />")),
    ]


def _swap_types_and_ied():
    text = template_text()
    ied = ied_element(text)
    types = text[text.index("<DataTypeTemplates>"):text.index("</DataTypeTemplates>") + 20]
    return text.replace(ied, "\x00", 1).replace(types, ied, 1).replace("\x00", types, 1)
