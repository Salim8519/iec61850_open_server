"""Lesson 1, the opening video: what ScadaBench does for you, result first.

The owner found the old menu tour unmotivating (2026-11-20): this one shows a whole
substation coming alive from its SCD in about a minute, then where to go next.
Public demo station only (OpenSCD example), never customer files.
"""
from __future__ import annotations

import re

from lessons_basics import button, import_station, nav, run_all


def getting_started(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    d.say("No relays, no lab, no waiting. Here is a whole substation, running on one PC.")
    d.say("Start with the SCD file you already have.")
    import_station(d, page, narrate=False)
    d.say("Six IEDs, each with its full data model, read straight from the file.")
    d.click(button(page, "SLD"))
    d.click(button(page, "Build from SCD"), after=0.6)
    d.click(button(page, "Preview"), after=1.0)
    d.click(button(page, "Apply to diagram"), after=0.8)
    d.click(button(page, "Tidy up"), after=1.0)
    d.say("The single-line diagram draws itself from the same SCD.")
    d.say("Press Run. Every IED becomes a real IEC 61850 server.")
    run_all(d, page, 6)
    d.wait(0.8)
    d.say("Operate a breaker, exactly as your SCADA would.")
    d.click(page.locator(".react-flow__node").filter(has_text=re.compile(r"^CBR_F1")).first, after=0.8)
    panel = page.get_by_role("complementary", name="CBR_F1 properties")
    d.click(panel.get_by_role("button", name="Close", exact=True).first, after=1.6)
    d.say("The relay answers, and the diagram follows it live.")
    layer = page.get_by_role("switch").filter(has_text="GOOSE messages").first
    if layer.get_attribute("aria-checked") != "true":
        d.click(layer, after=0.8)
    d.say("Switch on GOOSE, and see which IED talks to which.")
    d.wait(1.2)
    d.click(nav(page, "Devices"), after=0.6)
    d.say("Point your SCADA, gateway or test set at these addresses, and test against the real protocol.")
    d.point(page.get_by_text(re.compile(r"^127\.50\.0\.\d+:102$")).first, pause=1.0)
    d.say("Factory tests, site tests and training, before the first relay arrives. This is ScadaBench.")
    d.settle()
