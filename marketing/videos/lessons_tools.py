"""Lessons 7-8: IED Explorer with Find IEDs, and the SCADA gateway. Both run against the
public demo station simulated on this PC (127.50.0.x), so no real equipment is touched."""
from __future__ import annotations

import re

from lessons_basics import button, import_station, run_all, tab


def open_tool(d, page, name: str) -> None:
    d.click(button(page, "Tools"), after=0.4)
    back = button(page, "All tools")
    if back.count():
        d.click(back, after=0.4)
    d.click(page.get_by_role("button", name=re.compile(f"^{re.escape(name)}")).first, after=0.8)


def ied_explorer(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    import_station(d, page, narrate=False)
    d.say("Tools has an IEC 61850 client built in. To practise, run the demo station first.")
    run_all(d, page, 6)
    d.say("Find IEDs scans a network for port 102, and asks each device who it is.")
    open_tool(d, page, "Find IEDs")
    adapter = page.locator("main select").first
    d.point(adapter)
    adapter.select_option(label=page.evaluate("""() => [...document.querySelectorAll('main select option')]
        .map(o => o.textContent).find(t => t.startsWith('This PC')) || ''"""))
    d.say("This PC lists the simulated IEDs, so you can try it without a cable.")
    d.click(button(page, "Find IEDs"), after=0.5)
    page.get_by_role("button", name="Open in Explorer").first.wait_for(timeout=60000)
    d.wait(1.0)
    d.say("Every device answers with its vendor, model and logical devices. Open one in IED Explorer.")
    d.click(page.get_by_role("button", name="Open in Explorer").first, after=1.0)
    button(page, "Disconnect").first.wait_for(timeout=30000)
    d.say("Explorer connects and opens the data model. Key points are watched by themselves.")
    d.point(page.get_by_text(re.compile(r"^Watch list")).first, pause=0.8)
    d.say("It never takes a report block, so SCADA and gateways keep theirs.")
    d.say("Open a logical node, like CSWI1, the breaker control.")
    d.click(page.get_by_role("button", name="CSWI1", exact=True).first, after=0.8)
    d.click(page.locator("button.object-toggle").filter(has_text=re.compile(r"^Pos")).first, after=1.0)
    d.say("Send a command. ScadaBench always asks once before it operates anything.")
    d.click(page.get_by_role("button", name="Close…").first, after=0.6)
    d.say("It uses the device's own control model, select before operate when it asks for it.")
    d.click(page.get_by_role("button", name="Yes, send Close").first, after=1.6)
    d.say("The answer shows the result, and what the device reads now.")
    d.point(page.get_by_role("status").filter(has_text=re.compile("now reads|Close")).first, pause=1.2)
    d.say("Export the watch list as CSV for your test record.")
    d.point(page.locator("a, button").filter(has_text=re.compile(r"^Export CSV$")).last, pause=0.8)


def scada_gateway(d, page, data) -> None:
    d.click(button(page, "Skip tour"), after=0.3)
    import_station(d, page, narrate=False)
    d.say("A substation gateway serves the IEDs to the control centre over a SCADA protocol.")
    run_all(d, page, 6)
    d.say("In Tools, open SCADA gateway and master.")
    open_tool(d, page, "SCADA gateway")
    d.say("This picture tells you which side ScadaBench plays. Here, it is the station gateway.")
    d.point(page.get_by_text("(you are here)").first, pause=0.8)
    d.say("Pick IEC 104, Modbus TCP or DNP3, and who can connect.")
    settings = page.get_by_role("complementary", name="Gateway settings").locator("select")
    d.point(settings.nth(0), pause=0.5)
    d.point(settings.nth(1), pause=0.5)
    d.say("Press Start gateway. ScadaBench proposes the point list from your IEDs.")
    d.click(button(page, "Start gateway"), after=0.6)
    page.get_by_text(re.compile(r"^Listening on")).first.wait_for(timeout=60000)
    d.wait(1.0)
    d.say("Each point has its IOA, its IED signal and its live value. Your SCADA connects to this address.")
    d.point(page.get_by_text(re.compile(r"^Listening on")).first, pause=0.8)
    d.say("Commands work too. Send Close to a breaker, just like SCADA would.")
    main = page.locator("main")
    commands = main.locator("button, [role=tab], [role=radio]").filter(has_text=re.compile(r"^Commands$"))
    d.click(commands.nth(1), after=0.8)
    d.click(main.get_by_role("button", name="Close…").first, after=0.6)
    d.say("ScadaBench warns you first, then operates the IED over real IEC 61850.")
    d.click(main.get_by_role("button", name="Yes, send Close").first, after=1.6)
    d.click(commands.nth(0), after=1.2)
    d.say("The Commands tab lists every command, who sent it, and what changed.")
    d.say("Switch to SCADA master to test a gateway or RTU instead, or Protocol converter for both.")
    d.point(tab(page, "SCADA master"), pause=0.6)
    d.point(tab(page, "Protocol converter"), pause=0.8)
