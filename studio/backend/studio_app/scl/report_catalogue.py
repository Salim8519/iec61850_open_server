"""Resolve report identities and exact dataset membership for engineering pickers."""
from studio_app.contracts.errors import StudioError
from bisect import bisect_left
from collections import defaultdict


def enrich_reports(server, ied_name, datasets, reports, signals):
    by_fc = defaultdict(list)
    for signal in signals:
        by_fc[signal["fc"]].append(signal["ref"])
    for references in by_fc.values():
        references.sort()
    domains = {ld.get("inst"): ld.get("ldName") or ied_name + ld.get("inst")
               for ld in server.findall("LDevice")}
    dataset_iter, report_iter = iter(datasets), iter(reports)
    for ld in server.findall("LDevice"):
        for ln in ld:
            if ln.tag not in {"LN", "LN0"}:
                continue
            node = "LLN0" if ln.tag == "LN0" else (
                ln.get("prefix", "") + ln.get("lnClass", "") + ln.get("inst", ""))
            owner = domains[ld.get("inst")] + "/" + node
            local = {}
            for declaration in ln.findall("DataSet"):
                dataset = next(dataset_iter)
                resolved, unresolved, fcda_refs = set(), [], []
                for member in declaration.findall("FCDA"):
                    domain = domains.get(member.get("ldInst", ld.get("inst")))
                    member_node = member.get("prefix", "") + member.get("lnClass", "") + member.get("lnInst", "")
                    path = member.get("doName", "")
                    if member.get("ix") is not None:
                        path += "(" + str(int(member.get("ix"))) + ")"
                    if member.get("daName"):
                        path += "." + member.get("daName")
                    reference = (domain or "<unresolved LD>") + "/" + member_node + "." + path
                    fcda_refs.append(reference)
                    references = by_fc[member.get("fc", "ST")]
                    start = bisect_left(references, reference)
                    end = bisect_left(references, reference + "\uffff")
                    matches = [ref for ref in references[start:end] if domain and
                               (ref == reference or ref.startswith(reference + ".") or ref.startswith(reference + "("))]
                    if matches:
                        resolved.update(matches)
                    else:
                        unresolved.append({"ref": reference, "fc": member.get("fc", "ST")})
                dataset.update(ref=owner + "$" + dataset["name"], signal_refs=sorted(resolved),
                               unresolved=unresolved, fcda_refs=fcda_refs)
                local[dataset["name"]] = dataset
            for declaration in ln.findall("ReportControl"):
                enabled = declaration.find("RptEnabled")
                count = int(enabled.get("max", "1")) if enabled is not None else 1
                if declaration.get("indexed") == "false":
                    count = 1
                for _ in range(count):
                    report = next(report_iter)
                    dataset = local.get(report["dataset"])
                    report.update(ref=owner + "." + ("BR." if report["buffered"] else "RP.") + report["name"],
                                  dataset_ref=dataset["ref"] if dataset else None,
                                  signal_count=len(dataset["signal_refs"]) if dataset else 0,
                                  unresolved=dataset["unresolved"] if dataset else [])


def report_members(ied, reference):
    report = next((item for item in ied["reports"] if item["ref"] == reference), None)
    if report is None:
        raise StudioError("Unknown report reference")
    dataset = next((item for item in ied["datasets"] if item["ref"] == report["dataset_ref"]), None)
    return set(dataset["signal_refs"]) if dataset else set()
