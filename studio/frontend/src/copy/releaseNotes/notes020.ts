/** Release notes 0.1.20 to 0.1.29, newest first (change 0238): shown in the app and on scadabench.com. */
import type { Release } from './types.ts';

export const RELEASES_020: Release[] = [
  { version: '0.1.29', title: 'Lab addresses only while your IEDs run', notes: [
    'Changed: on the ScadaBench Lab network, an IED\'s address is now on your PC only while that IED runs. "Use the lab network" saves the addresses; Start puts them on, and Stop takes them off again. Before, they stayed on the PC until Windows restarted, even if you never pressed Start.',
    'Closing ScadaBench also takes the addresses off. If ScadaBench was closed by force, it cleans up the leftover addresses the next time it opens.',
    'Ping an IED after you press Start. Changing an address while running still moves the IED at once.',
  ] },
  { version: '0.1.28', title: 'Lab network on this PC, GOOSE follows your IEDs', notes: [
    'New: a lab network on this PC. Each simulated IED answers at its own address from the SCL file, for example 192.168.216.1, without a virtual machine. Ping, MMS clients such as IEDScout or a SCADA, and GOOSE work on those addresses, and nothing goes out on your Ethernet or Wi-Fi.',
    'Windows asks for permission once, to add the "ScadaBench Lab" card (the Windows loopback adapter, no download, no restart). After that, Use and every address change need no prompt. Type another address and it moves at once, even while running; the SCL file is never changed.',
    'Clash checks: an address used twice or one of your PC\'s own addresses is refused; when a real device on your Wi-Fi or Ethernet already answers at an address, ScadaBench warns you and offers a free one.',
    'New: GOOSE now follows your IEDs. It goes to the same network as your simulated IEDs (the lab network, a real card, or only inside ScadaBench), and sending next to a real device asks you once first. A missing Npcap is one clear line with a download link.',
    'Uninstalling ScadaBench also removes the lab card and its service.',
  ] },
  { version: '0.1.27', title: 'NR/NARI relays and SCD deadbands', notes: [
    'New: NR and NARI relays (for example PCS-9705 and PCS-921S) now import. They have three access points; ScadaBench simulates the station bus one, S1, which a SCADA connects to, exactly as the SCD defines it. The import dialog says which access points (process-bus GOOSE on G1, Sampled Values on M1) are not simulated.',
    'New: deadbands from the SCD. Each simulated IED has a "Use SCD deadbands" switch. When it is on, a measurement is reported only when it moves more than its db, like a real IED. It is off by default, so nothing changes until you turn it on.',
    'New: the IED page lists every measurement\'s deadband in plain words. A measurement without a measuring range (rangeC) never gets a deadband, and the list says so.',
    'New: a SCADA can change a measurement\'s db or zeroDb over MMS when the SCD allows it (valKind Set or not given). Read-only values stay refused, as on the real IED.',
    'Values you type by hand always go out; only automatic sources are filtered by a deadband.',
  ] },
  { version: '0.1.26', title: 'Lighter and faster with many IEDs', notes: [
    'Faster: ScadaBench now needs about half the memory and a third of the CPU for large stations. On a 20-IED station memory dropped from about 1.6 GB to under 0.8 GB at its peak, and the project opens and starts faster.',
    'Faster: Run starts sooner (2.3 s instead of 3.2 s for 20 IEDs), reopening a project is about twice as fast, and SLD commands answer a little quicker.',
    'Memory is given back to Windows after opening a project, Run and Stop, so ScadaBench no longer keeps showing its highest memory use.',
    'Nothing else changes: the same screens, the same IED behaviour on the network, and the same reports and GOOSE messages.',
  ] },
  { version: '0.1.25', title: 'IEDs on the diagram, a smart point picker, many vendor files fixed', notes: [
    'New: an IED symbol for the single-line diagram. GOOSE now travels the way it does in a real station: from the whole sending IED to every IED that subscribes to it. "Add all GOOSE IEDs" places them for you, and a chip names subscribers that are not on the drawing.',
    'New: send a GOOSE message from the IED symbol, from any equipment\'s GOOSE link (one value, or open and closed separately), or from a point panel row.',
    'New: a smarter point picker. Chips filter by point type (single point, double point, measurement, setpoint, integer, command and more) in the usual colours, with counts, a filter by logical node and search by number or description. Suggestions name the equipment you are binding.',
    'Fixed: many SCL files from ABB, Siemens, GE, SEL, Schneider, NR and others that could not be imported before now import (Unicode text, unusual names, enum defaults, trigger options, datasets without a data object name, files up to 100 MB). Import errors now say where in the file the problem is.',
    'Fixed: Beh follows Mod as the standard describes, large IEDs start reliably, quieter logs, correct timestamps at Run start and the command originator shown as "ScadaBench".',
  ] },
  { version: '0.1.24', title: 'Your drawing always in view, push buttons as point lists', notes: [
    'Fixed: the single-line diagram could open on an empty area when one symbol lay far away from the rest of the drawing. The view now shows your drawing, and a note names the far-away symbol with a "Bring it here" button.',
    'New: "Put push buttons into point panels" in a push button\'s properties turns your push buttons into one point panel per IED. Status buttons become ON/OFF rows and command buttons become command rows, with the same signals. Undo brings the buttons back.',
  ] },
  { version: '0.1.23', title: 'Many fixes from a full test of the app', notes: [
    'IED Explorer no longer selects a control just because you opened it. Reading a select-before-operate control could reserve it on the device; that field is now never read.',
    'A command sent with the Test option no longer moves the breaker on the diagram when the IED is in service and ignores it.',
    'Tap-changer transformer: its settings list the tap changer controls of your IEDs again, and changing the IED never removes a working setup.',
    'After Stop and Run, the diagram shows the values the IEDs really start with, not the ones from the last run.',
    'When a device refuses Cancel, the selection is kept so Operate still works, and the reason is explained.',
    'DNP3 gateway: output statuses are reported as good and command results arrive at once as events.',
    'SCADA tools: Find registers has a Stop button, a refused connection says nothing is listening, Modbus writes from the master can choose the word order, and the protocol converter has the What to read tab for Modbus devices.',
    'Activity is no longer filled with empty "report created" rows, menus close properly, Enter keeps the confirm question open, and many texts are clearer and no longer cut off at 1366×768.',
  ] },
  { version: '0.1.22', title: 'Commands everywhere, point panels, tap changers, SCADA point lists', notes: [
    'Every IEC 61850 command type: besides breakers and switches you can now send integer and enum commands, analogue set points, binary step and integer step commands (tap raise / lower), in all four control models, from IED Explorer and the single-line diagram. Tap and set point commands also work through the SCADA gateway.',
    'Select before operate is one click on the single-line diagram, like a station HMI: ScadaBench selects and operates for you. Turn on "Show Select before Operate step by step" under Command options to see each step and its countdown.',
    'New Point panel in the single-line diagram: many points of one IED in one box. Read statuses, positions, measurements, tap positions and counters, and send commands or set points from a small popup.',
    'New tap-changer transformer symbol with tap position, Raise / Lower, Auto / Manual and a simulated voltage regulator.',
    'GOOSE on the diagram is clearer: one change shows one note ("Open · 7 of 7 IEDs got it") and one line to each IED that receives it.',
    'SCADA gateway: Add points from your IEDs, from an Excel template or by typing them, every row checked first. New point types for tap position, step commands and set points on IEC 104, DNP3 and Modbus.',
    'SCADA master: the most used commands of each protocol in one click, "Repeat automatically", and your own point list: add the addresses your SCADA expects (or copy them from the gateway) and see what was received, what is missing and what has the wrong type.',
  ] },
  { version: '0.1.21', title: 'Clearer VM setup, saves by itself, safer network setup', notes: [
    'The activation screen has a new "Using a virtual machine?" section next to the offline options, with the three steps to use your PC\'s license inside a VM. Both options start closed, so the screen stays short.',
    'A VM keeps looking for your licensed PC by itself and opens ScadaBench as soon as it finds it. Nothing to type in the VM.',
    'Fixed: right after you turn on "Let my virtual machines use this license", Settings no longer says the helper is not running while it is still starting.',
    'New picture guide on the tutorials page: Use ScadaBench in a virtual machine.',
    'Everything saves by itself: SNMP device settings, the SCADA gateway point list, test sequences and new gateway points no longer need a Save button. A wrong value shows a red line under the field and is not saved until you fix it.',
    'Undo in the single-line diagram now works from your very first change, and deleting a wired symbol comes back with its wires in one Undo.',
    'Network setup checks the addresses you type: an address another device already uses, or your PC\'s own address, is shown at once and cannot be applied. The button shows "Checking the addresses…" while it checks.',
    'Many small screen fixes at 1366×768: no more cut-off buttons, names or menus in the tools, the diagram side panel, Activity and the sidebar.',
  ] },
  { version: '0.1.20', title: 'Activate once, works forever; virtual machines on your PC', notes: [
    'Activate your license once and it works for life on that computer, with or without internet. Reinstalling Windows on the same computer is fine.',
    'Virtual machines on your licensed PC can use its license, even offline: turn on "Let my virtual machines use this license" in Settings > License.',
    'Stronger protection: each license works on one computer only, a cancelled license stays cancelled, and the app only trusts answers signed by the ScadaBench license server.',
    'Project files are now .scadabench files.',
  ] },
];
