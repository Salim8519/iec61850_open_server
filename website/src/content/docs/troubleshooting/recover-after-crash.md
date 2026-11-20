---
title: "Recover a project after a crash"
description: "What to do when the ScadaBench window stops drawing, when ScadaBench closed unexpectedly, or when a startup recovery notice appears."
section: "troubleshooting"
order: 6
---

Sometimes the window stops drawing or ScadaBench closes unexpectedly. Your project is usually safe. This page shows how to get it back.

## The window stops drawing

The page shows **Something went wrong. Your IEDs are still running.**

The IEDs run in the background engine, not in the window. They keep running.

1. Press **Reload**.
2. Check the project. It should open as it was.

## ScadaBench closed unexpectedly

1. Start ScadaBench again.
2. ScadaBench opens the last saved substation. The message says **Recovered the last saved substation. All IEDs remain stopped.**
3. Check your project.
4. Press **Run** to start the IEDs again.

With auto save on (the default), your last change is already saved. With auto save off, changes made after your last **Save project** may be lost. Press **Ctrl+S** often when auto save is off.

## Startup notice: the record was moved

A notice can say that ScadaBench could not read the last autosave and moved it to the recovery folder. For example:

> ScadaBench started with an empty substation; the record was moved to the recovery folder.

The unreadable autosave is moved to the recovery folder, so nothing is deleted. To bring back a project you saved earlier, open **File** and press **Open file…**.

## Get back a saved project

1. Open **File**.
2. Press **Open file…** and pick the project file.

To keep a copy of the project, press **Save a copy…**. It is saved to your Downloads folder or as a file you choose.

## Leftover network addresses

After a crash, the ScadaBench Lab card may still hold addresses from the last run. ScadaBench shows **Remove them** or **Keep the addresses for now**. Press **Show addresses** to see the list first.

- **Remove them** takes off the addresses that ScadaBench added. Windows asks once.
- **Keep the addresses for now** leaves them on the card for now.
