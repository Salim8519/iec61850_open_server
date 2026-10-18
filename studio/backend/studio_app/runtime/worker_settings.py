"""Derive the native start request settings and check the worker's capabilities."""
from studio_app.domain.network_settings import effective_endpoint

REQUIRED_CAPABILITIES = {"versioned_updates", "atomic_control_feedback", "control_selection_events",
                         "standard_controls", "goose_aware"}
STALE_ENGINE = ("Native engine is outdated. Use the rebuilt portable application or rebuild the "
                "native worker before Run")


def goose_request(ied, network):
    """GOOSE can only travel on a real adapter, so loopback mode never publishes."""
    publish = (network.get("mode") == "lab" and bool(network.get("adapter_guid"))
               and bool(ied.get("goose_publish", True)) and bool((ied.get("goose") or {}).get("publishes")))
    return {"publish": publish, "adapter_guid": network.get("adapter_guid", "") if publish else ""}


def start_request(ied, network, controls, config_path):
    host, port = effective_endpoint(ied, network)
    return {"cfg": str(config_path), "ip": host, "port": port, "controls": controls,
            "edition": ied["edition"], "goose": goose_request(ied, network)}


def check_capabilities(started):
    if not REQUIRED_CAPABILITIES <= set(started.get("capabilities", [])):
        raise ValueError(STALE_ENGINE)
