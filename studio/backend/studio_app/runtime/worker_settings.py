"""Derive the native start request settings and check the worker's capabilities."""
from studio_app.domain.network_settings import effective_endpoint

REQUIRED_CAPABILITIES = {"versioned_updates", "atomic_control_feedback", "control_selection_events",
                         "standard_controls", "goose_aware", "goose_bus"}
STALE_ENGINE = ("Native engine is outdated. Use the rebuilt portable application or rebuild the "
                "native worker before Run")


def goose_request(ied, bus):
    """Workers publish only to the app's GOOSE engine, in every network mode (GOOSE-05).
    `bus` is the engine interface for this IED ("udp:PORT:MAC"), never a network adapter."""
    publish = (bool(bus) and bool(ied.get("goose_publish", True))
               and bool((ied.get("goose") or {}).get("publishes")))
    return {"publish": publish, "interface": bus if publish else ""}


def start_request(ied, network, controls, config_path, bus=""):
    host, port = effective_endpoint(ied, network)
    return {"cfg": str(config_path), "ip": host, "port": port, "controls": controls,
            "edition": ied["edition"], "goose": goose_request(ied, bus)}


def check_capabilities(started):
    if not REQUIRED_CAPABILITIES <= set(started.get("capabilities", [])):
        raise ValueError(STALE_ENGINE)
