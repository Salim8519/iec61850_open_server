"""Allocate private loopback MMS listener addresses for simulated IEDs."""
from studio_app.contracts.errors import StudioError


def assign_endpoints(ieds):
    used = {item["ip"] for item in ieds if item.get("ip")}
    for ied in ieds:
        if not ied.get("ip"):
            for index in range(2, 255):
                candidate = f"127.50.0.{index}"
                if candidate not in used:
                    ied["ip"] = candidate
                    used.add(candidate)
                    break
            else:
                raise StudioError("No free loopback addresses")
