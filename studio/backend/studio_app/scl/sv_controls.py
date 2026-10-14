"""Read SampledValueControl blocks of one IED: the streams Studio expects to hear."""

SMP_MOD = {"SmpPerPeriod": 0, "SmpPerSec": 1, "SecPerSmp": 2}
SCALE = {"TCTR": 0.001, "TVTR": 0.01}  # 9-2LE: current in mA, voltage in 10 mV


def _channels(fcdas):
    """8 channels named after their data objects when the dataset is a 9-2LE layout."""
    whole = [item for item in fcdas if not item.get("daName")]
    split = [item for item in fcdas if item.get("daName")]
    if len(whole) == 8 and not split:
        objects = whole
    elif len(split) == 16 and not whole and all(
            a.get("daName") == "instMag.i" and b.get("daName") == "q"
            and (a.get("lnClass"), a.get("lnInst"), a.get("doName")) == (b.get("lnClass"), b.get("lnInst"), b.get("doName"))
            for a, b in zip(split[0::2], split[1::2])):
        objects = split[0::2]
    else:
        return None
    if any(item.get("lnClass") not in SCALE for item in objects):
        return None
    names = [f"{item.get('prefix', '')}{item.get('lnClass')}{item.get('lnInst', '')}.{item.get('doName')}"
             for item in objects]
    return names, [SCALE[item.get("lnClass")] for item in objects]


def sv_controls(ln0, domain, inst, smv_addresses, ied_name, warnings):
    """Every SampledValueControl with an SMV address; anything else is a warning."""
    addresses = {(item["ld_inst"], item["cb_name"]): item for item in smv_addresses}
    datasets = {item.get("name"): item for item in ln0.findall("DataSet")}
    result = []
    for element in ln0.findall("SampledValueControl"):
        name = element.get("name", "")
        where = f"Sampled Values block {ied_name}/{inst}/{name}"
        address = addresses.get((inst, name))
        if address is None:
            warnings.append(f"{where} has no network address (SMV) in the SCD; it is not listened to")
            continue
        try:
            conf_rev = int(element.get("confRev", "0"))
            smp_rate = int(element.get("smpRate", "0"))
            asdus = int(element.get("nofASDU", "1"))
        except ValueError:
            warnings.append(f"{where} has an unreadable confRev, smpRate or nofASDU; it is not listened to")
            continue
        dataset = datasets.get(element.get("datSet"))
        layout = _channels(dataset.findall("FCDA")) if dataset is not None else None
        result.append({"name": name, "ld_inst": inst, "svcb_ref": f"{domain}/LLN0.{name}",
                       "sv_id": element.get("smvID", ""), "dataset": element.get("datSet", ""),
                       "dataset_ref": f"{domain}/LLN0${element.get('datSet', '')}", "conf_rev": conf_rev,
                       "smp_rate": smp_rate, "smp_mod": SMP_MOD.get(element.get("smpMod", "SmpPerPeriod"), 0),
                       "nof_asdu": asdus, "mac": address["mac"], "app_id": address["app_id"],
                       "vlan_id": address["vlan_id"], "layout": "9-2LE" if layout else "raw",
                       "channels": layout[0] if layout else [], "scale": layout[1] if layout else [],
                       "members": len(dataset.findall("FCDA")) if dataset is not None else 0})
    return result
