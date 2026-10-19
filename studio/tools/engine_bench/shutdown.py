"""Ways a benchmark launcher can end, the way a user ends the app.

- ctrl_c: the launcher gets its own hidden console; a helper attaches to it and sends Ctrl+C.
  Python raises KeyboardInterrupt in the main thread, exactly what closing the app window
  does (interrupt_main), so the launcher's own cleanup (runtime.close, server.close) runs.
- kill: TerminateProcess on the backend interpreters only (crash / Task Manager); native
  workers must notice the closed pipe and exit by themselves.
Workers are started with CREATE_NO_WINDOW (own console), so Ctrl+C never reaches them.
"""
import subprocess
import sys
import time

CTRL_C_HELPER = ("import ctypes,sys;k=ctypes.windll.kernel32;k.FreeConsole();"
                 "ok=k.AttachConsole(int(sys.argv[1]));k.SetConsoleCtrlHandler(None,True);"
                 "sent=k.GenerateConsoleCtrlEvent(0,0);sys.exit(0 if ok and sent else 1)")


def hidden_console():
    """Popen options for a launcher with its own hidden console that accepts Ctrl+C.

    The "ignore Ctrl+C" flag is inherited from the parent (a tool runner often sets it), so this
    process clears it first; the launcher then inherits Ctrl+C processing like a normal start."""
    import ctypes
    ctypes.windll.kernel32.SetConsoleCtrlHandler(None, False)
    info = subprocess.STARTUPINFO()
    info.dwFlags |= subprocess.STARTF_USESHOWWINDOW
    info.wShowWindow = 0  # SW_HIDE
    return {"creationflags": subprocess.CREATE_NEW_CONSOLE, "startupinfo": info}


def send_ctrl_c(pid):
    python = getattr(sys, "_base_executable", sys.executable)
    completed = subprocess.run([python, "-c", CTRL_C_HELPER, str(pid)], capture_output=True, timeout=30,
                               creationflags=subprocess.CREATE_NO_WINDOW)
    return completed.returncode == 0


def backend_pids(engine):
    return [pid for pid, name in engine.job.processes().items() if name.lower() in engine.backend_names]


def native_pids(engine):
    return {pid: name for pid, name in engine.job.processes().items() if name.lower().startswith("studio_")}


def kill_backend(engine):
    import ctypes
    kernel = ctypes.windll.kernel32
    for pid in backend_pids(engine):
        handle = kernel.OpenProcess(0x0001, False, pid)  # PROCESS_TERMINATE
        if handle:
            kernel.TerminateProcess(handle, 1)
            kernel.CloseHandle(handle)


def wait_gone(engine, select, timeout):
    """Seconds until `select(engine)` is empty, or None when it is still not empty after `timeout`."""
    started = time.perf_counter()
    while time.perf_counter() - started < timeout:
        if not select(engine):
            return round(time.perf_counter() - started, 3)
        time.sleep(0.05)
    return None
