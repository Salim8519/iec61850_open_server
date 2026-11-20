---
title: "Import a CID or SCD file"
description: "Import an IEC 61850 CID, ICD or SCD file into ScadaBench, choose which IEDs to add, and remove IEDs you no longer need."
section: devices
order: 1
video: cid-scd
---

ScadaBench builds each IED from its configuration file. Use the file that your engineering tool
exported for the relay or for the whole substation.

## Which file to use

- **CID** (Configured IED Description): the file of one configured IED. Use it for a single relay.
- **ICD** (IED Capability Description): the file of an IED type. It describes what the device can do.
- **SCD** (Substation Configuration Description): the file of the whole substation. It holds every
  IED, the single-line diagram and the GOOSE links between IEDs.

An SCD gives you the most. Its single-line diagram can be built automatically, see
[Build the diagram from the SCD](/docs/sld/build-from-scd/).

## Import the file

1. Click the **Devices** place.
2. Press **Import CID / SCD** at the top of the IED list. In an empty project, the button is
   **Import CID / SCD file** in the middle of the screen.
3. Choose your file in the window that opens.
4. The title **Choose IEDs to add** lists the IEDs in the file, with their manufacturer and type.
5. Tick the IEDs you want.
6. Press **Add 2 selected IEDs** (the number changes with your choice).

The IEDs appear in the list. Each one starts as **Simulate**, which means ScadaBench plays it.
To change this, see [Simulate and run IEDs](/docs/devices/simulate-and-run/).

## Things to know

- You cannot import while IEDs run. The button is grey, and its tooltip says **Stop the run to import
  IEDs**. Press **Stop** first.
- ScadaBench keeps the original configuration. Anything unusual in the file is listed as a note.
  Nothing is dropped.
- If the file has no data type section, ScadaBench shows: "This file has no data type section
  (DataTypeTemplates)". Export the file again from your engineering tool, with the data types
  included.

## Find an IED in a long list

Use the **Filter IEDs** bar above the list. Type part of a name, or filter by role.
Press **Clear** to see all IEDs again.

## Remove IEDs

1. Press **Select** at the top of the list. Check boxes appear next to the IEDs.
2. Tick the IEDs to remove. The bar at the bottom of the list shows **N selected**. Press **All** to tick every IED in the filtered list, or **Cancel** to stop.
3. Press **Delete IED** (or **Delete 3 IEDs**) and confirm in the dialog.

You cannot remove IEDs while they run.
