"""Corpus cases built from the public generators: make_template, goose_samples, sld_fixtures."""
import goose_samples as g
import sld_fixtures as s
from corpus_base import case, template_text

TYPES = ('<DataTypeTemplates><LNodeType id="L" lnClass="LLN0"><DO name="Beh" type="S" /></LNodeType>'
         '<LNodeType id="G" lnClass="GGIO"><DO name="Ind1" type="S" /><DO name="Ind2" type="S" /></LNodeType>'
         '<DOType id="S" cdc="SPS"><DA name="stVal" bType="BOOLEAN" fc="ST" dchg="true" />'
         '<DA name="q" bType="Quality" fc="ST" qchg="true" /><DA name="t" bType="Timestamp" fc="ST" /></DOType>'
         '</DataTypeTemplates>')


def _typed(document):
    """goose_samples documents have no DataTypeTemplates; add the two LN types they use."""
    return document.replace("</SCL>", TYPES + "</SCL>")


def _goose_station():
    publisher = g.publisher("PUB1", [("GC1", "DS1", None), ("GC2", "DS2", "Disabled"), ("GC3", "", "Enabled")])
    subscriber = g.subscriber("SUB1", [g.named_input("PUB1", "GC1"), g.named_input("PUB1", "GC9", "Ind2", "X"),
                                       g.packed_input("PUB1")])
    return g.document([publisher, subscriber], [g.gse("PUB1", "GC1", "01-0C-CD-01-00-01", "0001"),
                                                g.gse("PUB1", "GC2", "01-0C-CD-01-00-02", "0002")])


def _vendor_pair():
    publisher = g.publisher("GE_A", [("Trip", "TripDs", "Enabled")])
    subscriber = g.subscriber("GE_B", [g.packed_input("GE_A", "01-0C-CD-01-00-10"),
                                       g.named_input("GE_A", "Trip", int_addr="IN1")])
    return g.document([publisher, subscriber], [g.gse("GE_A", "Trip", "01-0C-CD-01-00-10", "00AB")])


def sample_cases():
    station = _goose_station()
    feeder_ieds = g.publisher("PUB1", [("GC1", "DS1", None)])
    return [
        case("template_BayController", "make_template('BayController')", "make_template", template_text()),
        case("template_Bay_2", "make_template('Bay_2')", "make_template", template_text("Bay_2")),
        case("template_X1", "make_template('X1')", "make_template", template_text("X1")),
        case("goose_sample_station", "goose_samples: publisher with three blocks, subscriber, no types",
             "goose_samples", station),
        case("goose_sample_station_typed", "goose_samples station with DataTypeTemplates added", "goose_samples",
             _typed(station)),
        case("goose_sample_vendor_typed", "goose_samples GE vendor pair (packed intAddr) with types", "goose_samples",
             _typed(_vendor_pair())),
        case("sld_feeder", "sld_fixtures.feeder(): Substation only, no IED", "sld_fixtures", s.feeder()),
        case("sld_feeder_esld", "sld_fixtures.feeder(esld=True) with esld:x/y", "sld_fixtures", s.feeder(esld=True)),
        case("sld_feeder_with_ied", "sld_fixtures.scl with a goose_samples IED and types", "sld_fixtures",
             _typed(s.scl([("BB", "", ["L"])], ieds=feeder_ieds))),
    ]
