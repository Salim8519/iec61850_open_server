"""Health rows for a connected real IED: data model, datasets, report blocks and mode."""
from .goose_consistency import result, sentence
from .health_reads import mode_items, mode_name


def model_rows(ied, comparison, when):
    summary = comparison["summary"]
    missing, mismatched, extra = (summary["missing_on_device"], summary["type_mismatch"],
                                  summary["extra_on_device"])
    if missing or mismatched:
        model = result("model", "Data points", "problem", f"{missing} SCD points are missing on the device and "
                       f"{mismatched} have another data type.", when, "mms_read")
    elif extra:
        model = result("model", "Data points", "warning", f"The device has {extra} points the SCD does not "
                       "describe. The SCD may be older than the device configuration.", when, "mms_read")
    else:
        model = result("model", "Data points", "ok", "Every SCD data point exists on the device with the same type.",
                       when, "mms_read")
    rows = [model]
    for item in comparison["datasets"]:
        if item["status"] == "match":
            continue
        message = ("This dataset is missing on the device." if item["status"] == "missing_on_device" else
                   f"Members differ: {len(item['missing'])} missing and {len(item['extra'])} extra on the device.")
        rows.append(result("datasets", item["ref"], "problem", message, when, "mms_read"))
    if not any(row["area"] == "datasets" for row in rows):
        rows.append(result("datasets", "Datasets", "ok", "Every SCD dataset matches the device.", when, "mms_read"))
    for item in comparison["reports"]:
        if item["status"] != "match":
            rows.append(result("reports", item["ref"], "problem", "This report block is missing on the device.",
                               when, "mms_read"))
    return rows


def report_setting_rows(ied, rcbs):
    if rcbs is None:
        return [result("report_settings", "Report blocks", "not_checked",
                       "Polling access never reads report blocks, so their settings were not compared.")]
    rows, when = [], rcbs["checked_at"]
    for report in ied.get("reports", []):
        block = rcbs["blocks"].get(report["ref"])
        if not block or not block.get("readable"):
            continue
        differences = []
        expected = report.get("dataset_ref") or ""
        if str(block.get("dataset", "")).replace(".", "$") != expected.replace(".", "$"):
            differences.append(f"dataset is {block.get('dataset') or '(none)'} on the device but {expected or '(none)'} "
                               "in the SCD")
        if "conf_rev" in report and block.get("conf_rev") != report["conf_rev"]:
            differences.append(f"confRev is {block.get('conf_rev')} on the device but {report['conf_rev']} in the SCD")
        if differences:
            rows.append(result("report_settings", report["ref"], "problem", sentence(differences),
                               when, "mms_read", {"device": block, "scd": {"dataset": expected,
                                                                        "conf_rev": report.get("conf_rev")}}))
        else:
            rows.append(result("report_settings", report["ref"], "ok", "Dataset and confRev match the SCD.",
                               when, "mms_read"))
    return rows or [result("report_settings", "Report blocks", "ok", "No report blocks to compare.", when, "mms_read")]


def mode_rows(ied, mode):
    wanted = [item["ref"] for item in mode_items(ied)]
    if not wanted:
        return [result("mode", "Mode", "not_checked", "Mode, behaviour and simulation are not available in this IED.")]
    rows, when = [], mode["checked_at"]
    for ref in wanted:
        if ref in mode["errors"]:
            rows.append(result("mode", ref, "warning", "Could not be read from the device.", when, "mms_read"))
            continue
        value = mode["values"].get(ref)
        if ref.endswith(".Sim.stVal"):
            verdict = "warning" if value is True else "ok"
            message = ("The IED is in simulation mode: it listens to test GOOSE/SV instead of the real ones."
                       if value is True else "Not in simulation mode.")
        else:
            name = mode_name(value)
            verdict = "ok" if name == "on" else "warning"
            message = f"{name[:1].upper() + name[1:]}." if name else f"The device reports {value}, which is not a valid mode (1 to 5)."
            if name and name != "on":
                message = f"This logical device is {name}, not in normal service."
        rows.append(result("mode", ref, verdict, message, when, "mms_read", {"value": value}))
    return rows
