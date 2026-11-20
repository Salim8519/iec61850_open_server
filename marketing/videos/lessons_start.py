"""Lesson 1, the opening video: a quick tour that shows ScadaBench working, not a menu list.

Owner feedback (2026-11-20): show the features happening (GOOSE moving, IED Explorer reading an
IED, the gateway serving SCADA) and keep voice and picture together. Each scene acts first, or
speaks while the action it describes runs, then lets the line finish before moving on.
Public demo station only (OpenSCD example), never customer files.
"""
from __future__ import annotations

import re

from lessons_basics import button, import_station, run_all
from lessons_sld import add_item, build_sld, node, sheet
from lessons_tools import open_tool


def scene(d, text: str, points=(), pause: float = 0.9) -> None:
    """Say one line over the current screen, pointing at its parts while it plays."""
    d.say(text)
    for locator in points:
        d.point(locator, pause=pause)
    d.settle()


def station(d, page) -> None:
    d.say("Start with the SCD file you already have.")
    import_station(d, page, narrate=False)
    scene(d, "ScadaBench turns it into a working substation on one PC. Every IED comes in with its full data model.",
          [page.locator("aside .list-row").filter(has_text="FEED1").first])
    d.say("The single-line diagram draws itself from the same file, with a box for every IED.")  # spoken while it builds
    build_sld(d, page)
    add_item(d, page, "Add all GOOSE IEDs")
    page.locator(".react-flow__pane").click(position={"x": 60, "y": 700})  # clear the new selection
    d.settle()
    run_all(d, page, 6)
    layer = page.get_by_role("switch").filter(has_text="GOOSE messages").first
    if layer.get_attribute("aria-checked") != "true":
        d.click(layer, after=0.6)
    scene(d, "Press Run, and every IED becomes a real IEC 61850 server that your SCADA can connect to.")
    d.click(node(page, "CBR_F1"), after=0.5)
    d.say("Operate a breaker with a real command. The IED answers, and the diagram follows.")
    d.click(sheet(page).get_by_role("button", name="Close", exact=True).first, after=1.5)
    d.settle()
    d.click(node(page, "FEED1"), after=0.8)
    # The breaker was just closed, so Open is a real change and FEED1 publishes a new GOOSE message.
    d.click(sheet(page).locator(".goose-send-actions").first.get_by_role("button", name="Open", exact=True), after=0.3)
    scene(d, "Send a GOOSE message from FEED1, and watch it run to both busbar IEDs.",
          [node(page, "BUS1"), node(page, "BUS2")], pause=1.2)


def tools(d, page) -> None:
    open_tool(d, page, "Find IEDs")
    adapter = page.locator("main select").first
    adapter.select_option(label=page.evaluate("""() => [...document.querySelectorAll('main select option')]
        .map(o => o.textContent).find(t => t.startsWith('This PC')) || ''"""))
    d.click(button(page, "Find IEDs"), after=0.3)
    d.say("Find IEDs scans a network for IEC 61850 devices, and asks each one who it is.")
    page.get_by_role("button", name="Open in Explorer").first.wait_for(timeout=60000)
    d.settle()
    d.click(page.get_by_role("button", name="Open in Explorer").first, after=0.3)
    d.say("IED Explorer is a full IEC 61850 client: browse the data model, watch live values, and send commands.")
    button(page, "Disconnect").first.wait_for(timeout=30000)
    d.click(page.get_by_role("button", name="CSWI1", exact=True).first, after=0.8)
    d.click(page.locator("button.object-toggle").filter(has_text=re.compile(r"^Pos")).first, after=0.8)
    d.settle()
    open_tool(d, page, "SCADA gateway")
    d.click(button(page, "Start gateway"), after=0.3)
    d.say("Be the station gateway: your IEDs go to SCADA over IEC 104, DNP3 or Modbus, with the point list proposed for you.")
    listening = page.get_by_text(re.compile(r"^Listening on")).first
    listening.wait_for(timeout=60000)
    d.point(listening, pause=1.0)
    d.point(page.locator("main tbody tr").first, pause=1.0)
    d.settle()
    scene(d, "Or be the SCADA master, to test a real gateway or RTU.",
          [page.locator("button, [role=tab], [role=radio]").filter(has_text=re.compile(r"^SCADA master$")).first])


def getting_started(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    station(d, page)
    tools(d, page)
    d.click(button(page, "All tools"), after=0.6)
    scene(d, "And more: SNMP switches and clocks, GOOSE traffic checks, generators and timed test sequences.",
          [page.locator(".tool-card[data-tool=snmp]"), page.locator(".tool-card[data-tool=traffic]")])
    d.click(page.get_by_role("navigation", name="Main").get_by_role("button", name=re.compile("^SLD")), after=1.0)
    scene(d, "Factory tests, site tests and training, before the first relay arrives. This is ScadaBench.")
