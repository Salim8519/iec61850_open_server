"""Lessons 4-6: simulation and test values, GOOSE and the lab network, real IEDs and health."""
from __future__ import annotations

import re
import subprocess

from director import start_app
from lessons_basics import ADD, IMPORT, STATION, button, import_station, open_import, run_all, tab


def simulation(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    d.say("Simulation means ScadaBench plays the IED. Your SCADA connects to it like to the real relay.")
    d.click(button(page, "Use a sample IED"), after=0.6)
    d.say("Test points keep the values you change most.")
    d.click(button(page, "Test"))
    d.click(button(page, "Add test point"), after=0.5)
    search = page.get_by_placeholder("Search signals…")
    d.say("Search for the breaker position, and add it.")
    d.type(search, "XCBR1.Pos.stVal")
    d.click(page.locator(".signal-search-row").get_by_role("button", name="Add", exact=True).first, after=0.3)
    d.say("Then a measurement.")
    d.type(search, "AnIn1.mag.f")
    d.click(page.locator(".signal-search-row").get_by_role("button", name="Add", exact=True).first, after=0.3)
    d.click(button(page, "Close"))
    d.say("Press Run. Clients now read these values.")
    run_all(d, page, 1)
    d.say("Close the breaker. Every connected client gets the new position.")
    position = page.locator("select").filter(has_text="Closed (2)").first
    d.point(position)
    position.select_option(label="Closed (2)")
    d.wait(0.8)
    d.say("Set a measurement the same way.")
    value = page.locator("input.signal-input").last
    d.type(value, "132.5")
    d.click(button(page, "Set"), after=0.8)
    d.say("Generators move values for you. Sequences play timed steps or a CSV trace.")
    d.click(tab(page, "Generators"), after=1.0)
    d.click(tab(page, "Sequences"), after=1.2)
    d.say("And every change is listed in Activity, with its time.")
    d.point(page.get_by_role("button", name=re.compile(r"^Activity")), pause=0.8)


def ethernet_label(page) -> str:
    """The text of the wired Ethernet choice in the network card list (labels hold live addresses)."""
    return page.evaluate("""() => [...document.querySelectorAll('main select option')]
        .map(o => o.textContent).find(t => /^Ethernet ·/.test(t)) || ''""")


def goose(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    d.say("GOOSE is the fast message IEDs send each other, for trips and interlocks.")
    import_station(d, page, narrate=False)
    d.say("It is multicast on the Ethernet: one publisher, many subscribers.")
    d.say("Each IED shows its GOOSE partners from the SCD. FEED1 sends to both busbar IEDs.")
    d.point(page.get_by_text("GOOSE").filter(has_text=re.compile("^GOOSE$")).last, pause=0.3)
    d.point(page.locator("td").filter(has_text=re.compile("Sends")).first, pause=0.8)
    d.say("Signal flow draws who sends GOOSE and reports to whom.")
    d.click(page.get_by_role("button", name=re.compile(r"^Signal flow")).first, after=2.0)
    d.say("GOOSE runs on a real cable, so put your IEDs on a network.")
    d.click(button(page, "Settings"), after=0.6)
    d.click(button(page, "Put my IEDs on a network"), after=0.8)
    d.say("Pick the network card that goes to your test switch.")
    card = page.locator("main select").first
    d.point(card)
    card.select_option(label=ethernet_label(page))
    d.say("Each IED keeps its address from the SCD, and ScadaBench checks it is free.")
    page.get_by_text(re.compile(r"^Put \d+ IEDs on")).first.wait_for(timeout=60000)
    d.point(page.get_by_text("from the SCD").first, pause=0.8)
    d.say("You see every Windows change before it happens. Clean up removes only what ScadaBench added.")
    d.click(page.get_by_text("Show the exact Windows changes").first, after=1.2)
    d.settle()
    d.click(button(page, "Cancel"), after=0.4)  # leave without changing Windows
    d.say("Tools shows the GOOSE that is really on the wire, block by block.")
    d.click(button(page, "Tools"), after=0.6)
    d.point(page.get_by_text("GOOSE & Sampled Values traffic", exact=True).first, pause=0.8)
    d.say("Press Run, and your IEDs publish their GOOSE, just like the SCD says. The SLD shows each message as a moving dot.")


def _start_station_copy(browser, index: int) -> tuple[subprocess.Popen, str]:
    """A second ScadaBench simulates one IED of the station: the demo's 'real relay'."""
    process, url, _ = start_app()
    context = browser.new_context()
    page = context.new_page()
    page.goto(url)
    button(page, "Skip tour").first.click()
    button(page, "Devices").first.click()
    with page.expect_file_chooser() as chooser:
        page.get_by_role("button", name=IMPORT).first.click()
    chooser.value.set_files(str(STATION))
    page.locator("dialog.import-dialog input[type=checkbox]").nth(index).check()
    page.get_by_role("button", name=ADD).first.click()
    button(page, "Run").first.click()
    page.get_by_text("Running · 1 IED").first.wait_for(timeout=90000)
    address = page.get_by_text(re.compile(r"^127\.\d+\.\d+\.\d+:102$")).first.inner_text()
    context.close()
    return process, address.split(":")[0]


def real_devices(d, page, data) -> None:
    station, address = _start_station_copy(page.context.browser, 1)
    try:
        d.click(button(page, "Skip tour"), after=0.3)
        d.say("Simulate is for when the relay is not there. Read real IED is for when it is.")
        d.say("Import the SCD and pick the relay to check.")
        open_import(d, page)
        d.click(page.locator("dialog.import-dialog input[type=checkbox]").nth(1), after=0.2)
        d.click(page.get_by_role("button", name=ADD), after=0.3)
        page.get_by_text("Choose IEDs to add").wait_for(state="hidden", timeout=90000)
        d.wait(0.6)
        d.say("Choose Read real IED. ScadaBench connects as a client, and only reads.")
        d.click(tab(page, "Read real IED").last, after=0.6)
        d.say("Enter its address. Here, a second ScadaBench plays the real relay.")
        field = page.get_by_placeholder(re.compile(r"^(e\.g\. |\d+\.\d+)"))
        d.type(field, address)
        d.say("Reports are used only when a block is free, and given back on Stop.")
        d.point(page.get_by_text("Reports (recommended)").first, pause=0.6)
        d.click(button(page, "Save connection"), after=0.6)
        d.say("Press Run. Live means values are arriving from the device.")
        d.click(button(page, "Run"), after=0.5)
        arriving = page.get_by_text("Values are arriving from the device.")
        arriving.first.wait_for(timeout=120000)
        d.point(arriving, pause=1.0)
        d.settle()
        d.say("Compare with SCD reads the device model and lists every difference.")
        d.click(tab(page, "Compare with SCD"), after=3.0)
        d.say("Health check gives one verdict per IED.")
        d.click(page.get_by_role("button", name=re.compile(r"^Health check")).first, after=3.0)
        d.say("Export it as CSV or HTML for your commissioning report.")
    finally:
        station.kill()
        station.wait()
