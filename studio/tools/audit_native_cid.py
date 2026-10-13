"""Verify customer SCL models through actual native MMS listeners and reads."""
import argparse
import json
from pathlib import Path
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "studio/backend"))
from studio_app.scl.importer import import_scl
from studio_app.runtime.worker_process import WorkerProcess


def audit(path, browse=False):
    binary = ROOT / "studio/native/build/Release"
    models = import_scl(path.read_bytes(), path.name)
    results = []
    with tempfile.TemporaryDirectory(prefix="cid-audit-", dir=ROOT / ".local") as temporary:
        for index, model in enumerate(models):
            config = Path(temporary) / f"ied-{index}.cfg"
            config.write_text(model["cfg"], encoding="utf-8")
            worker = WorkerProcess(binary / "studio_ied.exe")
            result = {"ied": model["name"], "attributes": len(model["signals"])}
            try:
                worker.request("start", timeout=30, cfg=str(config), ip="127.50.0.223",
                               port=18103, controls=model["controls"], edition=model["edition"])
                signal = next(s for s in model["signals"] if s["fc"] == "ST" and s["kind"] in {"BOOLEAN", "INT32", "Enum", "Dbpos"})
                probe = subprocess.run([str(binary / "studio_probe.exe"), "127.50.0.223", "18103",
                                        "read", signal["ref"], signal["fc"]], capture_output=True, text=True, timeout=10)
                result.update(ok=probe.returncode == 0, read=json.loads(probe.stdout))
                if browse:
                    discovery = subprocess.run([str(binary / "studio_probe.exe"), "127.50.0.223", "18103",
                                                "browse"], capture_output=True, text=True, timeout=90)
                    tree = json.loads(discovery.stdout)
                    observed = set()
                    for device in tree.get("devices", []):
                        for node in device["nodes"]:
                            observed.update((leaf["ref"], leaf["fc"]) for leaf in node["leaves"])
                    expected = {(signal["ref"], signal["fc"]) for signal in model["signals"]}
                    result.update(browse_ok=discovery.returncode == 0 and tree.get("ok", False),
                                  missing_references=len(expected - observed), observed_variables=len(observed))
                    result["ok"] = result["ok"] and result["browse_ok"] and result["missing_references"] == 0
            except Exception as error:
                result.update(ok=False, error=str(error))
            finally:
                worker.close()
            results.append(result)
    return results


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("cid", type=Path)
    parser.add_argument("--browse", action="store_true")
    args = parser.parse_args()
    print(json.dumps(audit(args.cid, args.browse), indent=2))
