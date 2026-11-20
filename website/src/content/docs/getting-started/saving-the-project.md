---
title: "Saving the project and the File menu"
description: "How ScadaBench saves your work with auto save, how to turn auto save off, and how to use the File menu, Save, Save as, Undo and Redo."
section: getting-started
order: 4
---

A **project** holds your IEDs, your single-line diagram, your test points and your generators.
ScadaBench keeps one project open at a time.

## Auto save is on by default

With auto save on, every change is written to the project at once. If the computer or the app
closes unexpectedly, your last change is still there.

You can see the setting at the top of the window, in the **File** menu:

- While auto save is on, the menu item says **Turn off auto save**. Press it to turn auto save off.
- While auto save is off, the menu item says **Turn on auto save**. Press it to turn auto save on again.
- Turning auto save on writes your unsaved changes first.

Your choice is remembered the next time you start ScadaBench.

## Working with auto save off

Sometimes you want to try changes and keep the last saved version safe. Turn auto save off, then:

1. Make your changes. They stay in the open project, and every page uses them.
2. The top bar says **Not saved · Ctrl+S** while changes are waiting.
3. Press **Ctrl+S**, or open **File** and press **Save project**, to write them.

If ScadaBench closes before you save, the next start opens the last saved project. Your unsaved
changes are lost, so save first.

## The File menu

| Item | What it does |
| --- | --- |
| Open file… | Opens a saved project file. |
| New project… | Closes the open project and starts an empty one. You give it a name. |
| Save project | Writes the unsaved changes (Ctrl+S). |
| Save as… | Saves a copy of the project under another name or location. |
| Close project… | Closes the project and opens an empty one. Open file brings it back. |
| Turn off / on auto save | The switch described above. |

Before you start a new project or close the current one, the dialog offers **Save a copy…**. Use it
if you want to keep the project. Once it closes, the project is no longer open in ScadaBench.

Open, New, Save and Close are greyed out while IEDs run. Press **Stop** first.

## Undo and Redo

- **Undo** (Ctrl+Z) takes back your last change. **Redo** (Ctrl+Y) brings it back.
- ScadaBench remembers the last 20 changes in memory. Older steps cannot be undone.
- Undo and Redo work only while the IEDs are stopped. Changes made while the IEDs run are not undone.
