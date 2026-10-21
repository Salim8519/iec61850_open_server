"""validate_scenarios and validate_panel vectors."""

B = "1" * 32
P = "BayControllerCTRL/"
POS, LOC, ANIN, CF = P + "XCBR1.Pos.stVal", P + "XCBR1.Loc.stVal", P + "GGIO1.AnIn1.mag.f", P + "XCBR1.Pos.ctlModel"
NAME = P + "LLN0.NamPlt.vendor"
ALL = ["BayController", "Bay2", "FEED1"]


def step(step_id="s1", at=0, ref=LOC, value=True, **extra):
    return {"id": step_id, "at": at, "ied_id": B, "ref": ref, "value": value, **extra}


def scenario(*steps, **extra):
    return {"id": "sc", "name": "Test", "steps": list(steps), **extra}


def entry(**extra):
    return {"id": "c1", "label": "Close", "ied_id": B, "ref": LOC, **extra}


def cases(fixtures):
    out = []

    def add(name, value):
        out.append({"name": name, "fn": "validate_scenarios", "args": {"scenarios": value}, "ieds": ALL})

    def panel(name, value):
        out.append({"name": name, "fn": "validate_panel", "args": {"entries": value}, "ieds": ALL})

    for index, value in enumerate([None, {}, [5], [scenario(step())] * 2, [{"steps": [step()]}],
                                   [scenario(step(), id="a b")], [scenario(step(), id=5)], [scenario(step(), name="  ")],
                                   [scenario(step(), name=" x ")], [scenario(step(), name="x" * 101)],
                                   [scenario(step(), name=5)], [scenario(step(), enabled=1)],
                                   [scenario(step(), enabled=True)], [scenario()], [{"id": "sc", "name": "x"}],
                                   [scenario(*[step(f"s{i}") for i in range(1001)])], [scenario(5)],
                                   [scenario(step(), step())], [scenario(step(step_id="a b"))], [scenario(step(step_id=None))],
                                   list(range(101))]):
        add(f"scenario shape {index}", value)
    for index, at in enumerate([True, None, "1", -1, 86400, 86401, 86400.5, float("inf"), float("nan"), 10 ** 400,
                                10 ** 20, 5, 0.25]):
        add(f"step at {index}", [scenario(step(at=at))])
    steps = [step(ied_id=5), step(ref=None), step(ref="zz"), step(action="toggle"), step(action=["set"]),
             step(ref="zz", action=["set"]), step(ref=CF, value=3), step(ref=CF, value=3, action="assert"),
             step(value=1), step(ref=POS, value=4), step(ref=POS, value=2), step(ref=ANIN, value=1),
             step(ref=ANIN, value=1e39), step(ref=ANIN, value="1"), step(ref=NAME, value="x" * 256),
             step(ref=NAME, value="é" * 128), step(ref=P + "LLN0.Health.stVal", value=2 ** 31),
             step(ref=P + "XCBR1.Pos.q", value=8192), step(ref=P + "XCBR1.Pos.t", value=4294967295999)]
    for index, value in enumerate(steps):
        add(f"step {index}", [scenario(value)])
    add("sorted steps", [scenario(step("a", at=5), step("b", at=1.5), step("c", at=5), step("d", at=0),
                                  step("e", at=1, ref=ANIN, value=2.5, action="assert"))])
    add("two scenarios", [scenario(step()), {**scenario(step(at=3)), "id": "other", "enabled": True}])
    for index, value in enumerate([None, {}, [5], list(range(501)), [entry()] * 2, [entry(id="a b")],
                                   [entry(id=None)], [entry(label=" ")], [entry(label=" x ")], [entry(label=7)],
                                   [entry(label="x" * 101)], [entry(ied_id=5)], [entry(ref=None)],
                                   [entry(ied_id="9" * 32)], [entry(ref="zz")], [entry(ref=CF)],
                                   [entry(report_ref=5)], [entry(report_ref="")], [entry(report_ref="zz")],
                                   [entry(report_ref=P + "LLN0.RP.EventsRCB")],
                                   [entry(ref=POS, report_ref=P + "LLN0.RP.EventsRCB")],
                                   [entry(duration_ms=99)], [entry(duration_ms=True)], [entry(duration_ms=100.0)],
                                   [entry(duration_ms=60000)], [entry(active_value=1)], [entry(active_value=False)],
                                   [entry(ref=ANIN, id="m", extra=1)], [entry(ref=NAME)],
                                   [entry(), entry(id="c2", ref=POS, label="Pos")]]):
        panel(f"panel {index}", value)
    feed = fixtures["FEED1"]
    for signal in [s for s in feed["signals"] if s["writable"] and s["fc"] != "CF"][:20]:
        panel(f"FEED1 panel {signal['ref']}", [{"id": "c", "label": "x", "ied_id": "3" * 32, "ref": signal["ref"]}])
    return out
