"""Which read-only references the station health check reads from a real IED."""
import re

GOCB_ATTRIBUTES = ("GoEna", "GoID", "DatSet", "ConfRev")
MODE_NAMES = {1: "on", 2: "on-blocked", 3: "test", 4: "test/blocked", 5: "off"}
_LPHD_SIM = re.compile(r"^[^/]+/LPHD\d*\.Sim\.stVal$")


def gocb_object(gocb_ref):
    """'LD/LLN0$GO$name' (SCL/MMS style) -> 'LD/LLN0.name' (client object reference)."""
    return gocb_ref.replace("$GO$", ".", 1)


def gocb_items(ied):
    return [{"ref": f"{gocb_object(control['gocb_ref'])}.{name}", "fc": "GO"}
            for control in (ied.get("goose") or {}).get("publishes", []) for name in GOCB_ATTRIBUTES]


def mode_items(ied):
    """Mod/Beh of every logical device and LPHD.Sim, only where the SCD model has them."""
    refs = {signal["ref"] for signal in ied["signals"] if signal["fc"] == "ST"}
    domains = sorted({ref.split("/", 1)[0] for ref in refs})
    wanted = [f"{domain}/LLN0.{name}.stVal" for domain in domains for name in ("Mod", "Beh")]
    wanted += sorted(ref for ref in refs if _LPHD_SIM.match(ref))
    return [{"ref": ref, "fc": "ST"} for ref in wanted if ref in refs]


def parse_answer(answer):
    """{ref: value} for readable items and {ref: error} for the rest."""
    values, errors = {}, {}
    for entry in answer.get("values", []):
        if "value" in entry:
            values[entry["ref"]] = entry["value"]
        elif "structure" in entry:
            values.update(entry["structure"])
        else:
            errors[entry["ref"]] = entry.get("error")
    return values, errors


def mode_name(value):
    return MODE_NAMES.get(value) if isinstance(value, int) and not isinstance(value, bool) else None
