"""Lesson 13: activate or go offline. Real license screens with demo answers."""
from __future__ import annotations

import json

from lessons_basics import button, tab

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
    d.say("Using a virtual machine on a licensed PC? This box shows how the VM uses that PC's license.")
    d.point(page.get_by_text("Using a virtual machine?").first, pause=0.8)
    d.say("Have a key? It is in your purchase email, and works on one computer. Paste it here.")
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
    d.say("Each license stays on this computer. Reinstalling Windows on the same PC is fine.")
    d.say("No internet on site? Copy the Machine ID, and use the offline page on any other PC.")
    d.settle()
    page.goto("https://license.scadabench.com/activate")
    page.get_by_text("without internet").first.wait_for(timeout=20000)
    d.type(page.locator("#key"), "SUBS-7K4P-QX2M-9TDF-R8WC", after=0.2)
    d.type(page.locator("#hwid"), DEMO_HWID, after=0.4)
    d.say("Download the license file, and load it on the offline PC.")
