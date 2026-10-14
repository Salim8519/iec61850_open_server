"""Safely import SCL while retaining the original document for preservation."""
from uuid import uuid4
from .document import parse_document
from studio_app.contracts.errors import StudioError
from .model_builder import ModelBuilder
from .type_resolver import identifier
from .reports import compile_reports
from .report_catalogue import enrich_reports
from studio_app.domain.sample import companions
from .schema_validation import validate_schema
from .communication import read_communication
from .goose_controls import goose_controls, goose_cfg_lines
from .sv_controls import sv_controls
from .goose_links import goose_subscriptions, publisher_catalogue

def import_scl(source, filename="import.scd", selected_names=None):
    if isinstance(source, str):
        source = source.encode("utf-8")
    root = parse_document(source)
    validation = validate_schema(source)
    if selected_names is not None:
        if (not isinstance(selected_names, list) or not selected_names
                or any(not isinstance(name, str) for name in selected_names)
                or len(selected_names) != len(set(selected_names))):
            raise StudioError("Select one or more unique IED names")
        available = {ied.get("name") for ied in root.findall("IED")}
        if not set(selected_names) <= available:
            raise StudioError("Selected IED is not present in this file")
    declarations = root.find("DataTypeTemplates")
    if declarations is None:
        raise StudioError("Missing DataTypeTemplates")
    types = {}
    for declaration in declarations:
        key = declaration.tag, declaration.get("id")
        if key in types:
            raise StudioError(f"Duplicate type declaration {key[1]}")
        types[key] = declaration
    result = []
    names = set()
    catalogue = None
    for ied in root.findall("IED"):
        name = identifier(ied.get("name"), "IED name")
        if selected_names is not None and name not in selected_names:
            continue
        if name in names:
            raise StudioError(f"Duplicate IED name {name}")
        names.add(name)
        servers = ied.findall("AccessPoint/Server")
        if not servers:
            if selected_names is not None:
                raise StudioError(f"IED {name} has no server model")
            continue
        if len(servers) != 1:
            raise StudioError(f"IED {name} has multiple servers; select an access point first")
        builder = ModelBuilder(types)
        builder.lines.append(f"MODEL({name}){{")
        datasets, reports, publishes, streams = [], [], [], []
        communication, warnings = read_communication(root, name)
        for ld in servers[0].findall("LDevice"):
            inst = identifier(ld.get("inst"), "logical device")
            ld_name = ld.get("ldName")
            if ld_name:
                identifier(ld_name, "functional logical device name")
            domain = ld_name or name + inst
            builder.lines.append(f"LD({inst}{' ' + ld_name if ld_name else ''}){{")
            for ln in list(ld):
                if ln.tag not in {"LN", "LN0"}:
                    continue
                ln_name = "LLN0" if ln.tag == "LN0" else (
                    ln.get("prefix", "") + ln.get("lnClass", "") + ln.get("inst", ""))
                identifier(ln_name, "logical node")
                definition = builder.declaration("LNodeType", ln.get("lnType"))
                builder.lines.append(f"LN({ln_name}){{")
                for obj in definition.findall("DO"):
                    instance = next((x for x in ln.findall("DOI") if x.get("name") == obj.get("name")), None)
                    builder.data_object(obj, domain + "/" + ln_name, instance)
                ds, rs = compile_reports(ln, inst, builder.lines, warnings)
                datasets.extend(ds)
                reports.extend(rs)
                if ln.tag == "LN0":
                    controls = goose_controls(ln, domain, inst, communication["gse"], ds, name, warnings)
                    for control in controls:
                        builder.lines.extend(goose_cfg_lines(control))
                    publishes.extend(controls)
                    streams.extend(sv_controls(ln, domain, inst, communication["smv"], name, warnings))
                builder.lines.append("}")
            builder.lines.append("}")
        builder.lines.append("}")
        enrich_reports(servers[0], name, datasets, reports, builder.signals)
        for signal in builder.signals:
            related = companions(signal, builder.by_reference)
            signal.update(quality_ref=related.get("q", {}).get("ref"),
                          timestamp_ref=related.get("t", {}).get("ref"),
                          quality=related.get("q", {}).get("value", 0),
                          timestamp=related.get("t", {}).get("value", 0))
        warnings.extend(builder.control_warnings)
        catalogue = publisher_catalogue(root) if catalogue is None else catalogue
        subscribes, diagnostics = goose_subscriptions(root, ied, catalogue)
        if validation["status"] != "valid":
            warnings.append(f"Source document XSD validation: {validation['status']}. "
                            + validation["errors"][0]["message"])
        result.append({"id": uuid4().hex, "name": name, "filename": filename,
                       "source": source.decode("utf-8-sig"), "cfg": "\n".join(builder.lines) + "\n",
                       "signals": builder.signals, "controls": builder.controls,
                       "datasets": datasets, "reports": reports, "warnings": warnings,
                       "validation": validation, "scl_address": communication["address"],
                       "goose": {"publishes": publishes, "subscribes": subscribes, "diagnostics": diagnostics},
                       "sv": {"publishes": streams},
                       "edition": 1 if root.get("version") is None else 2,
                       "ip": "", "port": 102, "state": "stopped"})
    if not result:
        raise StudioError("No IED servers found in SCL")
    return result
