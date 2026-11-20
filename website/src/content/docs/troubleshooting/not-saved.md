---
title: "Not saved"
description: "What the Not saved message means in the top bar and on the SNMP, gateway and command settings, and how to save again."
section: "troubleshooting"
order: 5
---

ScadaBench shows **Not saved** when a change could not be written to the project file. Your change is not lost while ScadaBench is open. Fix the problem and save again.

## Where you see it

- **Not saved yet** in the top bar. The diagram could not be saved.
- **Not saved · Ctrl+S** next to the File menu. Auto save is off and some changes are not written yet.
- **Not saved yet. The Traps tab needs attention** (or the **Settings** tab) in SNMP devices.
- **These changes are not saved yet** in the gateway point list.
- **Not saved, so the setting went back** for a select before operate setting.

## What to do

1. Read the message under the warning. It names the problem.
2. Fix the problem, for example a value that is out of range or a field left empty.
3. Wait a few seconds. The change saves by itself once the problem is fixed, or the next edit tries again.

If it still says **Not saved**:

1. Press **Ctrl+S**, or open **File** and press **Save project**.
2. If the message stays, press **Save as…** and save a copy to a new file.

## Auto save

Auto save is on by default. Every change is written at once.

- To turn it off, open **File** and press **Turn off auto save**. Changes are kept while ScadaBench is open. Press **Ctrl+S** to write them.
- To turn it on again, press **Turn on auto save**.

With auto save off, changes that are not saved are lost if ScadaBench closes before you save. The next start opens the last saved project.

## Gateway point list

The point list can only change while the gateway is stopped. If the page says **Stop the gateway to change the point list.**, press **Stop gateway** first.

If a row has a problem, the list does not save until the row is fixed. You can also press **Discard unsaved changes** to drop the changes.
