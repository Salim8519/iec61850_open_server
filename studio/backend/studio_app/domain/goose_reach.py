"""Where GOOSE goes (ADR-0073): inside the app by default, or also shared with the network.

Pure rules: which adapter GOOSE uses, how risky sharing is, and the plain status line.
"""

REACHES = ("inside", "outside")
REACH_KEYS = ("goose_reach", "goose_uplink_guid", "goose_share_guid")

NO_ADAPTER = ("No connected wired network adapter was found, so GOOSE stays inside ScadaBench. "
              "Plug in a network cable, or choose an adapter under Advanced.")


def is_npcap_loopback(adapter):
    text = f"{adapter.get('name', '')} {adapter.get('description', '')}".lower()
    return "npcap" in text and "loopback" in text


def _choice(adapter, auto, why):
    return {"guid": adapter["guid"], "label": adapter.get("name") or adapter.get("description") or adapter["guid"],
            "kind": adapter.get("kind", ""), "connected": adapter.get("status") == "Up", "auto": auto,
            "why": why, "reason": ""}


def pick_uplink(adapters, network):
    """The adapter GOOSE listens on (and sends on, when shared). GOOSE-24.

    Order: the adapter chosen under Advanced; the lab adapter in lab mode; then a
    connected wired adapter, preferring the one with the default route. Wi-Fi, VPN,
    virtual and loopback adapters are never picked automatically.
    """
    by_guid = {adapter["guid"]: adapter for adapter in adapters}
    chosen = network.get("goose_uplink_guid") or ""
    if chosen:
        if chosen in by_guid:
            return _choice(by_guid[chosen], False, "chosen")
        return {"guid": "", "label": "", "kind": "", "connected": False, "auto": False, "why": "chosen",
                "reason": "The adapter chosen under Advanced is not on this PC any more, so GOOSE stays inside."}
    lab = network.get("adapter_guid") or ""
    if network.get("mode") == "lab" and lab in by_guid:
        return _choice(by_guid[lab], True, "lab")
    wired = [adapter for adapter in adapters if adapter.get("kind") == "ethernet"
             and adapter.get("status") == "Up" and not is_npcap_loopback(adapter)]
    wired.sort(key=lambda adapter: (not adapter.get("default_route"), adapter.get("name", "").lower()))
    if wired:
        return _choice(wired[0], True, "wired")
    return {"guid": "", "label": "", "kind": "", "connected": False, "auto": True, "why": "", "reason": NO_ADAPTER}


def advanced_choices(adapters):
    """Adapters the user may pick under Advanced, with a warning where GOOSE is unreliable."""
    choices = []
    for adapter in adapters:
        if is_npcap_loopback(adapter) or adapter.get("kind") in {"bluetooth", "vpn"}:
            continue
        warning = ("Wi-Fi usually drops GOOSE from simulated IEDs; use a cable when you can."
                   if adapter.get("kind") == "wifi" else "")
        choices.append({**_choice(adapter, False, "chosen"), "warning": warning})
    return choices


def share_risk(network, uplink, ieds, heard_real_goose):
    """How dangerous sharing is on this adapter. 'high' when real devices are likely to act on it."""
    reasons = []
    if network.get("mode") == "lab" and uplink.get("guid") and uplink["guid"] == network.get("adapter_guid"):
        reasons.append("Your simulated IEDs use real network addresses on this adapter, so real devices "
                       "may treat their GOOSE as coming from the real IEDs.")
    if heard_real_goose:
        reasons.append("Real GOOSE is already being received on this adapter, so real devices are connected.")
    if any(ied.get("role") == "real" for ied in ieds):
        reasons.append("This project also has real IEDs; a real IED may already send the same GOOSE.")
    level = "high" if reasons else "warning"
    if uplink.get("kind") == "wifi":
        reasons.append("Wi-Fi usually drops GOOSE from simulated IEDs.")
    return {"level": level, "reasons": reasons}


def share_hold(network, uplink, npcap_installed):
    """Why sharing that the user switched on cannot run right now, or '' when it can. GOOSE-19."""
    if network.get("goose_reach") != "outside":
        return ""
    if not npcap_installed:
        return "GOOSE sharing needs the free Npcap driver. Install it from npcap.com, then press Run again."
    if not uplink.get("guid"):
        return uplink.get("reason") or NO_ADAPTER
    if network.get("goose_share_guid") != uplink["guid"]:
        return (f"GOOSE sharing is paused because the adapter is now {uplink['label']}. "
                "Switch sharing on again to confirm this adapter.")
    return ""


def reach_status(network, uplink, hold, link):
    """The single status line shown for GOOSE while running, as (state, message)."""
    label = uplink.get("label") or "the network adapter"
    if hold:
        return "share_held", hold
    share = (link or {}).get("share", "off")
    if share == "sharing":
        return "sharing", f"Sharing GOOSE with the network on {label}. The smart guard is on."
    if share == "checking":
        return "checking", f"Checking {label} for real devices before sharing GOOSE…"
    if (link or {}).get("uplink") == "open":
        return "listening", (f"GOOSE runs inside ScadaBench and listens to real devices on {label}. "
                             "Nothing is sent to the network.")
    return "inside", "GOOSE runs inside ScadaBench. Nothing is sent to the network."
