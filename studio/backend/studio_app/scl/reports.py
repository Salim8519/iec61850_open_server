"""Translate datasets and report control blocks into native configuration."""
from studio_app.contracts.errors import StudioError
from .type_resolver import identifier


def bits(element, pairs):
    if element is None:
        return 0
    return sum(bit for key, bit in pairs if element.get(key) == "true")


def usable_reports(ln):
    """ReportControl elements with a name; unnamed ones cannot be served or addressed."""
    return [report for report in ln.findall("ReportControl") if (report.get("name") or "").strip()]


def compile_reports(ln, ld_inst, lines, warnings=None):
    datasets = []
    reports = []
    for dataset in ln.findall("DataSet"):
        name = identifier(dataset.get("name"), "dataset")
        lines.append(f"DS({name}){{")
        members = []
        for item in dataset.findall("FCDA"):
            member_ld = identifier(item.get("ldInst", ld_inst), "dataset logical device")
            node = item.get("prefix", "") + item.get("lnClass", "") + item.get("lnInst", "")
            identifier(node, "dataset logical node")
            fc = item.get("fc", "ST")
            object_path = item.get("doName", "")
            attribute = item.get("daName", "")
            parts = object_path.split(".") + (attribute.split(".") if attribute else [])
            for part in parts:
                identifier(part, "dataset member")
            indexed = item.get("ix") is not None
            member_parts = object_path.split(".") if indexed else parts
            member = "$".join([node, fc] + member_parts)
            if member_ld != ld_inst:
                member = member_ld + "/" + member
            index = " " + str(int(item.get("ix"))) if indexed else ""
            if indexed and int(item.get("ix")) < 0:
                raise StudioError("Dataset array index must be non-negative")
            component = " " + attribute.replace(".", "$") if indexed and attribute else ""
            lines.append(f"DE({member}{index}{component});")
            members.append(member + index + component)
        lines.append("}")
        datasets.append({"name": name, "members": members})
    skipped = len(ln.findall("ReportControl")) - len(usable_reports(ln))
    if skipped and warnings is not None:
        warnings.append(f"{skipped} report block(s) in {ld_inst}/{ln.get('lnClass', 'LLN0')} have no name in the "
                        "file and were skipped. Ask the IED engineer to fix the CID.")
    for report in usable_reports(ln):
        name = identifier(report.get("name"), "report control")
        ds = identifier(report.get("datSet"), "report dataset") if report.get("datSet") else "-"
        if ds != "-" and ds not in {item["name"] for item in datasets}:
            raise StudioError(f"Report {name} refers to unresolved dataset {ds}")
        buffered = report.get("buffered") == "true"
        triggers = bits(report.find("TrgOps"), [("dchg", 1), ("qchg", 2),
                        ("dupd", 4), ("period", 8), ("gi", 16)])
        options = bits(report.find("OptFields"), [("seqNum", 1), ("timeStamp", 2),
                       ("reasonCode", 4), ("dataSet", 8), ("dataRef", 16),
                       ("bufOvfl", 32), ("entryID", 64), ("configRef", 128)])
        enabled = report.find("RptEnabled")
        count = int(enabled.get("max", "1")) if enabled is not None else 1
        if not 1 <= count <= 64:
            raise StudioError(f"Invalid report client count for {name}")
        for index in range(count):
            final_name = name + f"{index + 1:02d}" if report.get("indexed", "true") == "true" else name
            rptid = report.get("rptID", final_name) or final_name
            if any(char.isspace() or char in "();{}" for char in rptid):
                raise StudioError(f"Unsupported report ID syntax: {rptid}")
            lines.append(f"RC({final_name} {rptid} {int(buffered)} {ds} "
                         f"{int(report.get('confRev', '1'))} {triggers} {options} "
                         f"{int(report.get('bufTime', '0'))} {int(report.get('intgPd', '1000'))});")
            reports.append({"name": final_name, "dataset": ds, "buffered": buffered,
                            "conf_rev": int(report.get("confRev", "1"))})
            if report.get("indexed") == "false":
                break
    return datasets, reports
