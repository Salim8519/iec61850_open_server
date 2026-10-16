"""Owner license tool: free keys for you, activate/remove this PC, reset trials.

    python studio/tools/license_admin.py status                 this PC: Machine ID and license
    python studio/tools/license_admin.py key                    make a lifetime key (type it in Studio)
    python studio/tools/license_admin.py key --trial-days 3     make a 3-day evaluation key
    python studio/tools/license_admin.py activate-here          license this PC instantly (no typing)
    python studio/tools/license_admin.py remove                 take the license off this PC
    python studio/tools/license_admin.py remove --free-key --reset-trial   ...and make key + trial reusable
    python studio/tools/license_admin.py reset-trial            let this PC start a free trial again
    python studio/tools/license_admin.py delete-key --key SUBS-...   |   list

Server: the local sandbox by default (start it with license_sandbox.py);
--real uses the server saved by deploy-server.cmd (or --ssh user@host) through the deploy key.
PC: the installed app's folder (%LOCALAPPDATA%\\ScadaBench) by default;
--sandbox uses the source sandbox (studio/build/license-sandbox/studio-license).
Removed files are moved to a removed-<time> folder, never deleted.
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
from studio_app.licensing import hwid  # noqa: E402
from studio_app.licensing.service import default_home  # noqa: E402

LICENSE_FILES = ("license.cert", "license_state.json", ".time_anchor")


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
    command = ["issue", "--email", args.email]
    if args.trial_days:
        command += ["--type", "trial", "--days", str(args.trial_days)]
    license = server(args, *command)["license"]
    print(f"New key: {license['key']}  ({license['type']}{', ' + str(args.trial_days) + ' days' if args.trial_days else ''})")


def activate_here(args):
    folder = home(args)
    result = server(args, "issue", "--email", args.email, "--hwid", hwid.machine_id())
    folder.mkdir(parents=True, exist_ok=True)
    (folder / "license.cert").write_text(json.dumps(result["certificate"], indent=2), encoding="utf-8")
    state = folder / "license_state.json"
    if state.exists():
        state.unlink()
    print(f"This PC is licensed with {result['license']['key']}. Reopen Studio (or refresh the page).")


def remove(args):
    folder = home(args)
    payload = read_payload(folder)
    present = [name for name in LICENSE_FILES if (folder / name).exists()]
    if present:
        target = folder / f"removed-{time.strftime('%Y%m%d-%H%M%S')}"
        target.mkdir()
        for name in present:
            shutil.move(str(folder / name), str(target / name))
        print(f"License removed from this PC (kept in {target}). Reopen Studio to see the activation screen.")
    else:
        print("No license on this PC.")
    key = (payload or {}).get("license_id", "")
    if args.free_key and key.startswith("SUBS-"):
        server(args, "reset-hwid", "--key", key)
        print(f"Key {key} is free for another activation.")
    if args.reset_trial:
        reset_trial(args)


def reset_trial(args):
    result = server(args, "reset-trial", "--hwid", hwid.machine_id())
    print("Trial reset: this PC can start a new free trial." if result.get("had_trial")
          else "This PC had no trial on the server; it can start one.")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("command", choices=["status", "key", "activate-here", "remove", "reset-trial",
                                            "delete-key", "list"])
    parser.add_argument("--email", default="owner@example.com")
    parser.add_argument("--trial-days", type=int, default=0)
    parser.add_argument("--key")
    parser.add_argument("--free-key", action="store_true", help="remove: also unlock the key on the server")
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
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
