"""Decide how ScadaBench reads a real IED: free report blocks first, polling otherwise."""


def short_name(rcb_ref):
    return rcb_ref.rsplit(".", 1)[-1]


def choose_access(ied, block_status, access="auto_reports"):
    """block_status: device answers for every SCD report block, keyed by reference.

    Returns {"reports": [rcb refs to use], "polling_datasets": [...], "notes": [...]}.
    A block is used only when the device itself says it is free.
    """
    notes, chosen, polling = [], [], []
    by_dataset = {}
    for report in ied.get("reports", []):
        if report.get("dataset_ref"):
            by_dataset.setdefault(report["dataset_ref"], []).append(report["ref"])
    if access == "polling":
        notes.append("Read-only polling is selected, so ScadaBench will not use any report block.")
        return {"reports": [], "polling_datasets": sorted(by_dataset), "notes": notes}
    for dataset, blocks in sorted(by_dataset.items()):
        name = dataset.rsplit("$", 1)[-1]
        states = [block_status.get(ref) or {"readable": False} for ref in blocks]
        free = [ref for ref, state in zip(blocks, states) if state.get("readable") and state.get("free")]
        if free:
            chosen.append(free[0])
            notes.append(f"ScadaBench will use report block {short_name(free[0])} for '{name}' because the IED "
                         "shows it as free. It is given back when you press Stop.")
            continue
        polling.append(dataset)
        if not any(state.get("readable") for state in states):
            notes.append(f"The IED did not answer for the report blocks of '{name}', so ScadaBench will poll "
                         "these values instead.")
        else:
            notes.append(f"All {len(blocks)} report block(s) for '{name}' are in use by other clients "
                         "(for example SCADA or a gateway). ScadaBench will not take them and will poll "
                         "these values instead.")
    return {"reports": chosen, "polling_datasets": polling, "notes": notes}
