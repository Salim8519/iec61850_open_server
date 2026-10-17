"""Lessons 4-6: GOOSE and the lab network, real IEDs and health, activation."""
from __future__ import annotations

import json
import re
import subprocess

from director import start_app
from lessons_basics import STATION, button, import_station, open_import, tab


def goose(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    d.say("GOOSE is the fast message IEDs send each other, for trips and interlocks.")
    import_station(d, page, narrate=False)
    d.say("It is multicast on the Ethernet: one publisher, many subscribers.")
    d.click(button(page, "Settings"), after=0.6)
    d.say("Settings lists every GOOSE block in the SCD.")
    d.point(page.get_by_text("Feed1SwiPosCtl").first, pause=0.6)
    d.say("Each has an APPID and a multicast MAC address, so subscribers know what to listen to.")
    d.point(page.get_by_text("APPID").first, pause=0.8)
    d.say("GOOSE needs the free Npcap driver; ScadaBench checks it for you.")
    d.point(page.get_by_text(re.compile("Npcap")).first, pause=0.6)
    d.say("GOOSE runs on a real cable, so set up the lab network.")
    d.click(button(page, "Set up lab network"), after=0.8)
    d.say("Pick the adapter that goes to your test network.")
    d.click(page.get_by_role("button", name=re.compile(r"^Ethernet")).first, after=0.4)
    d.click(button(page, "Next"), after=0.6)
    d.say("Keep the addresses from the SCD, so every device finds its IED.")
    next_button = button(page, "Next")
    for _ in range(80):
        if next_button.is_enabled():
            break
        d.wait(0.5)
    d.settle()
    d.click(next_button, after=0.8)
    d.say("Review every Windows change. Nothing happens until you press Apply.")
    d.point(page.get_by_text(re.compile(r"^Add address .* for FEED1$")).first, pause=0.6)
    d.say("Later, one click removes only what ScadaBench added.")
    d.point(page.get_by_text(re.compile(r"^Allow other devices")).first, pause=0.6)
    d.settle()
    d.click(button(page, "Devices"))  # leave the wizard without applying anything
    d.say("Signal flow shows who sends GOOSE and reports to whom.")
    d.click(page.get_by_role("button", name=re.compile(r"^Signal flow")).first, after=2.0)
    d.say("Press Run, and your IEDs publish their GOOSE, just like the SCD says.")


def _start_station_copy(browser, index: int) -> tuple[subprocess.Popen, str]:
    """A second ScadaBench simulates one IED of the station: the demo's 'real relay'."""
    process, url, _ = start_app()
    context = browser.new_context()
    page = context.new_page()
    page.goto(url)
    button(page, "Skip tour").first.click()
    button(page, "Devices").first.click()
    with page.expect_file_chooser() as chooser:
        page.get_by_role("button", name=re.compile(r"^Import CID . SCD")).first.click()
    chooser.value.set_files(str(STATION))
    page.locator("input[type=checkbox]").nth(index).check()
    page.get_by_role("button", name=re.compile(r"^Add \d")).first.click()
    button(page, "Run").first.click()
    page.get_by_text("Running · 1 IED").wait_for(timeout=90000)
    address = page.get_by_text(re.compile(r"^127\.\d+\.\d+\.\d+:102$")).first.inner_text()
    context.close()
    return process, address.split(":")[0]


def real_devices(d, page, data) -> None:
    station, address = _start_station_copy(page.context.browser, 1)
    try:
        d.click(button(page, "Skip tour"), after=0.3)
        d.say("Simulate is for when the relay is not there. Real device is for when it is.")
        d.say("Import the SCD and pick the relay to check.")
        open_import(d, page)
        d.click(page.locator("input[type=checkbox]").nth(1), after=0.2)
        d.click(page.get_by_role("button", name=re.compile(r"^Add \d")), after=0.3)
        page.get_by_text("Choose IEDs to add").wait_for(state="hidden", timeout=90000)
        d.wait(0.6)
        d.say("Choose Real device. ScadaBench connects as a client, and only reads.")
        d.click(page.get_by_text("Real device", exact=True).first, after=0.6)
        d.say("Enter its address. Here, a second ScadaBench plays the real relay.")
        field = page.get_by_placeholder(re.compile(r"^(e\.g\. |\d+\.\d+)"))
        d.type(field, address)
        d.click(button(page, "Save connection"), after=0.6)
        d.say("Press Run. It uses free report blocks only, and gives them back on Stop.")
        d.click(button(page, "Run"), after=0.5)
        page.get_by_text("Values are arriving from the device").first.wait_for(timeout=120000)
        d.point(page.get_by_text(re.compile(r"will use report block")).first, pause=1.2)
        d.settle()
        d.say("Compare with SCD reads the device model and lists every difference.")
        d.click(button(page, "Compare with SCD"), after=3.0)
        d.say("Health check gives one verdict per IED.")
        d.click(page.get_by_role("button", name=re.compile(r"^Health check")).first, after=3.0)
        d.say("Export it as CSV or HTML for your commissioning report.")
    finally:
        station.kill()
        station.wait()


DEMO_HWID = "4f1c9a2e7b3d5c8f0a6e1b9d2c7f4a3e5b8d0c6f1a9e2b7d3c5f8a0e6b1d9c2f"
UNLICENSED = {"licensed": False, "mode": "unlicensed", "plan": None, "days_remaining": None, "expires_at": None,
              "reason": None, "license_id": None, "email": None, "enforced": True,
              "server_url": "https://license.scadabench.com", "masked_key": None, "hwid": DEMO_HWID,
              "transfer_available": False}
LICENSED = {**UNLICENSED, "licensed": True, "mode": "lifetime", "plan": "lifetime", "license_id": "SUBS-DEMO",
            "email": "you@example.com", "masked_key": "SUBS-••••-••••-••••-R8WC"}


def activation_prepare(page) -> None:
    """The development build has no license lock, so the license screens get demo answers."""
    state = {"status": UNLICENSED}

    def status(route):
        route.fulfill(status=200, content_type="application/json", body=json.dumps(state["status"]))

    def activate(route):
        state["status"] = LICENSED
        status(route)

    page.route("**/api/v1/license/status", status)
    page.route("**/api/v1/license/activate", activate)


def activation(d, page, data) -> None:
    d.say("A new install opens on the activation screen.")
    d.say("Start a free 7-day trial, no card needed. Or buy a lifetime license.")
    d.point(button(page, "Start 7-day free trial"), pause=0.8)
    d.say("Have a key? It is in your purchase email. Paste it here.")
    d.type(page.get_by_placeholder("SUBS-XXXX-XXXX-XXXX-XXXX"), "SUBS-7K4P-QX2M-9TDF-R8WC")
    d.click(button(page, "Activate this computer"), after=1.2)
    page.get_by_text("Engine connected").wait_for(timeout=15000)
    d.say("Done. Your projects open exactly as before.")
    try:  # a new install also opens the first-start tour; close it for this lesson
        button(page, "Skip tour").first.wait_for(timeout=4000)
        d.click(button(page, "Skip tour"), after=0.3)
    except Exception:
        pass
    d.click(button(page, "Settings"))
    d.click(tab(page, "License"), after=1.0)
    d.say("New computer? Enter the key there and choose Move the key here.")
    d.say("No internet on site? Copy the Machine ID, and use the offline page on any other PC.")
    d.settle()
    page.goto("https://license.scadabench.com/activate")
    page.get_by_text("without internet").first.wait_for(timeout=20000)
    d.type(page.locator("#key"), "SUBS-7K4P-QX2M-9TDF-R8WC", after=0.2)
    d.type(page.locator("#hwid"), DEMO_HWID, after=0.4)
    d.say("Download the license file, and load it on the offline PC.")
