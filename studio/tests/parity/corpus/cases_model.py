"""Names and identity, type resolution and enumerations."""
import re

from parity.corpus.base import Case, edit, template, utf8


def _name(text, value):
    return edit(text, '<IED name="Corpus"', f"<IED name={value}")


def _second_ied(text):
    ied = re.search(r"<IED .*?</IED>", text, re.S).group(0)
    return edit(text, "</IED>", "</IED>" + ied)


def _no_templates(text):
    return re.sub(r"<DataTypeTemplates>.*</DataTypeTemplates>", "", text, flags=re.S)


def _enum(text, body):
    return edit(text, '<EnumType id="StudioSboClass"><EnumVal ord="0">operate-once</EnumVal>'
                      '<EnumVal ord="1">operate-many</EnumVal></EnumType>', f'<EnumType id="StudioSboClass">{body}</EnumType>')


def _sbo_val(text, value):
    return edit(text, '<DA name="sboClass" bType="Enum" fc="CF" type="StudioSboClass"><Val>0</Val></DA>',
                f'<DA name="sboClass" bType="Enum" fc="CF" type="StudioSboClass"><Val>{value}</Val></DA>')


def _doi(text, body):
    return edit(text, '<LN lnClass="GGIO" inst="1" lnType="StudioAnalog" />',
                f'<LN lnClass="GGIO" inst="1" lnType="StudioAnalog">{body}</LN>')


def cases():
    text = template()
    stval = '<DA name="stVal" bType="INT32" fc="ST" dchg="true">'
    items = [
        ("name_space", "IED name with a space", _name(text, '"Bay 1"')),
        ("name_leading_digit", "IED name with a leading digit", _name(text, '"1Bay"')),
        ("name_non_ascii", "non-ASCII IED name (repr)", _name(text, '"Ä1"')),
        ("name_quote", "IED name with an apostrophe (repr picks double quotes)", _name(text, "\"it's\"")),
        ("name_missing", "IED without a name", edit(text, '<IED name="Corpus" ', "<IED ")),
        ("name_65_chars", "65-character IED name", _name(text, '"' + "N" * 65 + '"')),
        ("name_duplicate", "two IEDs with the same name", _second_ied(text)),
        ("ld_inst_empty", "LDevice inst empty", edit(text, '<LDevice inst="CTRL">', '<LDevice inst="">')),
        ("ld_inst_long", "LDevice inst 64 characters", edit(text, '<LDevice inst="CTRL">', '<LDevice inst="' + "L" * 64 + '">')),
        ("ln_prefix_digits", "LN prefix with digits", edit(text, '<LN lnClass="XSWI" inst="2"', '<LN prefix="1A" lnClass="XSWI" inst="2"')),
        ("ln_class_unknown", "unknown lnClass", edit(text, '<LN lnClass="LPHD" inst="1" lnType="StudioLPHD" />',
                                                     '<LN lnClass="ZZZZ" inst="1" lnType="StudioLPHD" />')),
        ("ln_duplicate", "duplicate LN", edit(text, '<LN lnClass="LPHD" inst="1" lnType="StudioLPHD" />',
                                              '<LN lnClass="LPHD" inst="1" lnType="StudioLPHD" />' * 2)),
        ("templates_missing", "no DataTypeTemplates", _no_templates(text)),
        ("lntype_unresolved", "LN refers to a missing LNodeType", edit(text, 'lnType="StudioAnalog" />', 'lnType="Missing" />')),
        ("dotype_unresolved", "DO refers to a missing DOType", edit(text, '<DO name="AnIn2" type="StudioMV" />', '<DO name="AnIn2" type="Nope" />')),
        ("datype_cycle", "DAType referring to itself", edit(text, '<DAType id="StudioMagnitude"><BDA name="f" bType="FLOAT32"><Val>0</Val></BDA>',
                                                            '<DAType id="StudioMagnitude"><BDA name="f" bType="FLOAT32"><Val>0</Val></BDA>'
                                                            '<BDA name="loop" bType="Struct" type="StudioMagnitude" />')),
        ("btype_unknown", "unknown basic type", edit(text, stval, '<DA name="stVal" bType="FLOAT99" fc="ST" dchg="true">')),
        ("fc_unknown", "unknown functional constraint", edit(text, stval, '<DA name="stVal" bType="INT32" fc="ZZ" dchg="true">')),
        ("da_count", "array attribute (count on DA)", edit(text, stval, '<DA name="stVal" bType="INT32" fc="ST" dchg="true" count="3">')),
        ("bda_count", "array attribute (count on BDA)", edit(text, '<BDA name="ctlNum" bType="INT8U" />', '<BDA name="ctlNum" bType="INT8U" count="2" />')),
        ("doi_ix", "DOI with ix", _doi(text, '<DOI name="AnIn1" ix="1"><DAI name="q"><Val>0</Val></DAI></DOI>')),
        ("dai_ix", "DAI with ix", _doi(text, '<DOI name="AnIn1"><DAI name="q" ix="2"><Val>0</Val></DAI></DOI>')),
        ("sdi_value", "SDI/DAI initial value", _doi(text, '<DOI name="AnIn1"><SDI name="mag"><DAI name="f"><Val>12.5</Val></DAI></SDI></DOI>')),
        ("sgroup_values", "setting-group values (several Val)",
         _doi(text, '<DOI name="AnIn1"><SDI name="mag"><DAI name="f"><Val sGroup="1">1</Val><Val sGroup="2">2</Val></DAI></SDI></DOI>')),
        ("val_kind_import", "valKind and valImport on DAI",
         _doi(text, '<DOI name="AnIn1"><DAI name="q" valKind="Set" valImport="true"><Val>0</Val></DAI></DOI>')),
        ("enum_negative_ord", "enum with a negative ord", _enum(text, '<EnumVal ord="-1">minus</EnumVal><EnumVal ord="0">operate-once</EnumVal>')),
        ("enum_gaps", "enum with gaps", _enum(text, '<EnumVal ord="0">operate-once</EnumVal><EnumVal ord="7">seven</EnumVal>')),
        ("enum_duplicate_ord", "duplicate ord", _enum(text, '<EnumVal ord="0">a</EnumVal><EnumVal ord="0">b</EnumVal>')),
        ("enum_empty_literal", "empty enum literal", _enum(text, '<EnumVal ord="0"></EnumVal><EnumVal ord="1">operate-many</EnumVal>')),
        ("enum_non_ascii", "non-ASCII enum literal", _enum(text, '<EnumVal ord="0">été</EnumVal><EnumVal ord="1">operate-many</EnumVal>')),
        ("enum_missing", "enum referenced but missing", _enum(text, "").replace('<EnumType id="StudioSboClass"></EnumType>', "")),
        ("enum_val_unknown", "Val text not in the enum", _sbo_val(text, "nonsense")),
        ("enum_val_literal", "Val given as literal", _sbo_val(text, "operate-many")),
    ]
    for case, purpose, value in items:
        yield Case(case, purpose, "template", utf8(value))
