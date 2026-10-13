"""Build configuration nodes and signal metadata from SCL type declarations."""
from studio_app.contracts.errors import StudioError
from .type_resolver import TYPES, FCS, identifier, initial_value, config_initial, trigger_bits


class ModelBuilder:
    def __init__(self, declarations):
        self.types = declarations
        self.signals = []
        self.by_reference = {}
        self.controls = []
        self.control_warnings = []
        self.lines = []
        self.active = set()

    def declaration(self, tag, type_id):
        result = self.types.get((tag, type_id))
        if result is None:
            raise StudioError(f"Unresolved {tag} {type_id}")
        return result

    def data_object(self, element, reference, instance=None, depth=0):
        if depth > 32:
            raise StudioError("SCL type nesting exceeds 32 levels")
        name = identifier(element.get("name"), "data object")
        count = int(element.get("count", "0"))
        if not 0 <= count <= 256:
            raise StudioError("Array element count must be between 0 and 256")
        ref = reference + "." + name
        definition = self.declaration("DOType", element.get("type"))
        self.lines.append(f"DO({name} {count}){{")
        if count:
            for index in range(count):
                self.lines.append(f"[{index}]{{")
                self.object_content(definition, f"{ref}({index})", instance, depth, element.get("desc", ""))
                self.lines.append("}")
        else:
            self.object_content(definition, ref, instance, depth, element.get("desc", ""))
        self.lines.append("}")

    def object_content(self, definition, ref, instance, depth, description=""):
        description = (instance.get("desc") if instance is not None else None) or description or definition.get("desc", "")
        for child in definition:
            override = self._override(instance, child.get("name"))
            if child.tag == "SDO":
                self.data_object(child, ref, override, depth + 1)
            elif child.tag == "DA":
                self.attribute(child, ref, override, None, depth + 1, description=description)
        model = self.by_reference.get(ref + ".ctlModel")
        feedback = self.by_reference.get(ref + ".stVal")
        if model and model["value"] and feedback and feedback["kind"] in {"BOOLEAN", "Dbpos"}:
            ctl_val = self.by_reference.get(ref + ".Oper.ctlVal")
            cdc = definition.get("cdc")
            expected = "BOOLEAN" if cdc in {"SPC", "DPC"} else None
            feedback_ok = ((cdc == "SPC" and feedback["kind"] == "BOOLEAN")
                           or (cdc == "DPC" and feedback["kind"] == "Dbpos"))
            if expected and feedback_ok and ctl_val and ctl_val["kind"] == expected:
                timeout = self.by_reference.get(ref + ".sboTimeout")
                self.controls.append({"ref": ref, "status": feedback["ref"], "cdc": cdc,
                                      "ctl_val_kind": ctl_val["kind"], "model": model["value"],
                                      "sbo_timeout_ms": timeout["value"] if timeout else 0,
                                      "delay_ms": 0, "blocked": False})
            else:
                self.control_warnings.append(
                    f"Control {ref} is not exposed: expected a consistent SPC/DPC Boolean control model")

    @staticmethod
    def _override(parent, name):
        if parent is None:
            return None
        return next((child for child in parent if child.get("name") == name), None)

    def attribute(self, element, reference, instance, inherited_fc, depth, inherited_triggers=0, description=""):
        if depth > 32:
            raise StudioError("SCL attribute nesting exceeds 32 levels")
        name = identifier(element.get("name"), "data attribute")
        kind = element.get("bType")
        if kind not in TYPES:
            raise StudioError(f"Unsupported basic type {kind} at {reference}.{name}")
        fc = element.get("fc", inherited_fc)
        if fc not in FCS:
            raise StudioError(f"Unknown functional constraint {fc}")
        count = int(element.get("count", "0"))
        if count:
            raise StudioError(f"Array attribute {reference}.{name} needs array-attribute support")
        ref = reference + "." + name
        description = (instance.get("desc") if instance is not None else None) or element.get("desc") or description
        triggers = trigger_bits(element) | inherited_triggers
        header = f"DA({name} 0 {TYPES[kind]} {FCS[fc]} {triggers} 0)"
        if kind == "Struct":
            self.lines.append(header + "{")
            definition = self.declaration("DAType", element.get("type"))
            for child in definition.findall("BDA"):
                self.attribute(child, ref, self._override(instance, child.get("name")),
                               fc, depth + 1, triggers, description)
            self.lines.append("}")
            return
        enums = {}
        if kind == "Enum":
            definition = self.declaration("EnumType", element.get("type"))
            enums = {item.text.strip(): int(item.get("ord")) for item in definition.findall("EnumVal")
                     if item.text and item.text.strip()}
        text = instance.findtext("Val") if instance is not None else None
        if text is None:
            text = element.findtext("Val")
        value = initial_value(kind, text, enums)
        self.lines.append(header + config_initial(kind, value) + ";")
        self.signals.append({"ref": ref, "fc": fc, "kind": kind, "value": value,
                             "description": description, "enums": enums,
                             "writable": fc in {"ST", "MX", "SP", "CF", "DC"},
                             "quality": 0, "timestamp": 0})
        if ref in self.by_reference:
            raise StudioError(f"Ambiguous duplicate data reference {ref}")
        self.by_reference[ref] = self.signals[-1]
