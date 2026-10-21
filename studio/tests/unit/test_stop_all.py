"""Change 0152: stop_all.py picks only ScadaBench's own processes, one id per process tree."""
from studio.tools.stop_all import ROOT, is_scadabench, targets


def row(pid, name, command="", parent=1):
    return {"ProcessId": pid, "ParentProcessId": parent, "Name": name, "CommandLine": command}


def test_app_and_workers_are_chosen_but_not_the_network_helper():
    assert is_scadabench(row(1, "ScadaBench.exe"))
    assert is_scadabench(row(2, "studio_ied.exe"))
    assert is_scadabench(row(3, "STUDIO_SNMP.EXE"))
    assert not is_scadabench(row(4, "ScadaBench.exe", "ScadaBench.exe --network-helper plan.json"))


def test_python_and_node_only_when_they_run_this_repository():
    launcher = f'python "{ROOT / "studio" / "tools" / "run_dev_engine.py"}"'
    vite = f"node {(ROOT / 'studio' / 'frontend' / 'node_modules' / 'vite' / 'bin' / 'vite.js').as_posix()}"
    assert is_scadabench(row(1, "python.exe", launcher))
    assert is_scadabench(row(2, "node.exe", vite))
    assert not is_scadabench(row(3, "python.exe", "python -m http.server 8765"))
    assert not is_scadabench(row(4, "node.exe", "node C:/other/vite.js"))
    assert not is_scadabench(row(5, "python.exe", f'python "{ROOT / "studio" / "tools" / "verify.py"}"'))


def test_children_of_a_chosen_process_are_left_to_its_tree_stop():
    rows = [row(10, "studio_server.exe"), row(11, "studio_ied.exe", parent=10),
            row(12, "studio_goose.exe", parent=10), row(20, "studio_snmp.exe", parent=99)]
    assert [item["ProcessId"] for item in targets(rows, own=0)] == [10, 20]
    assert targets([row(5, "studio_ied.exe")], own=5) == []
