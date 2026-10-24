"""Lessons 7, 8 and 12: IED Explorer with Find IEDs, the SCADA gateway, and SNMP devices. They run
against the public demo station or a simulated switch on this PC (127.x), so no real equipment is touched."""
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
    d.say("Send a command. It follows the device's own control model: this breaker asks for select before operate.")
    d.click(page.get_by_role("button", name="Select Close…").first, after=0.6)
    d.say("ScadaBench always asks once. Select only reserves the control; nothing moves yet.")
    d.click(page.get_by_role("button", name="Yes, select Close").first, after=1.0)
    d.say("Selected, with a countdown. Press Operate before it runs out.")
    d.click(page.get_by_role("button", name="Operate", exact=True).first, after=1.6)
    d.say("The answer shows the result, and what the device reads now.")
    d.point(page.get_by_text(re.compile("selected and operated it")).first, pause=0.8)
    d.point(page.get_by_text("on (closed)").first, pause=0.8)
    d.say("Export the watch list as CSV for your test record.")
    d.click(page.locator("button, [role=tab]").filter(has_text=re.compile(r"^Watch list")).first, after=0.8)
    d.click(page.get_by_role("button", name=re.compile(r"^More")).last, after=0.6)
    d.point(page.get_by_text("Export CSV", exact=True).last, pause=0.8)
    page.keyboard.press("Escape")


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
    main = page.locator("main")
    d.say("Add points adds points from an IED, made-up points you change yourself, or rows pasted from Excel.")
    d.click(main.get_by_role("button", name="Add points").first, after=1.0)
    d.point(page.get_by_text("Paste from Excel").first, pause=0.8)
    d.click(page.get_by_role("button", name="Close without adding").first, after=0.5)
    d.say("Or just press Start gateway. ScadaBench proposes the point list from your IEDs.")
    d.click(button(page, "Start gateway"), after=0.6)
    page.get_by_text(re.compile(r"^Listening on")).first.wait_for(timeout=60000)
    d.wait(1.0)
    d.say("Each point has its type, address, IED signal and live value. Your SCADA connects to this address.")
    d.point(page.get_by_text(re.compile(r"^Listening on")).first, pause=0.8)
    d.say("Commands work too. Show the double commands, and send Close to a breaker, just like SCADA would.")
    kinds = main.locator("select").filter(has_text="All types").first
    d.point(kinds)
    labels = kinds.locator("option").all_inner_texts()
    kinds.select_option(index=next(i for i, text in enumerate(labels) if "Double command" in text))
    d.wait(0.8)
    d.click(main.locator("tbody tr").first.get_by_role("button", name="Close…"), after=0.6)
    d.say("ScadaBench warns you first, then operates the IED over real IEC 61850.")
    d.click(main.get_by_role("button", name="Yes, send Close").first, after=1.6)
    d.click(tab(page, "Commands"), after=1.2)
    d.say("The Commands tab lists every command, who sent it, and what changed.")
    d.say("Switch to SCADA master to test a gateway or RTU instead. Its point list marks each point received, missing or wrong type.")
    d.point(tab(page, "SCADA master"), pause=0.6)
    d.say("Protocol converter plays both sides at once.")
    d.point(tab(page, "Protocol converter"), pause=0.8)


def snmp_devices(d, page, data) -> None:
    """Tools > SNMP devices: simulate a network switch, then read it back on this PC."""
    d.click(button(page, "Skip tour"), after=0.3)
    d.say("Substations also have network switches and GPS clocks. SCADA watches them over SNMP.")
    open_tool(d, page, "SNMP devices")
    d.say("Simulate devices plays them. Add a network switch with eight ports.")
    d.click(button(page, "Add device"), after=0.5)
    d.click(page.get_by_role("menuitem", name=re.compile("^Network switch")).or_(
        page.get_by_role("button", name=re.compile("^Network switch"))).first, after=1.0)
    d.say("Press Start devices. The switch now answers on its own address, port 161.")
    d.click(button(page, "Start devices"), after=1.5)
    page.get_by_text(re.compile(r"answers SNMP$")).first.wait_for(timeout=30000)
    d.say("Its values have names. Counters grow and the uptime counts, like on a real switch.")
    d.wait(1.5)
    d.say("Read devices is the other side: ScadaBench reads any SNMP device. The address is filled in.")
    d.click(tab(page, "Read devices"), after=0.8)
    d.click(button(page, "Connect and read everything"), after=0.5)
    page.get_by_text(re.compile(r"^Connected$")).first.wait_for(timeout=60000)
    d.wait(1.0)
    d.say("It walks every value once, puts the important ones on top, and re-reads them by itself.")
    d.wait(1.0)
    d.say("Traps arrive here too. Versions one, two c and three all work.")
