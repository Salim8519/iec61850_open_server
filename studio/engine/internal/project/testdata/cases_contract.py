"""Pydantic contract vectors: Diagram.model_validate lax coercion, limits and error lists."""

BIG = 10 ** 30
FLOAT_INPUTS = [0, -5, 1.5, True, False, "1.5", " 2 ", "1_0", "1__0", "_1", "+_1", "in_f", "inf", "nan",
                "-Infinity", "1e400", "1e-400", "abc", "", "  ", "0x10", "1.e5", ".5", "5.", ".", "1e", "-.5",
                " 1", " 1", "\x1c1", "1_.5", "1._5", "1._", "1e_1", float("inf"), float("nan"),
                BIG, 10 ** 400, 2 ** 64, None, [1], {"a": 1}, 100000, 100000.0001, -100000, -100001,
                99999.99999999999, 5e-324, -0.0]
INT_INPUTS = [0, 7, -3, True, False, 1.0, 1.5, 2.0 ** 70, -9.223372036854775808e18, 9.2e18, "7", " 7 ",
              "7.0", "7.", "7.00", ".0", "-.0", "1_0.0", "7.0.0", "1_000", "1__0", "_1", "1_", "+_1",
              "+1_0", "-1_0", "07", "-00", "+07", "٣", "0x1", "1e3", "1" * 30, "-" + "1" * 30,
              "1" * 4300, "1" * 4301, " " * 3 + "5" + "　", "", "+", float("inf"), float("nan"),
              BIG, None, [], {}, "abc"]
BOOL_INPUTS = [True, False, 0, 1, 2, -1, 0.0, 1.0, 2.0, 0.5, float("inf"), BIG, 2 ** 63, "true", "TRUE",
               "TrUe", "yes", "on", "1", "t", "y", "false", "F", "No", "off", "0", "n", "", " true",
               "True ", "maybe", "0.0", None, [], {}]
TEXT_INPUTS = ["", "x", "x" * 100, "x" * 101, "\U0001F600" * 100, "\U0001F600" * 101, 5, None, True,
               ["a"], {"a": 1}, "é" * 64]
ID_INPUTS = ["n1", "a-b_C9", "x" * 64, "x" * 65, "", "a b", "abc\n", "é", 5, None, "--"]
TAGS = [None, True, 1, 1.5, "", "x", "Direct", ["x"], {"a": 1}, {"a": "b'c"}, "it's", 'q"', "é", "\n"]


def node(node_id="n1", **data):
    return {"id": node_id, "position": {"x": 1, "y": 2}, "data": {"label": "L", "kind": "CB", **data}}


def repeat(value, count):
    """A list of count copies, expanded by the generator and the Go test."""
    return {"@repeat": value, "count": count}


def binding(**extra):
    return {"ied_id": "i", "ref": "r", **extra}


def diagram(*nodes, edges=None):
    result = {"nodes": list(nodes)}
    if edges is not None:
        result["edges"] = edges
    return result


def cases(fixtures):
    out = []

    def add(name, value):
        out.append({"name": name, "fn": "validate_diagram", "args": {"diagram": value}, "ieds": []})

    for index, value in enumerate([None, [], "x", 5, {}, {"nodes": []}, {"edges": []}, {"nodes": None},
                                   {"nodes": {"a": 1}, "edges": "x"}, {"extra": 1}, {"nodes": [], "edges": [], "z": 1}]):
        add(f"top {index}", value)
    for index, value in enumerate(FLOAT_INPUTS):
        add(f"float x {index}", diagram({"id": "n1", "position": {"x": value, "y": 0}, "data": {"label": "L", "kind": "CB"}}))
    for index, value in enumerate(INT_INPUTS):
        add(f"int rotation {index}", diagram(node(rotation=value)))
    for index, value in enumerate([99, 100, 60000, 60001, "100", 100.0, True, BIG, -BIG]):
        add(f"int push_reset_ms {index}", diagram(node(push_reset_ms=value)))
    for index, value in enumerate(BOOL_INPUTS):
        add(f"bool push_value {index}", diagram(node(push_value=value)))
        add(f"bool push_rest_value {index}", diagram(node(push_rest_value=value)))
    for index, value in enumerate(TEXT_INPUTS):
        add(f"text label {index}", diagram(node(label=value)))
    for index, value in enumerate(ID_INPUTS):
        add(f"node id {index}", diagram({"id": value, "position": {"x": 0, "y": 0}, "data": {"label": "L", "kind": "CB"}}))
    literals = {"kind": ["CB", "INFEED", "TRANSFORMER", "cb", 5, None],
                "trip_command_polarity": ["operate_true", "x", None],
                "trip_state_polarity": ["operated_false", 1],
                "push_mode": ["latching_status", "self_resetting", 1],
                "initial_position": ["closed", None, "Open"],
                "infeed_type": [None, "grid", "x"], "infeed_state": [None, "dead", 3],
                "active_color": [None, "#aabbcc", "#AABBCZ", "#aabbcc\n", 5, "aabbcc"],
                "inactive_color": ["#000000", ""]}
    for key, values in literals.items():
        for index, value in enumerate(values):
            add(f"field {key} {index}", diagram(node(**{key: value})))
    for index, tag in enumerate(TAGS):
        add(f"tag {index}", diagram(node(state_mapping={"mode": tag, "source": binding()})))
    mappings = [None, False, [1], "direct", 0, {}, {"source": binding()}, {"mode": "direct"},
                {"mode": "direct", "source": binding(report_ref=None)},
                {"mode": "direct", "source": binding(report_ref=5)},
                {"mode": "direct", "source": {**binding(), "zz": 1, "aa": 2}, "extra": 1},
                {"mode": "single_to_dbpos", "source": 5},
                {"mode": "contacts_to_dbpos", "open": binding(), "closed": 5},
                {"mode": "contacts_to_dbpos", "open": binding(), "closed": binding(ref="s")},
                {"mode": "contacts_to_dbpos", "source": binding()},
                {"source": binding(), "mode": "direct", "x": None}]
    for index, value in enumerate(mappings):
        add(f"state_mapping {index}", diagram(node(state_mapping=value)))
    controls = [None, {}, [], {"ied_id": 1, "ref": None, "x": 2}, binding(), binding(report_ref="x"), "x"]
    for index, value in enumerate(controls):
        add(f"control_binding {index}", diagram(node(control_binding=value)))
    relay = {"id": "a", "label": "b", "source": binding()}
    relays = [None, {"a": 1}, [], [relay], [relay] * 50, [relay] * 51, [{"id": "a b", "label": "", "source": 1}] * 52,
              [{"id": "a b", "label": "", "source": 1}, {"id": 5, "label": "x" * 101, "source": binding(), "q": 1}],
              [{"label": "x"}], [5, None]]
    for index, value in enumerate(relays):
        add(f"relay_statuses {index}", diagram(node(kind="RELAY", relay_statuses=value)))
    add("data extra", diagram(node(zeta=1, alpha=2, rotation="x")))
    add("data empty", diagram({"id": "n1", "position": {"x": 1, "y": 2}, "data": {}}))
    add("node empty", diagram({}))
    add("node types", diagram(5, "x", None, []))
    add("node extra ignored", diagram({"id": "n1", "type": "equipment", "position": {"x": 1, "y": 2, "z": 3},
                                       "data": {"label": "L", "kind": "CB"}, "foo": 1}))
    add("node bad type", diagram({"id": "n1", "type": "x", "position": [], "data": None}))
    add("position missing", diagram({"id": "n1", "position": {}, "data": {"label": "L", "kind": "CB"}}))
    add("many nodes", {"nodes": repeat(node(), 2000)})
    add("too many nodes", {"nodes": repeat(node(), 2001)})
    add("too many bad nodes", {"nodes": repeat({}, 2002)})
    add("many edges", {"edges": repeat({"id": "e", "source": "a", "target": "b"}, 5000)})
    add("too many edges", {"edges": repeat({"id": "e", "source": "a", "target": "b"}, 5001)})
    wires = [{"id": "e" * 129, "source": 1, "target": None, "sourceHandle": 5, "targetHandle": "t", "x": 1}, {},
             {"targetHandle": "t", "target": "b", "source": "a", "id": "e" * 128}, 5]
    add("wires", diagram(edges=wires))
    add("mixed errors", {"nodes": [node(rotation=1.5, push_value="x"), {}], "edges": [{}]})
    return out
