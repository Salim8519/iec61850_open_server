# Substation Studio developer preview

Copy the entire SubstationStudio folder to a writable Windows x64 location.
Double-click SubstationStudio.exe. It opens the browser and keeps a small console
window running. Do not copy the executable without its _internal folder.
No Docker, Node or separately installed Python is needed at runtime.

Use IEDs & Signals to import a CID/ICD/SCD or create a laboratory bay controller.
Import first asks which IEDs to add; GOOSE peers are not started automatically.
Delete IED previews affected bindings/sources/steps and saves a recovery project
under data/recovery. Physical drawing equipment and original CID files remain.
Add equipment in Substation, select it and bind its status/control in the inspector.
Run starts MMS workers at the addresses shown under Connections.
Loopback listeners are accessible only on the same PC. LAN setup is not included.
Scenarios can set/check signal values at scheduled times and pause with automation.
CSV sources support explicit time/value columns, hold/linear interpolation and repeat.
The complete trace stays inside the project archive. SCL schema diagnostics appear
under the selected IED; unsupported revisions are not silently treated as valid.

Use Stop before exiting; press Ctrl+C in the console for graceful shutdown.
Closing the browser alone does not stop simulation. Project downloads are .subsim
archives; Open Project restores them. Current autosave is data/autosave.subsim.
Autosave is restored automatically with every IED stopped.
Unreadable archives are preserved under data/recovery before allowing new work.
Only one launcher may use a data folder at a time. Logs are under data/logs.
New saves use compact format 4, retaining original SCL once per source and
rebuilding runtime tables on reopen. Older preview versions cannot open these
new saves; retain your pre-upgrade recovery backups.

Signal mapping follows IED, report, description and exact path; All signals is an
explicit alternative. Report selection filters mapping choices, not subscriptions.
Signal Test saves independent value controls and Boolean pulses without SLD symbols.
Pulse timing continues in the backend while automation is paused; Stop cancels it.
Newer writes win over delayed pulse returns. Resume automatic source explicitly
after a manual override. CB/DS/ES motion visualizes confirmed feedback only.
Switch state may use native Dbpos, one Boolean, or two Boolean contacts; command
mapping is separate and accepts DPC controls only. Trip relay uses an SPC Operate
and explicit Reset and never auto-resets.
Select IED checkboxes for a combined deletion preview and one recovery backup.

For an offline engine check, run `SubstationStudio.exe --self-test`. It uses
temporary generated data, verifies the bundled schema and an independent loopback
MMS read, then removes its temporary project and stops its workers. This is not
a substitute for clean-machine or interactive browser acceptance.

This is an engineering preview, not a production or protection-system tool.
Do not connect it to operational substation networks.
Visual acceptance, clean-machine verification and the remaining feature gates
listed in the repository docs/known-issues.md are still pending.

Dependency license files are under _internal/licenses. libiec61850 is GPL-3.0;
IEC SCL code-component notices/EULA are retained under _internal/schemas.
this preview is for local testing. External distribution requires a complete
license/source-compliance review, including bundled Python/frontend dependencies.
