"""Lessons 9-11: GOOSE from the SLD, point panels with the smart point picker, and commands
(select before operate, typed commands, the tap-changer transformer).

Public data only: the demo station and, for typed commands and the tap changer, the ScadaBench-made
SampleTyped IED (studio/tests/fixtures/typed_controls, "not a manufacturer model").
"""
from __future__ import annotations

import re
from pathlib import Path

from lessons_basics import button, import_station, open_import, run_all, sheet_button, tab

ROOT = Path(__file__).resolve().parents[2]
TYPED = ROOT / "studio" / "tests" / "fixtures" / "typed_controls" / "SampleTyped.scd"


def node(page, label: str):
    return page.locator(".react-flow__node").filter(has=page.get_by_text(label, exact=True)).first


def sheet(page):
    return page.locator("aside").last


def section(page, heading: str):
    return sheet(page).locator("section").filter(has=page.get_by_role("heading", name=heading, exact=True)).first


def popup(page):
    return page.locator("dialog[open], [role=dialog]").last


def build_sld(d, page) -> None:
    """SLD > Build from SCD > Preview > Apply > Tidy up, without narration."""
    d.click(button(page, "SLD"), after=0.4)
    d.click(button(page, "Build from SCD"), after=0.4)
    d.click(button(page, "Preview"), after=0.8)
    d.click(button(page, "Apply to diagram"), after=0.6)
    d.click(button(page, "Tidy up"), after=1.0)


def add_item(d, page, name: str) -> None:
    d.click(button(page, "Add"), after=0.4)
    d.click(page.get_by_role("button", name=re.compile(f"^{re.escape(name)}")).first, after=0.8)


def stop(d, page) -> None:
    d.click(button(page, "Stop"), after=0.4)
    button(page, "Run").first.wait_for(timeout=30000)
    d.wait(0.6)


def goose_sld(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    import_station(d, page, narrate=False)
    build_sld(d, page)
    d.say("GOOSE is how IEDs warn each other in milliseconds. Now you can see it, and send it, on the SLD.")
    d.say("In Add, choose Add all GOOSE IEDs. Every IED that sends or receives GOOSE gets its own box.")
    add_item(d, page, "Add all GOOSE IEDs")
    d.wait(1.0)
    page.locator(".react-flow__pane").click(position={"x": 60, "y": 700})  # clear the new selection
    d.say("Each box is one whole IED. It shows its state and how many messages it sends and gets.")
    d.point(node(page, "FEED1"), pause=0.8)
    d.say("Press Run, and switch on the GOOSE messages layer.")
    run_all(d, page, 6)
    layer = page.get_by_role("switch").filter(has_text="GOOSE messages").first
    if layer.get_attribute("aria-checked") != "true":
        d.click(layer, after=0.5)
    d.say("Click FEED1. Its sheet lists the GOOSE message it sends, value by value, and who receives it.")
    d.click(node(page, "FEED1"), after=1.0)
    d.point(sheet(page).get_by_text(re.compile(r"^To BUS1")).first, pause=0.6)
    d.say("Change a value to send it. Here, the breaker position goes to Closed.")
    member = sheet(page).locator(".goose-send-actions").first
    d.click(member.get_by_role("button", name="Closed", exact=True), after=2.0)
    d.say("A line runs from FEED1 to both busbar IEDs, and got it means the new message really arrived. Send Open again.")
    d.click(member.get_by_role("button", name="Open", exact=True), after=0.4)
    page.get_by_text(re.compile(r"IEDs got it")).first.wait_for(timeout=5000)
    d.wait(1.6)
    d.say("You can also send GOOSE from a breaker, or from a point panel row.")
    stop(d, page)
    d.click(node(page, "CBR_F2"), after=0.8)
    goose = section(page, "GOOSE")
    d.click(goose.get_by_role("button", name="Send GOOSE from this equipment"), after=0.6)
    d.say("Pick the IED and the message, and its sheet gets Send buttons while the IEDs run.")
    selects = goose.locator("select")
    d.point(selects.nth(0))
    selects.nth(0).select_option(label="FEED2")
    d.wait(0.6)
    d.point(selects.nth(1))
    selects.nth(1).select_option(index=1)
    d.wait(0.5)
    d.point(selects.nth(3))
    selects.nth(3).select_option(index=1)
    d.wait(0.6)
    d.say("Press Run. Now the line starts at the breaker, goes through FEED2, and on to the busbar IEDs.")
    run_all(d, page, 6)
    d.click(node(page, "CBR_F2"), after=0.6)
    d.click(section(page, "GOOSE").get_by_role("button", name=re.compile("^Send ")).last, after=2.4)
    d.say("Real devices are only watched. ScadaBench never sends GOOSE for a real IED.")


def point_panels(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    import_station(d, page, narrate=False)
    build_sld(d, page)
    d.say("A point panel holds the points of one IED on the SLD: status, positions, values and commands.")
    add_item(d, page, "Point panel")
    d.say("Choose its IED, then Add points.")
    ied = sheet(page).locator("select").first
    d.point(ied)
    ied.select_option(label="FEED1")
    d.wait(0.5)
    d.click(sheet(page).get_by_role("button", name=re.compile("^Add points")), after=1.0)
    d.click(popup(page).get_by_role("button", name=re.compile("^Add points")), after=0.8)
    d.say("Pick a type first. Status is on and off, Double point is a position, then values and commands.")
    d.click(tab(page, "Double point"), after=0.6)
    d.say("The object list narrows it down. XCBR1 is the breaker.")
    objects = popup(page).locator("select").first
    d.point(objects)
    labels = objects.locator("option").all_inner_texts()
    objects.select_option(index=next(i for i, text in enumerate(labels) if text.startswith("XCBR1")))
    d.wait(0.6)
    d.click(popup(page).locator("input[type=checkbox]").first, after=0.3)
    d.say("Add a trip signal from Status, and the breaker command from Command.")
    d.click(tab(page, "Status"), after=0.5)
    objects.select_option(index=0)
    d.type(popup(page).get_by_placeholder(re.compile("^Search")), "PTRC1.Tr")
    d.click(popup(page).locator("input[type=checkbox]").first, after=0.3)
    d.click(tab(page, "Command"), after=0.6)
    d.type(popup(page).get_by_placeholder(re.compile("^Search")), "CSWI1")
    d.click(popup(page).locator("input[type=checkbox]").first, after=0.3)
    d.click(popup(page).get_by_role("button", name=re.compile(r"^Add \d+ point")), after=0.6)
    d.click(popup(page).get_by_role("button", name="Done", exact=True), after=0.8)
    d.say("Press Run. Now each row changes the live IED.")
    page.keyboard.press("Escape")
    run_all(d, page, 6)
    d.click(node(page, "Points-1"), after=1.0)
    if not popup(page).get_by_role("button", name="ON", exact=True).count():
        d.click(sheet(page).get_by_role("button", name="Open points"), after=1.0)
    d.say("Raise the trip. SCADA and every client see it at once.")
    d.click(popup(page).get_by_role("button", name="ON", exact=True), after=1.0)
    d.say("The GOOSE tag means this position is also sent in a GOOSE message when it changes.")
    d.point(popup(page).get_by_text("GOOSE", exact=True).first, pause=0.8)
    d.say("Commands ask once, then operate like SCADA would.")
    d.click(popup(page).get_by_role("button", name="Close", exact=True).last, after=0.6)
    d.click(page.get_by_role("button", name="Yes, send it"), after=1.4)
    d.say("Give points a healthy value while stopped, and All healthy puts them all back in one go.")
    d.point(popup(page).get_by_role("button", name="All healthy"), pause=0.8)
    page.keyboard.press("Escape")
    stop(d, page)
    d.say("The same smart picker links any SLD symbol to its point. Open a breaker.")
    d.click(node(page, "CBR_F2"), after=0.8)
    position = section(page, "Position comes from")
    d.click(position.get_by_role("button", name="Change"), after=1.0)
    d.say("The picker already knows a breaker needs a double point, and says why.")
    d.point(position.get_by_text(re.compile(r"^Position: showing")).first, pause=1.0)
    d.say("Type chips show how many points of each type there are. All types shows everything.")
    d.point(position.get_by_text("All types").first, pause=0.8)
    d.say("Pick the object, then press Use this. Quality and time stay hidden until you need them.")
    objects = position.locator("select").filter(has_text="All objects").first
    d.point(objects)
    labels = objects.locator("option").all_inner_texts()
    objects.select_option(index=next(i for i, text in enumerate(labels) if text.startswith("XCBR1")))
    d.wait(0.6)
    d.point(position.get_by_text(re.compile("^Chosen")).first, pause=1.0)


def commands(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    import_station(d, page, narrate=False)
    open_typed(d, page)
    build_sld(d, page)
    d.say("Real breakers often ask for Select before Operate: first select, then operate within a time limit.")
    d.say("ScadaBench reads the control model from the SCD. Normally one press selects and operates.")
    run_all(d, page, 7)
    d.click(node(page, "CBR_F1"), after=0.8)
    d.say("For testing, open Command options and show Select before Operate step by step.")
    d.click(sheet(page).get_by_text("Command options").first, after=0.5)
    d.click(sheet(page).get_by_label("Show Select before Operate step by step"), after=0.5)
    d.say("Here you can also mark a test command, or ask for an interlock or synchro check.")
    d.point(sheet(page).get_by_label("Interlock check"), pause=0.6)
    d.say("Press Close. The device is selected, and the countdown shows how long you have.")
    d.click(sheet_button(page, "CBR_F1", "Operate", "Close"), after=1.0)
    d.point(page.get_by_text(re.compile(r"Operate within \d+ s")).first, pause=0.8)
    d.say("Press Operate to send it, or Cancel to give the selection back.")
    d.click(sheet(page).get_by_role("button", name=re.compile(r"^Operate ")), after=1.4)
    stop(d, page)
    d.say("Typed commands go beyond open and close: numbers, choices from a list, raise and lower.")
    d.say("Add a tap-changer transformer. It needs its tap position and a raise and lower command.")
    add_item(d, page, "Tap-changer transformer")
    tap_setup(d, page)
    d.say("Tick the simulated voltage regulator. ScadaBench then works out the low-voltage side for you.")
    regulator = section(page, "Simulated voltage regulator (90)")
    d.click(regulator.get_by_label("Simulate the voltage regulator"), after=0.6)
    d.say("Press Run. Raise and Lower send real step commands to the IED.")
    run_all(d, page, 7)
    card = node(page, "T-1")
    d.click(card.get_by_role("button", name=re.compile("Raise")), after=1.2)
    d.click(card.get_by_role("button", name=re.compile("Raise")), after=1.2)
    d.say("The tap moves, and the voltage follows. Switch to Auto, and the regulator steps by itself to stay in its band.")
    d.point(card, pause=0.6)
    d.click(sheet(page).get_by_role("button", name="Auto", exact=True), after=1.0)
    d.point(card, pause=0.8)
    d.say("Point panels and IED Explorer send the other typed commands, in all four control models.")


def open_typed(d, page) -> None:
    """Add the SampleTyped IED (ScadaBench's typed-command sample) to the demo station."""
    with page.expect_file_chooser() as chooser:
        d.click(page.get_by_role("button", name=re.compile(r"^Import")), after=0.1)
    chooser.value.set_files(str(TYPED))
    page.get_by_text("Choose IEDs to add").wait_for()
    box = page.locator("dialog[open] input[type=checkbox]").first
    d.point(box)
    box.check()  # a plain mouse click on this single row does not tick it in headless Edge
    d.wait(0.2)
    d.click(page.get_by_role("button", name=re.compile(r"^Add \d")), after=0.3)
    page.get_by_text("Choose IEDs to add").wait_for(state="hidden", timeout=90000)


def tap_setup(d, page) -> None:
    """Tap position and step command from SampleTyped's ATCC1 (BSC, direct control)."""
    where = section(page, "Tap position comes from")
    d.click(where.get_by_role("button", name="Choose").first, after=0.6)
    where.locator("select").first.select_option(label="SampleTyped")
    d.wait(0.8)
    row = where.locator("div").filter(has_text="CTRL/ATCC1.TapChg.valWTr.posVal").last
    d.click(row.get_by_role("button", name="Choose"), after=0.6)
    step = section(page, "Raise and lower go to")
    pick = step.locator("select")
    d.point(pick.nth(0))
    pick.nth(0).select_option(label="One step control (BSC or ISC)")
    d.wait(0.4)
    pick.nth(1).select_option(label="SampleTyped — 8 fitting")
    d.wait(0.4)
    d.point(pick.nth(2))
    pick.nth(2).select_option(label="CTRL/ATCC1.TapChg · BSC")
    d.wait(0.6)
