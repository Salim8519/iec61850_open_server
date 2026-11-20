---
title: "Offline activation"
description: "Activate ScadaBench on a computer with no internet: copy the activation code, get a license file on another computer and load it."
section: "licence"
order: 3
video: "activation"
---

Use offline activation for a substation PC or lab PC without internet. You need one other computer with internet.

## What you need

- The ScadaBench computer, with no internet.
- Another computer with internet and a browser.
- Your license key from the purchase email.
- A USB stick or another way to move a small text file.

## Step 1: Copy the activation code

1. Open the activation screen on the offline computer, then open **Server & offline options**.
2. Press **Copy activation code**. It is shown under **Activation code**.
3. If no code is shown, press **Copy Machine ID** instead.
4. Paste the code into a text file and save it on the USB stick.

The activation code identifies this computer. It also lets your virtual machines use this license later.

## Step 2: Get the license file

1. On the other computer, open the offline activation page at `license.scadabench.com/activate`.
2. Type your license key.
3. Type the activation code (or the Machine ID) from the text file.
4. Download the file named `license.cert`.
5. Copy it to the USB stick.

## Step 3: Load the file

1. Copy `license.cert` to the offline computer.
2. Open the activation screen, then **Server & offline options**.
3. Press **Load license file** and pick `license.cert`.

You can also open the file in a text editor, copy all its text, paste it under **Or paste the license file text** and press **Activate offline**.

When it works, the status reads **Lifetime license**.

## Notes

- The license file works only on the computer whose code you copied.
- Keep the purchase email and the license file safe.
