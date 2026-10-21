"""Owner license tool: free keys for you, activate/remove this PC, reset trials.

    python studio/tools/license_admin.py status                 this PC: Machine ID and license
    python studio/tools/license_admin.py key                    make a lifetime key (type it in Studio)
    python studio/tools/license_admin.py key --trial-days 3     make a 3-day evaluation key
    python studio/tools/license_admin.py activate-here          license this PC instantly (no typing)
    python studio/tools/license_admin.py remove                 take the license off this PC
    python studio/tools/license_admin.py remove --reset-trial   ...and let this PC start a free trial again
    python studio/tools/license_admin.py reset-trial            let this PC start a free trial again
    python studio/tools/license_admin.py reset-trial --machine-id 64HEX   ...a customer's PC (its Machine ID)
    python studio/tools/license_admin.py reset-trial-ip --ip 203.0.113.7  stop asking that network for an email
    python studio/tools/license_admin.py allow-email --email name@company.com   one more trial for that address
    python studio/tools/license_admin.py delete-key --key SUBS-...   |   list

Server: the local sandbox by default (start it with license_sandbox.py);
--real uses the server saved by deploy-server.cmd (or --ssh user@host) through the deploy key.
PC: the installed app's folder (%LOCALAPPDATA%\\ScadaBench) by default;
--sandbox uses the source sandbox (studio/build/license-sandbox/studio-license).
Removed files are moved to a removed-<time> folder, never deleted. A license the server cancelled
stays cancelled on this PC (revoked_ids, ADR-0097): remove never touches that record, and a key
can no longer be freed for another computer.
"""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

STUDIO = Path(__file__).resolve().parents[1]
SANDBOX = STUDIO / "build/license-sandbox"
sys.path.insert(0, str(STUDIO / "backend"))
from studio_app.licensing import hwid, trial_parts  # noqa: E402
from studio_app.licensing.clock_store import REGISTRY_KEY, REGISTRY_NAMES  # noqa: E402
from studio_app.licensing.service import default_home  # noqa: E402

OWNER_EMAIL = "owner@example.com"
LICENSE_FILES = ("license.cert", "license_state.json", ".time_anchor", ".server_time", ".grace_start")


DEPLOY_STATE = Path(os.environ.get("LOCALAPPDATA", "")) / "ScadaBench-deploy"


def server_command(args, *command):
    if args.ssh or args.real:
        host = args.ssh or (DEPLOY_STATE / "server.txt").read_text(encoding="utf-8").strip()
        key = Path.home() / ".ssh/scadabench_deploy"
        ssh = ["ssh", "-o", "BatchMode=yes"] + (["-i", str(key), "-o", "IdentitiesOnly=yes"] if key.exists() else [])
        quoted = [part if part.replace("-", "").replace("@", "").replace(".", "").isalnum() else f"'{part}'" for part in command]
        return [*ssh, host, "sudo", "/usr/local/bin/license-server-admin", *quoted]
    exe = SANDBOX / "license-server.exe"
    if not exe.exists():
        raise SystemExit("Local sandbox server not built yet. Run: python studio/tools/license_sandbox.py")
    return [str(exe), *command, "--data", str(SANDBOX / "server-data")]


def server(args, *command):
    done = subprocess.run(server_command(args, *command), capture_output=True, text=True)
    if done.returncode != 0:
        raise SystemExit((done.stderr or done.stdout).strip())
    return json.loads(done.stdout)


def home(args):
    return SANDBOX / "studio-license" if args.sandbox else default_home()


def read_payload(folder):
    try:
        bundle = json.loads((folder / "license.cert").read_text(encoding="utf-8"))
        return json.loads(bundle["payload"])
    except (OSError, ValueError, KeyError):
        return None


def status(args):
    folder, machine = home(args), hwid.machine_id()
    print(f"Machine ID : {machine}\nFolder     : {folder}")
    payload = read_payload(folder)
    if payload is None:
        print("License    : none on this PC")
        return
    expires = payload.get("expires_at") or 0
    left = f"{(expires - time.time()) / 86400:.1f} days left" if expires else "never expires"
    print(f"License    : {payload.get('type')} {payload.get('license_id')} ({left})")
    print(f"For this PC: {'yes' if payload.get('hwid') == machine else 'NO - belongs to another PC'}")


def make_key(args):
    command = ["issue", "--email", args.email or OWNER_EMAIL]
    if args.trial_days:
        command += ["--type", "trial", "--days", str(args.trial_days)]
    license = server(args, *command)["license"]
    print(f"New key: {license['key']}  ({license['type']}{', ' + str(args.trial_days) + ' days' if args.trial_days else ''})")


def activate_here(args):
    folder = home(args)
    result = server(args, "issue", "--email", args.email or OWNER_EMAIL, "--hwid", hwid.machine_id())
    folder.mkdir(parents=True, exist_ok=True)
    (folder / "license.cert").write_text(json.dumps(result["certificate"], indent=2), encoding="utf-8")
    state = folder / "license_state.json"
    if state.exists():
        state.unlink()
    print(f"This PC is licensed with {result['license']['key']}. Reopen Studio (or refresh the page).")


def move_registry_aside(target):
    """Save the HKCU clock copies into the removed-<time> folder, then clear them."""
    import winreg
    saved = {}
    try:
        with winreg.OpenKey(winreg.HKEY_CURRENT_USER, REGISTRY_KEY, 0, winreg.KEY_ALL_ACCESS) as key:
            for name in (value for key, value in REGISTRY_NAMES.items() if key != "revoked_ids"):
                try:
                    saved[name] = winreg.QueryValueEx(key, name)[0]
                    winreg.DeleteValue(key, name)
                except OSError:
                    continue
    except OSError:
        return
    if saved:
        (target / "registry.json").write_text(json.dumps(saved, indent=2), encoding="utf-8")


def remove(args):
    folder = home(args)
    present = [name for name in LICENSE_FILES if (folder / name).exists()]
    if present:
        target = folder / f"removed-{time.strftime('%Y%m%d-%H%M%S')}"
        target.mkdir()
        for name in present:
            shutil.move(str(folder / name), str(target / name))
        if not args.sandbox:
            move_registry_aside(target)
        print(f"License removed from this PC (kept in {target}). Reopen Studio to see the activation screen.")
    else:
        print("No license on this PC.")
    if args.reset_trial:
        reset_trial(args)


def reset_trial(args):
    """Owner override: forget a trial and its hardware parts (this PC, or --machine-id)."""
    machine = getattr(args, "machine_id", None)
    command = ["reset-trial", "--hwid", machine or hwid.machine_id()]
    if not machine:
        parts = trial_parts.machine_parts()
        for kind in ("guid", "volume", "firmware"):
            if parts.get(kind):
                command += [f"--{kind}-hash", parts[kind]]
    result = server(args, *command)
    who = "That computer" if machine else "This PC"
    print(f"Trial reset: {who.lower()} can start a new free trial." if result.get("had_trial")
          else f"{who} had no trial on the server; it can start one.")


def reset_trial_ip(args):
    if not args.ip:
        raise SystemExit("--ip is required, for example --ip 203.0.113.7")
    result = server(args, "reset-trial-ip", "--ip", args.ip)
    print(f"Network {args.ip} starts new free trials without an email step again." if result.get("had_count")
          else f"Network {args.ip} had no recent trials.")


def allow_email(args):
    if not args.email or "@" not in args.email:
        raise SystemExit("--email is required, for example --email name@company.com")
    result = server(args, "allow-email", "--trial-email", args.email)
    print("That address can confirm one more free trial." if result.get("email_had_trial")
          else "That address had no trial; nothing to change.")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("command", choices=["status", "key", "activate-here", "remove", "reset-trial",
                                            "reset-trial-ip", "allow-email", "delete-key", "list"])
    parser.add_argument("--email", default=None, help="key/activate-here: buyer email; allow-email: the address")
    parser.add_argument("--trial-days", type=int, default=0)
    parser.add_argument("--key")
    parser.add_argument("--machine-id", help="reset-trial: a customer's 64-character Machine ID")
    parser.add_argument("--ip", help="reset-trial-ip: the network address to clear")
    parser.add_argument("--reset-trial", action="store_true", help="remove: also reset this PC's trial")
    parser.add_argument("--ssh", help="use a server by address, e.g. scadabench@203.0.113.10")
    parser.add_argument("--real", action="store_true", help="use the deployed server saved by deploy-server.cmd")
    parser.add_argument("--sandbox", action="store_true", help="act on the source sandbox, not the installed app")
    args = parser.parse_args(argv)
    if os.name != "nt":
        raise SystemExit("Run this on the Windows PC that runs Studio.")
    if args.command == "status":
        status(args)
    elif args.command == "key":
        make_key(args)
    elif args.command == "activate-here":
        activate_here(args)
    elif args.command == "remove":
        remove(args)
    elif args.command == "reset-trial":
        reset_trial(args)
    elif args.command == "reset-trial-ip":
        reset_trial_ip(args)
    elif args.command == "allow-email":
        allow_email(args)
    elif args.command == "delete-key":
        if not args.key:
            raise SystemExit("--key is required")
        print(json.dumps(server(args, "delete", "--key", args.key)))
    else:
        result = server(args, "list")
        for item in result["licenses"]:
            state = "revoked" if item["revoked"] else ("on a PC" if item["hwid"] else "unused")
            print(f"{item['key']}  {item['type']:<8} {state:<8} {item['email']}")
        print(f"Trial machines: {result['trial_machines']}")
        if result.get("busy_networks"):
            print(f"Networks with many recent trials (they get an email step): {result['busy_networks']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
