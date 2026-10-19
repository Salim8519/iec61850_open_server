"""Trial part hashes: identical to the Go server, placeholders never sent, refusals in plain words."""
import io
import json
from pathlib import Path
from urllib.error import HTTPError
import pytest
from studio_app.domain import trial_policy as policy
from studio_app.licensing import client, trial_parts

VECTOR = json.loads((Path(__file__).resolve().parents[1] / "fixtures/licensing/go_vector.json").read_text())


def test_part_hashes_match_the_go_server_byte_for_byte():
    parts = trial_parts.compose_parts(" 1A2B3C4D-AAAA-BBBB-CCCC-0123456789EF ", 0xA1B2C3D4,
                                      "00112233445566778899AABBCCDDEEFF")
    assert parts == VECTOR["part_hashes"]


@pytest.mark.parametrize("junk", ["", "00" * 16, "ff" * 16, "FF" * 16, "11" * 16, "not-hex",
                                  "03000200040005000006000700080009", "00020003000400050006000700080009",
                                  "12345678123456789" + "0abcddeefaabbcc", "78563412341278569" + "0abcddeefaabbcc"])
def test_placeholder_firmware_uuids_are_never_sent(junk):
    assert trial_parts.is_junk_firmware(junk)
    assert trial_parts.compose_parts("guid", 0x1234, junk)["firmware"] == ""


def test_unreadable_or_placeholder_parts_are_left_empty_not_invented():
    assert trial_parts.compose_parts(None, None, "") == {"guid": "", "volume": "", "firmware": ""}
    assert trial_parts.compose_parts("g", 0, "00112233445566778899aabbccddeeff")["volume"] == ""
    assert trial_parts.compose_parts("g", 0xFFFFFFFF, "")["volume"] == ""
    assert not trial_parts.is_junk_firmware("4c4c4544004e3510804ab4c04f4e3732")  # a real Dell-style UUID


def test_this_pc_reports_three_part_slots():
    assert set(trial_parts.machine_parts()) == {"guid", "volume", "firmware"}


def test_client_sends_only_known_parts():
    seen = {}

    def opener(request, timeout):
        body = json.loads(request.data) if request.data else None
        seen[request.full_url.rsplit("/", 1)[-1]] = body
        answer = {"nonce": "00", "difficulty": 1, "timestamp": 1, "mac": "m"} if body is None else {"certificate": {}}
        return io.BytesIO(json.dumps(answer).encode())
    client.LicenseClient("https://license.example.com", opener).start_trial("ab" * 32, {"guid": "g", "firmware": ""})
    assert seen["trial"]["parts"] == {"guid": "g"}


@pytest.mark.parametrize("code, message", [("trial_used", policy.TRIAL_USED), ("email_required", policy.EMAIL_REQUIRED),
                                           ("email_used", policy.EMAIL_USED), ("bad_code", policy.BAD_CODE),
                                           ("email_disposable", policy.EMAIL_DISPOSABLE)])
def test_trial_refusals_and_steps_are_plain_english_with_a_support_link(code, message):
    def opener(request, timeout):
        body = io.BytesIO(json.dumps({"error": "terse server text", "code": code}).encode())
        raise HTTPError(request.full_url, 409, "refused", {}, body)
    with pytest.raises(client.LicenseServerError) as refused:
        client.LicenseClient("https://license.example.com", opener).verify("TRIAL-x", "ab" * 32)
    assert str(refused.value) == message and refused.value.code == code
    assert policy.SUPPORT_URL in message
