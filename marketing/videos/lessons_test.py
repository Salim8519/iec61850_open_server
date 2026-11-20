"""Lesson: the Test page (generators and timed sequences).

Public material only: the built-in sample IED.
"""
from __future__ import annotations

import re

from lessons_basics import button, nav, run_all, tab


def _label(page, name: str):
    """The input or select inside a <label> whose text starts with name (wrapped labels
    include the chosen option in their accessible name, so get_by_label cannot match exactly)."""
    label = page.locator("main label").filter(has_text=re.compile("^" + re.escape(name)))
    return label.locator("input, select").first


def _number(d, page, name: str, text: str) -> None:
    """Number boxes keep a 0 after clearing; select it so the typed value replaces it."""
    d.click(_label(page, name), after=0.1)
    page.keyboard.press("Control+A")
    page.keyboard.type(text, delay=32)
    d.wait(0.3)


def test_station(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    d.say("A test needs values that move by themselves, and steps that happen on time. The Test page does both.")
    d.click(button(page, "Use a sample IED"), after=0.6)
    d.click(nav(page, "Test"), after=0.4)
    d.click(tab(page, "Generators"), after=0.6)
    d.say("A generator moves a measurement for you while the IED runs.")
    d.click(page.locator("main").get_by_role("button", name=re.compile("New generator")).first, after=0.6)
    d.type(_label(page, "Search measurements"), "AnIn1")
    _label(page, "Measurement").select_option(index=1)
    d.wait(0.6)
    d.say("Pick how it changes: a smooth wave, a ramp, steps, random noise, or a recording from a CSV file.")
    d.point(_label(page, "How the value changes"), pause=0.3)
    _label(page, "How the value changes").select_option(label="Smooth wave")
    d.wait(0.4)
    _number(d, page, "Middle value", "132")
    _number(d, page, "Swing (±)", "4")
    _number(d, page, "Cycle time (seconds)", "10")
    d.click(page.locator("main").get_by_role("button", name=re.compile("Save generator")).first, after=0.8)
    d.click(tab(page, "Sequences"), after=0.6)
    d.say("A sequence plays timed steps after you press Run. Here: open the breaker after five seconds.")
    d.type(_label(page, "Sequence name"), "Breaker trip")
    _number(d, page, "Time (seconds)", "5")
    _label(page, "IED").select_option(index=1)
    d.type(_label(page, "Search references"), "XCBR1.Pos.stVal")
    _label(page, "Signal").select_option(index=1)
    d.type(_label(page, "Value"), "1")
    d.click(button(page, "Add step"), after=0.6)
    d.say("Add a Check value step to make it a test. A failed check stops the sequence and says why.")
    d.point(_label(page, "Action"), pause=0.6)
    d.click(_label(page, "Start automatically with Run"), after=0.5)
    d.say("Press Run. The wave starts, and five seconds later the breaker opens, every time the same.")
    run_all(d, page, 1)
    d.wait(1.0)
    d.point(page.locator("main").get_by_text("Breaker trip", exact=True).first, pause=5.0)
    d.click(tab(page, "Generators"), after=0.8)
    d.click(page.locator("main").get_by_role("button", name=re.compile("Edit")).first, after=0.8)
    d.point(page.locator("main").get_by_text("Value now").first, pause=2.5)
    d.say("Save the project, and the same test runs again tomorrow, or on another PC.")
    d.settle()

