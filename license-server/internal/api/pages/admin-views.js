"use strict";
// Owner page views: Overview, Licenses and Purchases.
(() => {
  const { api, el, risky, toast, busy, when, copy, views, refresh } = Owner;

  const revenue = (map) => Object.entries(map || {}).map(([currency, amount]) =>
    currency === "USD" ? "$" + amount : amount + " " + currency).join(" + ") || "$0.00";

  function stat(label, value, sub) {
    return el("div", { class: "card stat" }, el("span", { class: "label", text: label }),
      el("span", { class: "value", text: String(value) }), sub ? el("span", { class: "sub", text: sub }) : null);
  }

  // Price box: the new price shows on the buy page and scadabench.com within a minute.
  function priceCard(current) {
    const input = el("input", { id: "new-price", inputmode: "decimal", placeholder: "149.00", autocomplete: "off" });
    const button = el("button", { class: "btn", type: "submit", text: "Change price" });
    const form = el("form", { onsubmit: (event) => {
      event.preventDefault();
      busy(button, async () => {
        const result = await risky("Change the price from " + current + " to " + input.value + "?", "POST", "price",
          { price: input.value });
        if (result) { toast("Price is now " + result.amount); refresh(); }
      });
    } },
    el("div", {}, el("label", { for: "new-price", text: "New price in USD" }), input),
    el("div", { class: "row" }, button));
    return el("div", { class: "card" }, el("h2", { text: "Price: " + current }),
      el("p", { class: "hint", text: "One price for the lifetime key. It changes the buy page, PayPal and scadabench.com " +
        "within a minute. Buyers already inside PayPal still pay the price they saw." }), form);
  }

  // Versions: what the website offers vs. what the daily update check tells installed apps.
  function versionsCard() {
    const body = el("div", { class: "stack" }, el("p", { class: "muted", text: "Checking versions…" }));
    const line = (label, value) => el("div", { class: "meta" }, el("span", { text: label }), el("b", { text: value }));
    api("GET", "versions").then((v) => {
      const release = v.release;
      body.replaceChildren(
        line("Website offers", v.website || v.website_problem || "unknown"),
        line("Update check tells installed apps", release ? release.version + " (published " + release.date + ")" :
          "nothing published yet"),
        release && release.min_version ? line("Older than this gets “Important update”", release.min_version) : null,
        el("span", { class: "pill " + (v.in_step ? "ok" : "wait"), text: v.in_step ?
          "In step: installed apps learn about " + v.website + " within a day" :
          "Installed apps are not told about the newest version yet" }),
        v.in_step ? null : el("p", { class: "hint", text: "Fix: run website/deploy.cmd (it publishes the update " +
          "notice at its end), or owner.ps1 -Action release in license-server/deploy." }),
        line("License server program built", when(v.server_built) + " · " + v.go),
        el("p", { class: "hint", text: "Which version each customer runs is not visible: the app sends nothing " +
          "about itself when it checks (privacy by design)." }));
    }).catch((error) => body.replaceChildren(el("p", { class: "msg bad", text: error.message })));
    return el("div", { class: "card" }, el("h2", { text: "Versions" }), body);
  }

  views.overview = async () => {
    const data = await api("GET", "overview");
    const keys = data.licenses;
    const used = Math.min(100, (data.data_mb / Math.max(1, data.data_budget_mb)) * 100);
    const status = el("div", { class: "row" },
      el("span", { class: "pill " + (data.checkout ? "ok" : "bad"),
        text: data.checkout ? "PayPal checkout on (" + data.paypal_mode + ")" : "PayPal checkout off" }),
      el("span", { class: "pill " + (data.mail ? "ok" : "bad"), text: data.mail ? "Email on" : "Email off" }),
      null);
    const recent = el("div", { class: "list" }, (data.recent || []).map((entry) =>
      el("div", { class: "meta" }, el("span", { text: when(entry.time) }),
        el("span", { text: (entry.ok ? "✓ " : "✕ ") + entry.action + (entry.detail ? " · " + entry.detail : "") }),
        entry.ip ? el("span", { text: entry.ip }) : null)));
    return el("div", { class: "stack" },
      status, versionsCard(), priceCard(data.price),
      el("div", { class: "grid" },
        stat("Sales today", data.sales.today.sales, revenue(data.sales.today.revenue)),
        stat("Sales this month", data.sales.month.sales, revenue(data.sales.month.revenue)),
        stat("All sales", data.sales.all.sales, revenue(data.sales.all.revenue) +
          (data.sales.all.refunds ? " · " + data.sales.all.refunds + " refunded" : "")),
        stat("Keys in use", keys.active || 0, (keys.waiting || 0) + " not activated yet · " + (keys.revoked || 0) + " cancelled"),
        stat("Keys you made", keys.hand_made || 0, (keys.sold || 0) + " sold through PayPal"),
        stat("Free trials started", data.trials, "one per computer"),
        stat("News subscribers", data.subscribers, "people who agreed to get news")),
      el("div", { class: "card" },
        el("h2", { text: "Server storage" }),
        el("div", { class: "meter" + (used > 80 ? " high" : "") }, el("i", { style: "width:" + used.toFixed(1) + "%" })),
        el("p", { class: "hint", text: Owner.bytes(data.data_mb * 1048576) + " of " + data.data_budget_mb +
          " MB used. Above the limit, new free trials pause; sales keep working. See Disk for details." })),
      el("div", { class: "card" }, el("h2", { text: "Latest owner activity" }), recent,
        el("p", { class: "hint", text: "Every sign-in and change is recorded. See Activity for the full list." })));
  };

  function licenseState(license) {
    if (license.revoked) return el("span", { class: "pill bad", text: "Cancelled" + (license.reason ? ": " + license.reason : "") });
    if (license.hwid) return el("span", { class: "pill ok", text: "Active on a PC" });
    return el("span", { class: "pill wait", text: "Not activated yet" });
  }

  function licenseCard(license) {
    const action = (label, work, tone) => el("button", { class: "btn small " + (tone || "ghost"), type: "button",
      text: label, onclick: (event) => busy(event.currentTarget, work) });
    const buttons = [action("Copy key", () => copy(license.key))];
    if (!license.revoked && license.email) buttons.push(action("Email key again", async () => {
      await api("POST", "licenses/resend", { key: license.key });
      toast("Key emailed to " + license.email);
    }));
    buttons.push(action("Edit note", async () => {
      const note = prompt("Your note for this key (only you see it):", license.note || "");
      if (note === null) return;
      await api("POST", "licenses/note", { key: license.key, note });
      toast("Note saved"); refresh();
    }));
    if (!license.revoked) buttons.push(action("Cancel key", async () => {
      const reason = prompt("Why cancel this key? (for your records, e.g. refund)", "owner");
      if (reason === null) return;
      await api("POST", "licenses/revoke", { key: license.key, reason });
      toast("Key cancelled. It stops working at its next online check."); refresh();
    }, "danger"));
    if (!license.order_id) buttons.push(action("Delete", async () => {
      const done = await risky("Delete this key for good? Use this only for test keys.", "POST", "licenses/delete",
        { key: license.key });
      if (done) { toast("Key deleted"); refresh(); }
    }, "danger"));
    return el("div", { class: "item" },
      el("div", { class: "row spread" }, el("span", { class: "key", text: license.key }), licenseState(license)),
      el("div", { class: "meta" },
        el("span", { text: license.email || "No email" }),
        license.note ? el("span", { text: "Note: " + license.note }) : null,
        el("span", { text: (license.order_id ? "Sold " : "Made by you ") + when(license.created_at) }),
        license.hwid ? el("span", { text: "Activated " + when(license.activated_at) }) : null),
      el("div", { class: "row" }, buttons));
  }

  function makeKeyCard(reload) {
    const email = el("input", { id: "new-email", type: "email", autocomplete: "off", placeholder: "friend@example.com" });
    const note = el("input", { id: "new-note", maxlength: "120", placeholder: "Ahmed, friend" });
    const send = el("input", { id: "new-send", type: "checkbox" });
    const result = el("div", { class: "stack", hidden: true });
    const button = el("button", { class: "btn", type: "submit", text: "Make key" });
    const form = el("form", { onsubmit: (event) => {
      event.preventDefault();
      busy(button, async () => {
        const data = await api("POST", "licenses/issue", { email: email.value, note: note.value, send_email: send.checked });
        result.hidden = false;
        result.replaceChildren(el("div", { class: "bigkey", text: data.license.key }),
          el("div", { class: "row" }, el("button", { class: "btn ghost", type: "button", text: "Copy key",
            onclick: () => copy(data.license.key) })),
          el("p", { class: "msg ok", text: data.emailed ? "Emailed to " + data.license.email + "." :
            "Key made. Copy it and send it to them yourself." + (send.checked ? " (The email could not be sent.)" : "") }));
        email.value = ""; note.value = ""; send.checked = false;
        await reload();
      });
    } },
    el("div", {}, el("label", { for: "new-email", text: "Their email (optional)" }), email),
    el("div", {}, el("label", { for: "new-note", text: "Note for you (optional)" }), note),
    el("label", { class: "check" }, send, el("span", { text: "Email the key to them with activation steps" })),
    el("div", { class: "row" }, button));
    return el("div", { class: "card" }, el("h2", { text: "Make a lifetime key" }),
      el("p", { class: "hint", text: "For friends, testers or buyers who paid another way. The key works on one PC " +
        "and cannot be moved to another PC. Up to 25 keys a day, as a safety limit." }), form, result);
  }

  views.licenses = async () => {
    const search = el("input", { type: "search", placeholder: "Search by key, email or note", "aria-label": "Search keys" });
    const list = el("div", { class: "list" });
    const count = el("p", { class: "hint" });
    async function load() {
      const data = await api("GET", "licenses?q=" + encodeURIComponent(search.value));
      count.textContent = data.total + " key" + (data.total === 1 ? "" : "s") +
        (data.total > data.licenses.length ? " (showing the newest " + data.licenses.length + ")" : "");
      list.replaceChildren(...(data.licenses.length ? data.licenses.map(licenseCard) :
        [el("p", { class: "muted", text: "No keys match." })]));
    }
    let timer = 0;
    search.addEventListener("input", () => { clearTimeout(timer); timer = setTimeout(() => load().catch((e) => toast(e.message, "bad")), 300); });
    await load();
    return el("div", { class: "stack" }, makeKeyCard(load), el("div", { class: "card" },
      el("h2", { text: "All keys" }), search, count, list));
  };

  views.purchases = async () => {
    const data = await api("GET", "purchases");
    const words = { paid: "Paid", refunded: "Refunded", reversed: "Charged back" };
    const rows = data.records.map((record) => el("tr", {},
      el("td", { text: when(record.time) }),
      el("td", {}, el("span", { class: "pill " + (record.event === "paid" ? "ok" : "bad"), text: words[record.event] || record.event }),
        record.mode && record.mode !== "live" ? el("span", { class: "pill info", text: "test" }) : null),
      el("td", { class: "num", text: record.amount + " " + record.currency }),
      el("td", { text: [record.name, record.email, record.country].filter(Boolean).join(" · ") || "—" }),
      el("td", { class: "mono", text: record.license_key || "—" })));
    return el("div", { class: "card" }, el("h2", { text: "Purchases and refunds" }),
      el("p", { class: "hint", text: "Every PayPal payment, refund and chargeback, newest first. Refunds cancel the key automatically." }),
      rows.length ? el("table", { class: "wide" }, el("thead", {}, el("tr", {}, ["When", "What", "Amount", "Buyer", "Key"]
        .map((title) => el("th", { text: title })))), el("tbody", {}, rows)) :
        el("p", { class: "muted", text: "No purchases yet." }));
  };
})();
