"""Write generated/trial.json: the frozen Python trial-hardening answers for the Go unit tests.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/licensing/testdata/make_trial.py
Covers trial_parts.py, server_time.py, domain/trial_policy.py and clock_store.ClockValue
(with an in-memory registry). Stamps are signed with the test seed only.
"""
import json
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vector_common import OTHER_SEED, b64, compact, license_signer, outcome, signed, signed_raw, write  # noqa: E402
from studio_app.domain import license_policy, trial_policy  # noqa: E402
from studio_app.licensing import server_time, trial_parts  # noqa: E402
from studio_app.licensing.clock_store import ClockValue  # noqa: E402

NOW, DAY = 1_760_000_000, 86400
KID = list(license_signer.keys())[0]


def stamp_text(**fields):
    payload = {"kid": KID, "purpose": "server-time", "time": NOW, "version": 1, **fields}
    return signed_raw(compact(payload))


def parts_cases():
    guids = [None, "", " 1A2B3C4D-AAAA-BBBB-CCCC-0123456789EF ", "x", "\tGuid "]
    serials = [None, 0, 0xA1B2C3D4, 0xFFFFFFFF, 1]
    uuids = ["00112233445566778899AABBCCDDEEFF", "", "00" * 16, "ff" * 16, "11" * 16, "not-hex",
             "03000200040005000006000700080009", "00020003000400050006000700080009",
             "12345678123456789" + "0abcddeefaabbcc", "78563412341278569" + "0abcddeefaabbcc",
             " 4c4c4544004e3510804ab4c04f4e3732\n", "4c4c4544004e3510804ab4c04f4e373", "é" * 32,
             "4C4C4544004E3510804AB4C04F4E3732", "0" * 31 + "1"]
    cases = []
    for index, uuid in enumerate(uuids):
        cases.append({"guid": guids[index % len(guids)], "serial": serials[index % len(serials)], "uuid": uuid,
                      "junk": trial_parts.is_junk_firmware(uuid), "swap": trial_parts.swap_uuid_fields(uuid),
                      "parts": trial_parts.compose_parts(guids[index % len(guids)], serials[index % len(serials)],
                                                         uuid)})
    return cases


def stamp_cases():
    keys = license_signer.keys()
    good = stamp_text()
    altered = dict(good, payload=good["payload"].replace(str(NOW), str(NOW + 99 * DAY)))
    stamps = [good, json.dumps(good), {"certificate": good}, None, "", 5, [good], "not json", "[1]",
              "﻿" + json.dumps(good), stamp_text(purpose="license"), stamp_text(purpose=None),
              stamp_text(version="1"), stamp_text(version=2), stamp_text(version=True), stamp_text(version=1.9),
              stamp_text(version=[1]), stamp_text(time=0), stamp_text(time=-5), stamp_text(time="17"),
              stamp_text(time="x"), stamp_text(time=1.5), stamp_text(time=None), stamp_text(time=10 ** 18),
              signed_raw(compact({"kid": KID, "purpose": "server-time", "version": 1})), altered,
              signed_raw(compact({"kid": KID, "purpose": "server-time", "time": NOW}), OTHER_SEED),
              json.loads(signed()), stamp_text(kid=["x"]), {"payload": compact({"purpose": "server-time"}),
                                                            "signature": "!"}]
    cases = []
    for stamp in stamps:
        result = outcome(lambda: server_time.verify(stamp, keys))
        cases.append({"stamp": compact(stamp), **result})
    cases.append({"stamp": compact(good), "no_keys": True, **outcome(lambda: server_time.verify(good, {}))})
    return cases


def status(**fields):
    base = {"licensed": True, "mode": "trial", "plan": "trial", "days_remaining": 3,
            "expires_at": NOW + 7 * DAY, "license_id": "TRIAL-abababababab"}
    return license_policy.LicenseStatus(**{**base, **fields})


def policy_cases():
    statuses = [status(), status(expires_at=None), status(plan="lifetime", mode="lifetime"),
                license_policy.unlicensed("x", "trial", "TRIAL-a", NOW), status(license_id=None)]
    times = [(0, 0, NOW), (NOW, 0, NOW + 10), (NOW, 0, NOW + DAY), (NOW + 7 * DAY, 0, NOW),
             (NOW + 7 * DAY + 1, 0, NOW), (0, NOW, NOW + 3 * DAY - 1), (0, NOW, NOW + 3 * DAY),
             (NOW - 5, NOW, NOW + 3 * DAY), (NOW + 1, NOW, NOW + 9 * DAY), (NOW, NOW, NOW + 3 * DAY)]
    cases = []
    for index, item in enumerate(statuses):
        for confirmed, suspicious, now in times:
            host = ["license.scadabench.com", "the license server"][index % 2]
            applied = trial_policy.apply(item, confirmed, suspicious, now, host)
            cases.append({"status": compact(item.as_dict()), "confirmed": confirmed, "suspicious": suspicious,
                          "now": now, "host": host, "applied": compact(applied.as_dict()),
                          "due": trial_policy.check_due(confirmed, suspicious, now),
                          "latest": trial_policy.latest_seen(now - DAY, confirmed),
                          "missing": trial_policy.anchor_missing([1] if index % 2 else [], confirmed)})
    return cases


class FakeRegistry:
    def __init__(self, values):
        self.values = dict(values)

    def get(self, name):
        return self.values.get(name)

    def set(self, name, text):
        self.values[name] = str(text)


def snapshot(home):
    return {path.name: b64(path.read_bytes()) for path in sorted(home.iterdir()) if path.is_file()}


def clock_cases():
    setups = [({}, {}, None), ({".time_anchor": b" 17\n"}, {"TimeAnchor": "18"}, b'{"time_anchor": "19", "a": 1}'),
              ({".time_anchor": b"\xff"}, {"TimeAnchor": ""}, b'{"time_anchor": 20}'),
              ({".time_anchor": b"x"}, {"TimeAnchor": " 21 "}, b"[1]"), ({}, {}, b'{"time_anchor": "22"}'),
              ({".grace_start": b"0"}, {"GraceStart": "-3"}, b'{"grace_start": "1760000000"}')]
    cases = []
    for files, registry_values, state in setups:
        for with_registry in (True, False):
            for name in ("time_anchor", "grace_start"):
                with tempfile.TemporaryDirectory() as folder:
                    home = Path(folder)
                    for file_name, data in files.items():
                        (home / file_name).write_bytes(data)
                    if state is not None:
                        (home / "license_state.json").write_bytes(state)
                    registry = FakeRegistry(registry_values) if with_registry else None
                    value = ClockValue(name, home, registry)
                    case = {"name": name, "files": {key: b64(data) for key, data in files.items()},
                            "registry": registry_values if with_registry else None,
                            "state": None if state is None else b64(state), "read_all": value.read_all(),
                            "read_ints": [str(number) for number in value.read_ints()], "places": value.places()}
                    value.write("1760000500")
                    case["written"] = {"files": snapshot(home), "registry": registry.values if registry else None}
                    cases.append(case)
    return cases


def main():
    write("trial.json", {"parts": parts_cases(), "stamps": stamp_cases(), "policy": policy_cases(),
                         "clock": clock_cases(), "support_url": trial_policy.SUPPORT_URL,
                         "texts": {**trial_policy.SERVER_ERRORS, "needs_online": trial_policy.needs_online("h.example"),
                                   "clock_rollback": license_policy.CLOCK_ROLLBACK}})


if __name__ == "__main__":
    main()
