"""Lesson 14: the lab network on this PC (ADR-0120): each IED at its own SCD address, without a VM.

Needs the ScadaBench Lab card on the recording PC (created once with one Windows prompt); the
recording itself never asks Windows for anything. Uses only the public demo station.
"""
from __future__ import annotations

import re

from lessons_basics import button, import_station, run_all

MOVED = "192.168.216.1"


def lab_network(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    d.say("Every IED has its own address in the SCD. Testing at those addresses used to need a virtual machine.")
    import_station(d, page, narrate=False)
    d.say("ScadaBench puts them on a lab network inside this PC instead.")
    d.click(button(page, "Settings"), after=0.8)
    card = page.locator(".lab-card")
    d.point(card.get_by_role("heading", name=re.compile("^Lab network on this PC")), pause=0.4)
    d.say("Windows asks for permission once, to add the ScadaBench Lab card. No download, no restart.")
    use = page.locator("button:not([disabled])").filter(has_text=re.compile("^Use the lab network$"))
    use.first.wait_for(timeout=60000)
    d.say("Press Use the lab network. There are no more Windows prompts after that.")
    d.click(use, after=0.4)
    page.get_by_text(re.compile(r"^In use: 6 IEDs")).first.wait_for(timeout=60000)
    d.wait(0.6)
    d.say("Each IED now answers at its own address from the SCD.")
    summary = card.get_by_text(re.compile(r"^Show the 6 addresses"))
    d.wait(1.5)  # the list opens by itself while an address is being put back; let it settle first
    if card.locator("details[open]").count():
        d.point(summary, pause=0.8)
    else:
        d.click(summary, after=0.8)
    d.say("Change any address at any time. Type it and press Enter. The SCD file is never changed.")
    field = page.get_by_label("Address of FEED2 on this PC")
    d.type(field, MOVED, after=0.2)
    page.keyboard.press("Enter")
    page.locator("tr").filter(has=page.locator(f"input[value='{MOVED}']")).first.wait_for(timeout=60000)
    d.wait(0.6)
    d.say("Addresses already used on this PC, or by a real device on your Ethernet or Wi-Fi, are caught before they clash.")
    d.point(card.locator("td .address-check").nth(1), pause=0.6)
    d.say("Press Run.")
    run_all(d, page, 6)
    d.say("Then ping it, like on a real network.")
    row = page.locator("tr").filter(has=page.get_by_label("Address of FEED2 on this PC"))
    d.click(row.get_by_role("button", name="Ping"), after=0.3)
    reply = page.get_by_text(re.compile(rf"^Reply from {re.escape(MOVED)}"))
    reply.first.wait_for(timeout=30000)
    d.point(reply, pause=1.0)
    d.say("IEDScout, your SCADA and GOOSE reach the IEDs here, and nothing goes out on your Ethernet or Wi-Fi.")
