"""Lessons 1-3: IEC 61850 and ScadaBench, CID and SCD files, the SLD.

The narration teaches the idea while the real app shows it. Every d.say() line is
spoken (voice files from voices.py) and captioned; the recorder starts the next line
only when the previous one has been spoken.
"""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
# Public demo station (OpenSCD example, generic names); built by demo_station.py. Never use customer files.
STATION = Path(__file__).resolve().parent / "demo" / "scadabench_demo.scd"
IMPORT = re.compile(r"^Import CID . SCD")
ADD = re.compile(r"^Add \d")


def button(page, name: str):
    return page.get_by_role("button", name=name, exact=True)


def tab(page, name: str):
    """Segmented tabs (Test, Settings) are radio-style controls, found by their text."""
    return page.locator("button, [role=tab], [role=radio]").filter(has_text=re.compile(f"^{re.escape(name)}$"))


def open_import(d, page) -> None:
    with page.expect_file_chooser() as chooser:
        d.click(page.get_by_role("button", name=IMPORT), after=0.1)
    chooser.value.set_files(str(STATION))
    page.get_by_text("Choose IEDs to add").wait_for()


def import_station(d, page, count: int = 6, narrate: bool = True) -> None:
    """Devices > Import CID / SCD with the public demo station."""
    d.click(button(page, "Devices"))
    if narrate:
        d.say("In Devices, press Import and choose your SCD file.")
    open_import(d, page)
    if narrate:
        d.say("ScadaBench lists every IED in the file. Nothing is added until you confirm.")
        d.settle()
        d.say("Tick the IEDs you want, and add them.")
    for box in page.locator("dialog.import-dialog input[type=checkbox]").all()[:count]:
        d.click(box, after=0.08 if narrate else 0.02)
    d.click(page.get_by_role("button", name=ADD), after=0.3)
    page.get_by_text("Choose IEDs to add").wait_for(state="hidden", timeout=90000)
    d.wait(0.8)


def run_all(d, page, count: int) -> None:
    d.click(button(page, "Run"), after=0.3)
    label = "Running · 1 IED" if count == 1 else f"Running · {count} IEDs"
    page.get_by_text(label).first.wait_for(timeout=90000)


def getting_started(d, page, data) -> None:
    d.say("IEC 61850 is the standard substation devices use to talk. ScadaBench lets you test it on your PC.")
    for text in ["Devices are your IEDs: the relays and bay controllers of a substation.",
                 "SLD is the single-line diagram: the station, drawn with one line per circuit.",
                 "Run turns each IED into a live IEC 61850 server.",
                 "Test lets you change values on purpose.",
                 "Tools has helpers: an IED client, a network scan, a SCADA gateway and GOOSE traffic.",
                 "Settings puts the IEDs on a real network.",
                 "And Help brings this tour, and these videos, back any time."]:
        d.settle()
        d.click(button(page, "Next"), after=0.2)
        d.say(text)
    d.settle()
    d.click(button(page, "Start using ScadaBench"))
    d.quiet()
    d.click(button(page, "Devices"))
    d.say("Let's try one. Use a sample IED.")
    d.click(button(page, "Use a sample IED"), after=0.8)
    d.click(tab(page, "Data points"), after=0.6)
    d.say("Every IED has a data model. Feeder01CTRL is its logical device.")
    d.type(page.get_by_placeholder(re.compile("^Search name")), "XCBR1.Pos")
    d.say("XCBR1 is the circuit breaker. Pos is its position, and stVal holds the value.")
    d.point(page.get_by_text("Feeder01CTRL/XCBR1.Pos.stVal").first, pause=1.0)
    d.say("ST means status. MX means measurement. The badge before it tells the point type.")
    d.point(page.get_by_text("ST", exact=True).first, pause=0.8)
    d.say("Press Run. The IED now answers over MMS, on port 102, like a real relay.")
    run_all(d, page, 1)
    d.click(tab(page, "Overview"), after=0.4)
    d.say("Point your SCADA or test client at this address.")
    d.point(page.get_by_text(re.compile(r"^127\.50\.0\.\d+:102$")).first, pause=0.8)
    d.click(button(page, "Copy address"), after=0.8)
    d.say("A real IEC 61850 server, on your PC, without a single relay.")
    d.settle()
    d.click(button(page, "Settings"), after=0.6)
    d.click(tab(page, "Display"), after=0.6)
    d.say("On a small laptop, Fit to window draws the same layout, just evenly smaller.")
    d.point(page.get_by_text("Fit to window (recommended)").first, pause=1.0)


def cid_scd(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    d.say("IEC 61850 projects are described in SCL files.")
    d.say("A CID file describes one configured IED. An SCD describes the whole substation.")
    d.say("The SCD holds every IED, the network addresses, GOOSE, and the single-line layout.")
    import_station(d, page)
    d.say("Each IED keeps its full data model, and its address from the SCD.")
    d.point(page.get_by_text("Address in the SCD").first, pause=1.0)
    d.say("Choose a role for each IED. Simulate means ScadaBench plays it.")
    d.point(tab(page, "Simulate").last, pause=0.6)
    d.say("Monitor real device reads the real relay instead. Skip keeps it in the project, but does not start it.")
    d.point(tab(page, "Monitor real device").last, pause=0.5)
    d.point(tab(page, "Skip").last, pause=0.5)
    d.say("The list shows who sends GOOSE to whom, straight from the SCD.")
    d.point(page.get_by_text(re.compile(r"sends to \d")).first, pause=0.8)
    d.say("File check details lists anything unusual. Nothing is silently dropped.")
    d.click(page.get_by_text(re.compile(r"^File check details")).first, after=1.4)
    d.say("A CID file works the same way, for a single IED.")


def sheet_button(page, label: str, section: str, name: str):
    """A button inside one section (Operate, Test) of an SLD symbol's side panel."""
    sheet = page.get_by_role("complementary", name=f"{label} properties")
    part = sheet.locator("section").filter(has=page.get_by_role("heading", name=section, exact=True))
    return part.get_by_role("button", name=name, exact=True)


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
    d.say("Each symbol is linked to a signal of its IED. Press Run, and every switch shows its live position.")
    run_all(d, page, 6)
    d.wait(1.0)
    d.say("Click a breaker to open its panel.")
    d.click(page.locator(".react-flow__node").filter(has_text=re.compile(r"^CBR_F1")).first, after=0.8)
    d.say("Operate sends a real IEC 61850 command, exactly like SCADA. Close it.")
    d.click(sheet_button(page, "CBR_F1", "Operate", "Close"), after=1.6)
    d.say("This breaker asks for Select before Operate. ScadaBench selects and operates in one press, like a station HMI.")
    d.point(page.get_by_text(re.compile("^This device asks for Select before Operate")).first, pause=0.6)
    d.say("The IED switched, and the diagram follows.")
    d.wait(0.6)
    d.say("Test buttons move the breaker by itself, as if it tripped. No command is sent.")
    d.click(sheet_button(page, "CBR_F1", "Test", "Open"), after=1.2)
    d.say("Your station, drawn and alive in under a minute.")
