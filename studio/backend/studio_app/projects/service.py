"""Engineering project ownership, import, persistence, and public snapshots."""
import copy
import os
import re
import threading
from uuid import uuid4
from .archive import encode, decode
from .document import CURRENT_FORMAT
from .validation import validate_diagram, validate_sources
from .mapping_migration import migrate_diagram
from .migration_backup import preserve as preserve_migration
from .removal import impact
from . import bulk_removal
from .test_panel import validate_panel
from .stored_import import import_stored
from .goose_reach import LAB_NOTICE, LAB_TITLE, lab_notice
from studio_app.contracts.errors import StudioError
from studio_app.domain.scenario import validate_scenarios
from studio_app.domain.source import source_summary
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template
from studio_app.networking.loopback import assign_endpoints
from studio_app.domain.network_settings import DEFAULT_NETWORK, validate_ied_settings, validate_network
from studio_app.domain.gateway_rules import empty_gateway, normalize_gateway


class ProjectService:
    def __init__(self, data_directory, on_change=None):
        self.directory = data_directory
        self.directory.mkdir(parents=True, exist_ok=True)
        self.lock = threading.RLock()
        self.session_id = uuid4().hex
        self.revision = 0
        self.on_change = on_change or (lambda revision, source: None)
        self.project = {"id": uuid4().hex, "name": "Untitled substation", "ieds": [],
                        "diagram": {"nodes": [], "edges": []}, "sources": [], "scenarios": [], "test_panel": [],
                        "network": dict(DEFAULT_NETWORK), "gateway": empty_gateway()}
        self.migration_notice = None

    def changed(self, source=None):
        self.revision += 1
        self.on_change(self.revision, source)

    def add_scl(self, source, filename, selected_names=None):
        models = import_scl(source, filename, selected_names)
        with self.lock:
            names = {ied["name"] for ied in self.project["ieds"]}
            if any(model["name"] in names for model in models):
                raise StudioError("An IED with this name already exists; rename before importing")
            if len(models) + len(names) > 20:
                raise StudioError("This release supports up to 20 IEDs per project")
            for model in models:
                model.update(validate_ied_settings({}))
            self.project["ieds"].extend(models)
            assign_endpoints(self.project["ieds"])
            self.save()
            self.changed()
        return models

    def add_template(self, name):
        return self.add_scl(make_template(name), name + ".cid")

    def find_ied(self, key):
        with self.lock:
            ied = next((item for item in self.project["ieds"] if item["id"] == key), None)
            if ied is None:
                raise StudioError("Unknown IED", code="not_found")
            return ied

    def find_node(self, key):
        with self.lock:
            node = next((item for item in self.project["diagram"]["nodes"] if item["id"] == key), None)
            if node is None:
                raise StudioError("Unknown equipment", code="not_found")
            return node

    def removal_preview(self, key):
        with self.lock:
            ied = self.find_ied(key)
            return {"id": key, "name": ied["name"], **impact(self.project, key)}

    def remove_ied(self, key):
        with self.lock:
            preview = self.removal_preview(key)
            token = bulk_removal.preview(self, [key])["fingerprint"]
            result = bulk_removal.remove(self, [key], token)
            return {**preview, "recovery_file": result["recovery_file"]}

    def diagram(self, value, expected_revision=None, change_source=None):
        with self.lock:
            if expected_revision is not None and expected_revision != self.revision:
                raise StudioError("Another tab changed this project. This tab must synchronize before saving.",
                                  code="conflict")
            self.project["diagram"] = validate_diagram(value, self.project["ieds"])
            self.save()
            self.changed(change_source)

    def node_data(self, node_id, changes):
        with self.lock:
            node = self.find_node(node_id)
            updated = {**node, "data": {**node["data"], **changes}}
            nodes = [updated if item["id"] == node_id else item
                     for item in self.project["diagram"]["nodes"]]
            self.project["diagram"] = validate_diagram({"nodes": nodes, "edges": self.project["diagram"]["edges"]},
                                                       self.project["ieds"])
            self.save()

    def source(self, ied_id, reference, spec):
        with self.lock:
            sources = [item for item in self.project["sources"]
                       if (item["ied_id"], item["ref"]) != (ied_id, reference)]
            sources.append({"ied_id": ied_id, "ref": reference, "spec": spec})
            self.project["sources"] = validate_sources(sources, self.project["ieds"])
            self.save()
            self.changed()

    def snapshot(self):
        with self.lock:
            hidden = {"ieds", "sources", "gateway"}  # the gateway point list has its own API
            result = {key: copy.deepcopy(value) for key, value in self.project.items() if key not in hidden}
            result["sources"] = [{**item, "spec": source_summary(item["spec"])} for item in self.project["sources"]]
            result["ieds"] = [{key: copy.deepcopy(value) for key, value in ied.items()
                               if key not in {"source", "cfg", "signals"}}
                              for ied in self.project["ieds"]]
            for summary, original in zip(result["ieds"], self.project["ieds"]):
                summary["signal_count"] = len(original["signals"])
            if self.migration_notice:
                result["migration_notice"] = copy.deepcopy(self.migration_notice)
            result["session_id"] = self.session_id
            result["revision"] = self.revision
            return result

    def scenarios(self, value):
        with self.lock:
            self.project["scenarios"] = validate_scenarios(value, self.project["ieds"])
            self.save()
            self.changed()
            self.save()

    def export(self):
        with self.lock:
            return encode(self.project)

    def test_panel(self, entries):
        with self.lock:
            candidate = validate_panel(entries, self.project["ieds"])
            previous = self.project["test_panel"]
            self.project["test_panel"] = candidate
            try:
                self.save()
                self.changed()
            except Exception:
                self.project["test_panel"] = previous
                raise

    def gateway(self, config):
        with self.lock:
            previous = self.project.get("gateway")
            self.project["gateway"] = config
            try:
                self.save()
                self.changed("gateway")
            except Exception:
                self.project["gateway"] = previous
                raise

    def save(self):
        with self.lock:
            payload = self.export()
            destination = self.directory / "autosave.subsim"
            temporary = destination.with_suffix(".tmp")
            temporary.write_bytes(payload)
            os.replace(temporary, destination)

    def restore(self, payload, persist=True):
        raw = decode(payload)
        version = int(raw.get("format", 1))
        if not isinstance(raw.get("ieds"), list) or len(raw["ieds"]) > 20:
            raise StudioError("Project must contain a list of at most 20 IEDs")
        imported = []
        built = import_stored(raw["ieds"])
        for item in raw["ieds"]:
            model = built[item["id"]]
            model.update(id=item["id"], ip=item.get("ip", ""), port=int(item.get("port", 102)))
            try:
                model.update(validate_ied_settings(item))
            except ValueError as error:
                raise StudioError(f"Stored settings for IED {item['name']}: {error}") from error
            imported.append(model)
        if (len({item["name"] for item in imported}) != len(imported)
                or len({item["id"] for item in imported}) != len(imported)):
            raise StudioError("Project contains too many or duplicate IEDs")
        migrated, notices = migrate_diagram(raw.get("diagram", {}), imported, version)
        legacy = version < CURRENT_FORMAT
        recovery = preserve_migration(self.directory, payload) if legacy else None
        diagram = validate_diagram(migrated, imported)
        sources = validate_sources(raw.get("sources", []), imported)
        scenarios = validate_scenarios(raw.get("scenarios", []), imported)
        panel = validate_panel(raw.get("test_panel", []), imported)
        try:
            network = validate_network(raw.get("network"))
        except ValueError as error:
            raise StudioError(f"Stored network settings: {error}") from error
        try:
            gateway = normalize_gateway(raw.get("gateway"))
        except ValueError as error:
            raise StudioError(f"Stored gateway settings: {error}") from error
        project_id = raw.get("id", uuid4().hex)
        if not isinstance(project_id, str) or not re.fullmatch(r"[0-9a-f]{32}", project_id):
            raise StudioError("Invalid stored project identifier")
        with self.lock:
            previous = self.project
            previous_notice = self.migration_notice
            self.project = {"id": project_id, "name": str(raw.get("name", "Imported substation"))[:100],
                            "ieds": imported, "diagram": diagram,
                            "sources": sources, "scenarios": scenarios, "test_panel": panel,
                            "network": network, "gateway": gateway}
            details = notices or (["No equipment was changed."] if version >= 11 else
                                  ["Existing controls remain in standard IEC 61850 mode."])
            standard = " Every command follows standard IEC 61850."
            if version == 14:
                message = f"Project upgraded to format {CURRENT_FORMAT}." + standard
            elif version == 13:
                message = (f"Project upgraded to format {CURRENT_FORMAT}: the gateway tool can now also act as "
                           "SCADA master." + standard)
            elif version >= 11:
                message = (f"Project upgraded to format {CURRENT_FORMAT}. "
                           + ("See the details for what changed." if notices else
                              "Nothing in your diagram changed."))
            elif version >= 10:
                message = (f"Project upgraded to format {CURRENT_FORMAT}: network settings were added. "
                           "Every IED is still simulated on this PC only, exactly as before.")
            elif version >= 9:
                message = (f"Project upgraded to format {CURRENT_FORMAT}: "
                           "push buttons no longer accept wires.")
            else:
                message = f"Project upgraded to format {CURRENT_FORMAT}." + standard
            goose_note = lab_notice(raw.get("network"), network)
            self.migration_notice = ({
                "message": message,
                "details": [*details, *([LAB_NOTICE] if goose_note else [])], "backup": recovery,
            } if legacy else {"message": LAB_TITLE, "details": [LAB_NOTICE], "backup": ""} if goose_note else None)
            try:
                assign_endpoints(imported)
                if persist:
                    self.save()
                self.changed()
            except Exception:
                self.project = previous
                self.migration_notice = previous_notice
                raise
