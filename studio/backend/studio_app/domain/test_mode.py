"""IEC 61850-7-4 mode and behaviour for ScadaBench's simulated IEDs, and mode summaries.

Pure. Mod/Beh values: 1 on, 2 on-blocked, 3 test, 4 test/blocked, 5 off.
"""
import re
from .health_reads import MODE_NAMES

_SIM = re.compile(r"^([^/]+)/LPHD\d*\.Sim\.stVal$")
_BEH = re.compile(r"^([^/]+)/([^.]+)\.Beh\.stVal$")
MODE_SUFFIXES = (".Mod.stVal", ".Beh.stVal", ".Sim.stVal")


def mode_signals(signals):
    """Reference and FC of the only points mode_points and mode_items read (Mod, Beh, Sim).

    A model of 90,000 points has a few hundred of these, so health can keep a small list.
    """
    return [{"ref": signal["ref"], "fc": signal["fc"]} for signal in signals
            if signal["ref"].endswith(MODE_SUFFIXES)]


def derive_beh(ld_mod, ln_mod=1):
    """Behaviour of a logical node from its logical device's Mod and its own Mod."""
    modes = (ld_mod, ln_mod)
    if 5 in modes:
        return 5
    test = any(mode in (3, 4) for mode in modes)
    blocked = any(mode in (2, 4) for mode in modes)
    return 4 if test and blocked else 3 if test else 2 if blocked else 1


def mode_points(ied):
    """Per logical device: its Mod reference, the Beh references of its nodes, and LPHD.Sim."""
    refs = {signal["ref"] for signal in ied["signals"]}
    devices = {}
    for ref in sorted(refs):
        match = _BEH.match(ref)
        if match:
            device = devices.setdefault(match.group(1), {"mod": None, "beh": []})
            device["beh"].append((ref, f"{match.group(1)}/{match.group(2)}.Mod.stVal"))
    for domain, device in devices.items():
        mod = f"{domain}/LLN0.Mod.stVal"
        device["mod"] = mod if mod in refs else None
    simulation = sorted(ref for ref in refs if _SIM.match(ref))
    return devices, simulation


def validate_test_mode(value):
    """Stored per-IED test settings: {devices: {domain: 1..5}, simulation: bool}."""
    value = value or {}
    devices = value.get("devices") or {}
    if not isinstance(devices, dict) or len(devices) > 100:
        raise ValueError("Test mode needs one mode per logical device")
    clean = {}
    for domain, mode in devices.items():
        if not isinstance(domain, str) or isinstance(mode, bool) or mode not in MODE_NAMES:
            raise ValueError("Each logical device mode must be 1 (on) to 5 (off)")
        clean[domain] = mode
    simulation = value.get("simulation", False)
    if not isinstance(simulation, bool):
        raise ValueError("Simulation must be on or off")
    return {"devices": clean, "simulation": simulation}


def mode_writes(ied, settings, current):
    """(reference, value) writes for a simulated IED; `current` maps refs to present values."""
    devices, simulation = mode_points(ied)
    writes = []
    for domain, mode in settings["devices"].items():
        device = devices.get(domain)
        if device is None:
            continue
        if device["mod"]:
            writes.append((device["mod"], mode))
        for beh, own_mod in device["beh"]:
            ln_mod = mode if own_mod == device["mod"] else current.get(own_mod, 1)
            writes.append((beh, derive_beh(mode, ln_mod if ln_mod in MODE_NAMES else 1)))
    writes += [(ref, settings["simulation"]) for ref in simulation]
    return writes


def mode_summary(ied, values):
    """Mode of each logical device from read values, worst first, for badges."""
    devices, simulation = mode_points(ied)
    result = {"devices": {}, "simulation": None, "worst": None}
    rank = {name: index for index, name in enumerate(MODE_NAMES.values())}
    for domain in devices:
        value = values.get(f"{domain}/LLN0.Beh.stVal", values.get(f"{domain}/LLN0.Mod.stVal"))
        name = MODE_NAMES.get(value) if isinstance(value, int) and not isinstance(value, bool) else None
        result["devices"][domain] = name or "not available"
        if name and (result["worst"] is None or rank[name] > rank[result["worst"]]):
            result["worst"] = name
    sims = [values.get(ref) for ref in simulation if isinstance(values.get(ref), bool)]
    result["simulation"] = any(sims) if sims else None
    return result
