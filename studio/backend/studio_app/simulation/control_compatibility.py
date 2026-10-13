"""Resolve explicit non-standard native control profiles from diagram bindings."""


def profiles(diagram):
    result = {}
    for node in diagram.get("nodes", []):
        binding = node.get("data", {}).get("control_binding")
        if not binding:
            continue
        profile = binding.get("compatibility", "standard")
        if profile != "standard":
            result[(binding["ied_id"], binding["ref"])] = profile
    return result
