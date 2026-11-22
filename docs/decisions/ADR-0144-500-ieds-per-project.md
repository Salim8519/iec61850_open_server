# ADR-0144: A project holds up to 500 IEDs; the window title names the project

Accepted: 2026-11-22. Owner, 2026-11-22, for 0.1.36: "raise to 500 ieds allowed ... same version
0136, dont increase version" and "show project name on windows header of windows. not the app
inside, but the window header". Change [0232](../changes/2026/0232-500-ieds-and-window-title.md).

## Problem

A project was capped at 20 IEDs: import refused more ("This release supports up to 20 IEDs per
project"), a saved project with more than 20 IEDs or SCL sources was refused on open ("Invalid
compact project source table" / "Project must contain a list of at most 20 IEDs"), and one import
or one bulk delete took at most 20. A real station does not fit: Al Kamil 400/132 kV has 28 bay
controllers and had to be split into two projects run side by side.

The app window (a browser `--app` window) always showed "ScadaBench" in its title bar, so two
open stations could not be told apart on the taskbar or in Alt+Tab.

## Decision

1. **One limit, 500 IEDs.** `project.MaxIEDs` (Go) and `studio_app.contracts.limits.MAX_IEDS`
   (frozen Python, owner-approved so dev and the golden vectors match) are the only source of the
   number. It bounds: IEDs per project at import, the IED list and SCL source table of a saved
   project, IEDs selected in one import, and IEDs in one bulk delete (frontend `MAX_BULK`). The
   import preview's `available_slots` counts down from it. The messages keep their wording with
   the new number.
2. **Caches follow the limit.** The SCL source digest caches (Go `maxDigests = 2 * MaxIEDs`,
   `sourceCacheLimit = 1024`; Python `lru_cache(maxsize=1024)`) hold every source of a full
   project, so polls do not re-hash 500 sources.
3. **Window title.** The frontend sets `document.title` to `<project name> - ScadaBench` while a
   project is open and `ScadaBench` otherwise. The browser shows it in the window's own title
   bar, taskbar and Alt+Tab; nothing inside the page changes.

## Compatibility

Every project that opened before still opens. Projects with 21 to 500 IEDs now open in 0.1.36 and
are refused by earlier versions with their old message. Other caps (20 Modbus ranges, 20 network
helper actions per PowerShell start) are unrelated and unchanged. The release keeps version
0.1.36 as the owner asked.
