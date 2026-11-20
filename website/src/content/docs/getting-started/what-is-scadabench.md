---
title: "What ScadaBench is"
description: "ScadaBench is an offline Windows app that plays IEC 61850 substation devices (IEDs) on your PC, so you can test SCADA and other tools without the real hardware."
section: getting-started
order: 1
video: getting-started
---

ScadaBench is a test bench for IEC 61850 substation systems. It runs on your Windows PC, works
offline, and shows everything in English.

It does two jobs:

- It **plays IEDs**. An IED is a protection relay, bay controller or meter. You import the
  configuration file of a real IED, and ScadaBench answers as that IED would: on its own address,
  with the same data model, reports and GOOSE messages.
- It **reads real IEDs**. You can connect to a real device on your network and watch its live
  values, without changing anything on it.

## Words you will see

- **IED**: a substation device, such as a relay. Each one has a name, like `Feeder01`.
- **CID, ICD, SCD**: configuration files written in the IEC 61850 format (SCL). A CID describes one
  configured IED. An SCD describes the whole substation.
- **SLD**: single-line diagram. A drawing of the substation's switches, breakers and busbars.
- **GOOSE**: a fast message that IEDs send each other, for trips and interlocks.
- **Port 102**: the port on which an IED answers MMS, the protocol that SCADA uses to read it.

## The five places

The left side of the window has five places. Click one to open it.

1. **Devices**: your IEDs. Import files, choose what ScadaBench does with each IED, and look at
   its data points. This place opens first.
2. **SLD**: draw the substation and link each symbol to an IED signal.
3. **Test**: change values on purpose, run generators and timed sequences.
4. **Tools**: helpers such as Find IEDs, the SCADA gateway and the GOOSE and Sampled Values traffic.
5. **Settings**: the network adapter, your licence and information about the app.

Under the five places, **Help** opens the guided tour again and links to the video tutorials.

## The top bar

- The **title** and a short line under it tell you where you are.
- **File** holds the project commands: open, new, save, close and the auto save switch. See
  [Saving the project](/docs/getting-started/saving-the-project/).
- **Undo** and **Redo** step back and forward through your last changes.
- The **Run** button starts every IED that is not skipped. While IEDs run, it shows **Stop**.
  See [Your first run](/docs/getting-started/first-run/).

## Where to go next

Read [Install and first start](/docs/getting-started/install-and-first-start/), then run the
sample IED in [Your first run](/docs/getting-started/first-run/).
