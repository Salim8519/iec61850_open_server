"""Corpus cases about identifiers, type resolution and enumerations."""
from corpus_base import case, sub, template_text, variant, with_second_ied, with_types

BASE = "make_template(BayController)"
IED_OPEN = '<IED name="BayController"'
XCBR = '<LN lnClass="XCBR" inst="1" lnType="StudioCB">'
XSWI2 = '<LN lnClass="XSWI" inst="2" lnType="StudioSwitch">'
STATUS = '<DOType id="StudioStatus" cdc="INS">'
MAG = '<BDA name="f" bType="FLOAT32"><Val>0</Val></BDA>'
CTL = '<DA name="ctlModel" bType="Enum" fc="CF" type="StudioControlModel"><Val>3</Val></DA>'
ENUM = '<EnumType id="StudioControlModel">'
POS_DOI = '<DOI name="Pos"><DAI name="stVal"><Val>1</Val></DAI></DOI>'


def _ied_name(value):
    return variant((IED_OPEN, f'<IED name="{value}"' if value is not None else "<IED"))


def _enum(values):
    body = "".join(f'<EnumVal ord="{ordinal}">{label}</EnumVal>' for ordinal, label in values)
    return variant((CTL, '<DA name="ctlModel" bType="Enum" fc="CF" type="CaseEnum"><Val>3</Val></DA>'),
                   ("</DataTypeTemplates>", f'<EnumType id="CaseEnum">{body}</EnumType></DataTypeTemplates>'))


def name_cases():
    long_name = "B" + "a" * 64
    return [
        case("name_space", "IED name with a space", BASE, _ied_name("Bay Controller")),
        case("name_leading_digit", "IED name starting with a digit", BASE, _ied_name("1Bay")),
        case("name_non_ascii", "IED name with A-umlaut (repr keeps it)", BASE, _ied_name(chr(0xC4) + "1")),
        case("name_apostrophe", "IED name with an apostrophe (repr uses double quotes)", BASE, _ied_name("it's")),
        case("name_both_quotes", "IED name with both quote kinds (repr escapes)", BASE,
             _ied_name("a'b&quot;c")),
        case("name_backslash_tab", "IED name with backslash and tab character reference", BASE,
             _ied_name("a\\b&#9;c")),
        case("name_missing", "IED without a name attribute", BASE, _ied_name(None)),
        case("name_empty", "IED name is empty", BASE, _ied_name("")),
        case("name_65_chars", "65-character IED name (identifier allows, XSD may not)", BASE,
             _ied_name(long_name)),
        case("name_underscore_only", "IED name '_'", BASE, _ied_name("_")),
        case("name_duplicate", "two IEDs with the same name", BASE,
             with_second_ied(template_text(), "BayController")),
        case("name_two_ieds", "two IEDs BayController and Bay_2", BASE, with_second_ied(template_text())),
        case("name_ld_inst_empty", "LDevice inst=''", BASE,
             variant(('<LDevice inst="CTRL">', '<LDevice inst="">'))),
        case("name_ld_inst_digit", "LDevice inst starting with a digit", BASE,
             variant(('<LDevice inst="CTRL">', '<LDevice inst="1CTRL">'))),
        case("name_ld_name", "functional ldName on the LDevice", BASE,
             variant(('<LDevice inst="CTRL">', '<LDevice inst="CTRL" ldName="Bay1Func">'))),
        case("name_ld_name_invalid", "invalid functional ldName", BASE,
             variant(('<LDevice inst="CTRL">', '<LDevice inst="CTRL" ldName="Bay-1">'))),
        case("name_ln_prefix_digits", "LN prefix with digits; dataset member no longer resolves", BASE,
             variant((XCBR, '<LN prefix="Q0" lnClass="XCBR" inst="1" lnType="StudioCB">'))),
        case("name_lnclass_unknown", "unknown lnClass ZZZZ", BASE,
             variant((XSWI2, '<LN lnClass="ZZZZ" inst="2" lnType="StudioSwitch">'))),
        case("name_duplicate_ln", "two LN with the same name: ambiguous data reference", BASE,
             variant((XSWI2, '<LN lnClass="XSWI" inst="1" lnType="StudioSwitch">'))),
        case("name_dataset_invalid", "DataSet name with a dot", BASE,
             variant(('<DataSet name="Events">', '<DataSet name="Ev.ents">'))),
    ]


def type_cases():
    text = template_text()
    types_start = text.index("<DataTypeTemplates>")
    no_types = text[:types_start] + text[text.index("</DataTypeTemplates>") + 20:]
    return [
        case("type_missing_templates", "no DataTypeTemplates section", BASE, no_types),
        case("type_empty_templates", "empty DataTypeTemplates: unresolved LNodeType", BASE,
             no_types.replace("</SCL>", "<DataTypeTemplates /></SCL>")),
        case("type_unresolved_lntype", "LN refers to a missing LNodeType", BASE,
             variant((XSWI2, '<LN lnClass="XSWI" inst="2" lnType="Missing">'))),
        case("type_unresolved_dotype", "DO refers to a missing DOType", BASE,
             variant(('<DO name="Loc" type="StudioBoolean" />', '<DO name="Loc" type="MissingDO" />'))),
        case("type_unresolved_datype", "Struct DA refers to a missing DAType", BASE,
             variant(('type="StudioMagnitude"', 'type="MissingDA"'))),
        case("type_unresolved_enum", "Enum DA refers to a missing EnumType", BASE,
             variant(('type="StudioSboClass"', 'type="MissingEnum"'))),
        case("type_recursive_datype", "DAType containing itself", BASE,
             variant((MAG, MAG + '<BDA name="loop" bType="Struct" type="StudioMagnitude" />'))),
        case("type_recursive_dotype", "DOType containing itself through an SDO", BASE,
             variant((STATUS, STATUS + '<SDO name="self" type="StudioStatus" />'))),
        case("type_unknown_btype", "unknown basic type FLOAT16", BASE,
             variant(('<BDA name="f" bType="FLOAT32">', '<BDA name="f" bType="FLOAT16">'))),
        case("type_unknown_fc", "unknown functional constraint XX", BASE,
             variant(('<DA name="t" bType="Timestamp" fc="ST" />', '<DA name="t" bType="Timestamp" fc="XX" />'))),
        case("type_missing_fc", "DA without fc", BASE,
             variant(('<DA name="t" bType="Timestamp" fc="ST" />', '<DA name="t" bType="Timestamp" />'))),
        case("type_count_da", "DA with count=4 (array attribute)", BASE,
             variant(('<DA name="t" bType="Timestamp" fc="ST" />',
                      '<DA name="t" bType="Timestamp" fc="ST" count="4" />'))),
        case("type_count_bda", "BDA with count=2 (array attribute)", BASE,
             variant(('<BDA name="orIdent" bType="Octet64" />', '<BDA name="orIdent" bType="Octet64" count="2" />'))),
        case("type_count_not_integer", "DA count is not an integer", BASE,
             variant(('<DA name="t" bType="Timestamp" fc="ST" />',
                      '<DA name="t" bType="Timestamp" fc="ST" count="x" />'))),
        case("type_sdo_array", "array of SDO (count=2)", BASE,
             with_types(variant((STATUS, STATUS + '<SDO name="sub" type="CaseSub" count="2" />')),
                        '<DOType id="CaseSub" cdc="INS"><DA name="stVal" bType="INT16" fc="ST" /></DOType>')),
        case("type_do_count_300", "DO count above 256", BASE,
             variant(('<DO name="AnIn2" type="StudioMV" />', '<DO name="AnIn2" type="StudioMV" count="300" />'))),
        case("type_doi_ix", "DOI with ix attribute", BASE,
             variant((POS_DOI, '<DOI name="Pos" ix="1"><DAI name="stVal"><Val>1</Val></DAI></DOI>'))),
        case("type_sdi_dai_ix", "SDI and DAI with ix in an instance", BASE,
             variant(('<LN lnClass="GGIO" inst="1" lnType="StudioAnalog" />',
                      '<LN lnClass="GGIO" inst="1" lnType="StudioAnalog"><DOI name="AnIn1"><SDI name="mag" ix="0">'
                      '<DAI name="f" ix="0"><Val>3.5</Val></DAI></SDI></DOI></LN>'))),
        case("type_sgroup_vals", "DAI with several Val elements and sGroup", BASE,
             variant((POS_DOI, '<DOI name="Pos"><DAI name="stVal"><Val sGroup="1">1</Val><Val sGroup="2">2</Val>'
                               '</DAI></DOI>'))),
        case("type_valkind_valimport", "DAI with valKind and valImport", BASE,
             variant((POS_DOI, '<DOI name="Pos"><DAI name="stVal" valKind="RO" valImport="true"><Val>2</Val></DAI></DOI>'))),
        case("type_duplicate_declaration", "two DOType declarations with one id", BASE,
             with_types(text, '<DOType id="StudioStatus" cdc="INS" />')),
        case("type_doi_desc", "DOI desc overrides the signal description", BASE,
             variant((POS_DOI, '<DOI name="Pos" desc="Breaker position"><DAI name="stVal" desc="DAI text">'
                               '<Val>1</Val></DAI></DOI>'))),
    ]


def enum_cases():
    return [
        case("enum_negative_ord", "EnumVal with negative ord", BASE, _enum([(-1, "minus"), (0, "zero"), (3, "three")])),
        case("enum_gaps", "EnumVal ords with gaps", BASE, _enum([(0, "a"), (5, "b"), (100, "c")])),
        case("enum_duplicate_ord", "two labels with the same ord", BASE, _enum([(1, "one"), (1, "uno"), (3, "x")])),
        case("enum_duplicate_label", "one label with two ords (last wins in dict)", BASE,
             _enum([(1, "same"), (2, "same"), (3, "x")])),
        case("enum_empty_literal", "EnumVal with empty text is skipped", BASE, _enum([(0, ""), (3, "x")])),
        case("enum_whitespace_literal", "EnumVal text with surrounding whitespace", BASE,
             _enum([(0, "  padded  "), (3, "x")])),
        case("enum_non_ascii_literal", "EnumVal text with non-ASCII characters", BASE,
             _enum([(0, "Ein" + chr(0xDF)), (3, "aus" + chr(0x2713))])),
        case("enum_ord_not_integer", "EnumVal ord is not an integer", BASE, _enum([("x", "bad"), (3, "x")])),
        case("enum_val_label", "Val given as a label of the enum", BASE,
             variant((CTL, '<DA name="ctlModel" bType="Enum" fc="CF" type="StudioControlModel">'
                           '<Val>direct-normal</Val></DA>'))),
        case("enum_val_not_in_enum", "Val text that is neither a label nor an integer", BASE,
             variant((CTL, '<DA name="ctlModel" bType="Enum" fc="CF" type="StudioControlModel"><Val>bogus</Val></DA>'))),
        case("enum_val_ord_out_of_range", "Val given as an ord not declared in the enum", BASE,
             variant((CTL, '<DA name="ctlModel" bType="Enum" fc="CF" type="StudioControlModel"><Val>42</Val></DA>'))),
        case("enum_type_empty", "EnumType without EnumVal", BASE,
             variant((ENUM, '<EnumType id="Unused" />' + ENUM))),
    ]
