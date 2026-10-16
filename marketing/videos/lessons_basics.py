"""Lessons 1-4: IEC 61850 and ScadaBench, CID and SCD files, the SLD, simulation.

The narration teaches the idea while the real app shows it. Every d.say() line is
spoken (voice files from voices.py) and captioned; the recorder starts the next line
only when the previous one has been spoken.
"""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
STATION = ROOT / "test_cids" / "Studio_Test_400kV.scd"
IMPORT = re.compile(r"^Import CID . SCD")


def button(page, name: str):
    return page.get_by_role("button", name=name, exact=True)


def open_import(d, page) -> None:
    with page.expect_file_chooser() as chooser:
        d.click(page.get_by_role("button", name=IMPORT), after=0.1)
    chooser.value.set_files(str(STATION))
    page.get_by_text("Choose IEDs to add").wait_for()


def import_station(d, page, count: int = 8, narrate: bool = True) -> None:
    """Devices > Import CID / SCD with the 400 kV test station."""
    d.click(button(page, "Devices"))
    if narrate:
        d.say("In Devices, press Import and choose your SCD file.")
    open_import(d, page)
    if narrate:
        d.say("ScadaBench lists every IED in the file, with its vendor and type.")
        d.settle()
        d.say("Tick the IEDs you want to simulate.")
    for box in page.locator("input[type=checkbox]").all()[:count]:
        d.click(box, after=0.08 if narrate else 0.02)
    d.click(page.get_by_role("button", name=re.compile(r"^Add \d")), after=1.2)


def getting_started(d, page, data) -> None:
    d.say("IEC 61850 is the standard substation devices use to talk. ScadaBench lets you test it on your PC.")
    for text in ["Devices are your IEDs: the relays and bay controllers of a substation.",
                 "SLD is the single-line diagram: the station, drawn with one line per circuit.",
                 "Run turns each IED into a live IEC 61850 server.",
                 "Test lets you change values on purpose.",
                 "Settings puts the IEDs on a real network, for GOOSE.",
                 "And Help brings this tour back any time."]:
        d.settle()
        d.click(button(page, "Next"), after=0.2)
        d.say(text)
    d.settle()
    d.click(button(page, "Start using ScadaBench"))
    d.quiet()
    d.say("Let's try one. Use a sample IED.")
    d.click(button(page, "Use a sample IED"), after=0.8)
    d.click(button(page, "Data points"), after=0.6)
    d.say("Every IED has a data model. Feeder01CTRL is its logical device.")
    d.type(page.get_by_placeholder("Search data points"), "XCBR1.Pos")
    d.say("XCBR1 is the circuit breaker. Pos is its position, and stVal holds the value.")
    d.point(page.get_by_text("Feeder01CTRL/XCBR1.Pos.stVal").first, pause=1.0)
    d.say("ST means status. MX means measurement.")
    d.point(page.get_by_text("FC", exact=True).first, pause=0.8)
    d.say("Press Run. The IED now answers over MMS, on port 102, like a real relay.")
    d.click(button(page, "Run"), after=0.3)
    page.get_by_text("Running · 1 IED").wait_for(timeout=30000)
    d.click(button(page, "Overview"), after=0.4)
    d.say("Point your SCADA or test client at this address.")
    d.point(page.get_by_text("127.50.0.2:102"), pause=0.8)
    d.click(button(page, "Copy address"), after=0.8)
    d.say("A real IEC 61850 server, on your PC, without a single relay.")


def cid_scd(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    d.say("IEC 61850 projects are described in SCL files.")
    d.say("A CID file describes one configured IED. An SCD describes the whole substation.")
    d.say("The SCD holds every IED, the network addresses, GOOSE, and the single-line layout.")
    import_station(d, page)
    d.say("Each IED keeps its full data model, and its address from the SCD.")
    d.point(page.get_by_text("Address in the SCD").first, pause=1.0)
    d.say("File check details lists anything unusual. Nothing is silently dropped.")
    d.click(page.get_by_text(re.compile(r"^File check details")).first, after=1.4)
    d.say("A CID file works the same way, for a single IED.")


def sld(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    import_station(d, page, narrate=False)
    d.say("An SLD, or single-line diagram, draws a three-phase station with one line per circuit.")
    d.click(button(page, "SLD"))
    d.say("Your SCD already describes it, in its Substation section.")
    d.click(button(page, "Build from SCD"), after=0.6)
    d.say("Preview first. Nothing changes until you apply.")
    d.click(button(page, "Preview"), after=1.4)
    d.say("QA is a circuit breaker, QB a disconnector, QC an earthing switch.")
    d.click(button(page, "Apply to diagram"), after=1.0)
    d.say("Tidy up gives you a clean diagram. Undo puts it back.")
    d.click(button(page, "Tidy up"), after=1.4)
    d.say("Each symbol is linked to a signal of its IED.")
    d.say("Press Run, and every switch shows its live position.")
    d.click(button(page, "Run"), after=0.3)
    page.get_by_text(re.compile(r"Running · \d+ IEDs")).wait_for(timeout=60000)
    d.wait(1.5)
    d.say("Your station, drawn and alive in under a minute.")


def simulation(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    d.say("Simulation means ScadaBench plays the IED. Your SCADA connects to it like to the real relay.")
    d.click(button(page, "Use a sample IED"), after=0.6)
    d.say("Test points keep the values you change most.")
    d.click(button(page, "Test"))
    d.click(button(page, "Add test point"), after=0.5)
    search = page.get_by_placeholder("Search signals")
    d.say("Search for the breaker position, and add it.")
    d.type(search, "Pos.stVal")
    d.click(page.get_by_role("button", name="Add", exact=True).first, after=0.3)
    d.say("Then a measurement.")
    d.type(search, "AnIn1.mag.f")
    d.click(page.get_by_role("button", name="Add", exact=True).first, after=0.3)
    d.click(button(page, "Close"))
    d.say("Press Run. Clients now read these values.")
    d.click(button(page, "Run"), after=0.3)
    page.get_by_text("Running · 1 IED").wait_for(timeout=30000)
    d.say("Close the breaker. Every connected client gets the new position.")
    position = page.locator("select").filter(has_text="Closed (2)").first
    d.point(position)
    position.select_option(label="Closed (2)")
    d.wait(0.8)
    d.say("Set a measurement the same way.")
    value = page.locator("table input, [role=row] input, input[type=text]").last
    d.type(value, "132.5")
    d.click(button(page, "Set"), after=0.8)
    d.say("Generators move values for you. Sequences play timed steps or a CSV trace.")
    d.click(button(page, "Generators"), after=1.0)
    d.click(button(page, "Sequences"), after=1.2)
    d.say("And every change is listed in Activity, with its time.")
