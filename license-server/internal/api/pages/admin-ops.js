"use strict";
// Owner page views: Issues, Disk, Email (news) and Activity.
(() => {
  const { api, el, risky, toast, busy, when, bytes, views, refresh } = Owner;

  const kindWords = {
    internal_error: "Server problem", paypal_error: "PayPal did not answer", mail_error: "Email not sent",
    owner_mail_error: "Sale email to you not sent", webhook_rejected: "Fake or broken PayPal message refused",
    webhook_error: "PayPal message not handled", paypal_amount_mismatch: "Paid amount did not match the price",
    template_error: "Page could not be drawn", backup_error: "Backup failed", data_budget_reached: "Storage limit reached",
    release_unreadable: "Update notice unreadable", sales_record_error: "Payment record not saved",
    audit_error: "Activity record not saved", subscribe_error: "News sign-up not saved",
  };

  views.issues = async () => {
    const search = el("input", { type: "search", placeholder: "Search the log", "aria-label": "Search the log" });
    const everything = el("input", { type: "checkbox", id: "all-events" });
    const summary = el("div", { class: "row" });
    const list = el("div", { class: "list" });
    async function load() {
      const data = await api("GET", "issues?all=" + (everything.checked ? "1" : "0") + "&q=" + encodeURIComponent(search.value));
      const counts = Object.entries(data.problem_counts);
      summary.replaceChildren(...(counts.length ? counts.map(([kind, count]) =>
        el("span", { class: "pill bad", text: count + " × " + (kindWords[kind] || kind) })) :
        [el("span", { class: "pill ok", text: "No problems in the server log" })]));
      list.replaceChildren(...(data.events.length ? data.events.map((event) => {
        const { ts, kind, ...rest } = event;
        const details = Object.entries(rest).map(([key, value]) => key + ": " +
          (typeof value === "object" ? JSON.stringify(value) : value)).join("  ·  ");
        return el("div", { class: "item" },
          el("div", { class: "row spread" }, el("b", { text: kindWords[kind] || kind }),
            el("span", { class: "hint", text: ts ? new Date(ts).toLocaleString() : "" })),
          details ? el("div", { class: "details", text: details }) : null);
      }) : [el("p", { class: "muted", text: "Nothing to show." })]));
    }
    let timer = 0;
    search.addEventListener("input", () => { clearTimeout(timer); timer = setTimeout(() => load().catch((e) => toast(e.message, "bad")), 300); });
    everything.addEventListener("change", () => load().catch((e) => toast(e.message, "bad")));
    await load();
    return el("div", { class: "stack" }, el("div", { class: "card" }, el("h2", { text: "Issues" }),
      el("p", { class: "hint", text: "Problems the server noticed, newest first. License keys in the log are shortened " +
        "(SUBS-****-…-ABCD), so nothing secret is shown." }),
      summary, search, el("label", { class: "check" }, everything, el("span", { text: "Show every event, not only problems" }))), list);
  };

  function meter(part, whole) {
    const used = Math.min(100, (part / Math.max(1, whole)) * 100);
    return el("div", { class: "meter" + (used > 80 ? " high" : "") }, el("i", { style: "width:" + used.toFixed(1) + "%" }));
  }

  views.disk = async () => {
    const data = await api("GET", "disk");
    const removable = data.files.filter((file) => file.removable);
    const clean = el("button", { class: "btn danger", type: "button", text: "Remove older server logs",
      disabled: removable.length === 0, onclick: (event) => busy(event.currentTarget, async () => {
        const preview = await api("POST", "disk/clean", { preview: true });
        if (!preview.files.length) { toast("Nothing to remove"); return; }
        const done = await risky("Remove " + preview.files.length + " older log file(s) and free " + bytes(preview.bytes) +
          "? " + preview.files.join(", "), "POST", "disk/clean", { preview: false });
        if (done) { toast("Freed " + bytes(done.bytes)); refresh(); }
      }) });
    const rows = data.files.map((file) => el("tr", {}, el("td", { class: "mono", text: file.name }),
      el("td", { text: file.what }), el("td", { class: "num", text: bytes(file.bytes) })));
    const disk = data.disk_total ? el("div", { class: "card" }, el("h2", { text: "Whole server disk" }),
      meter(data.disk_total - data.disk_free, data.disk_total),
      el("p", { class: "hint", text: bytes(data.disk_free) + " free of " + bytes(data.disk_total) +
        ". The disk is shared with other software on this server; this page only touches ScadaBench files." })) : null;
    return el("div", { class: "stack" }, disk,
      el("div", { class: "card" }, el("h2", { text: "ScadaBench data folder" }), meter(data.data_bytes, data.budget_bytes),
        el("p", { class: "hint", text: bytes(data.data_bytes) + " of the " + bytes(data.budget_bytes) +
          " limit. Above the limit, new free trials pause; sales and activations keep working." }),
        el("table", { class: "wide" }, el("thead", {}, el("tr", {}, ["File", "What it is", "Size"].map((t) => el("th", { text: t })))),
          el("tbody", {}, rows)),
        el("p", { class: "hint", text: "Only older server logs can be removed here. The license database, signing key, " +
          "settings, payment record and activity record are never deleted." }),
        el("div", { class: "row" }, clean)));
  };

  views.email = async () => {
    const data = await api("GET", "subscribers");
    const subject = el("input", { id: "news-subject", maxlength: "150", placeholder: "ScadaBench 0.2 is out" });
    const body = el("textarea", { id: "news-body", placeholder: "Hi,\n\nScadaBench 0.2 adds …" });
    const progress = el("div", { class: "msg", hidden: true });
    const showJob = (job) => {
      if (!job || !job.started) return;
      progress.hidden = false;
      progress.textContent = (job.running ? "Sending “" + job.subject + "”: " : "Last send “" + job.subject + "”: ") +
        job.sent + " sent, " + job.failed + " failed, of " + job.total + (job.running ? "…" : ".");
    };
    async function poll() {
      const job = await api("GET", "news/status");
      showJob(job);
      if (job.running && document.body.contains(progress)) setTimeout(() => poll().catch(() => {}), 2000);
    }
    showJob(data.job);
    if (data.job.running) poll().catch(() => {});
    const test = el("button", { class: "btn ghost", type: "button", text: "Send a test to me",
      onclick: (event) => busy(event.currentTarget, async () => {
        const result = await api("POST", "news/test", { subject: subject.value, body: body.value });
        toast("Test sent to " + result.sent_to);
      }) });
    const send = el("button", { class: "btn", type: "button", text: "Send to " + data.active + " people",
      disabled: !data.ready || data.active === 0, onclick: (event) => busy(event.currentTarget, async () => {
        const job = await risky("Send “" + subject.value + "” to " + data.active + " people now? This cannot be undone.",
          "POST", "news/send", { subject: subject.value, body: body.value });
        if (job) { showJob(job); poll().catch(() => {}); }
      }) });
    const email = el("input", { id: "sub-email", type: "email", placeholder: "person@example.com" });
    const consent = el("input", { id: "sub-consent", type: "checkbox" });
    const add = el("button", { class: "btn ghost", type: "submit", text: "Add" });
    const people = data.subscribers.map((person) => el("div", { class: "item" },
      el("div", { class: "row spread" }, el("span", { text: person.email }),
        el("span", { class: "pill " + (person.unsubscribed_at ? "bad" : "ok"), text: person.unsubscribed_at ? "Unsubscribed" : "Subscribed" })),
      el("div", { class: "meta" }, el("span", { text: (person.source === "checkout" ? "Ticked the box at checkout " : "Added by you ") + when(person.consent_at) }),
        person.unsubscribed_at ? el("span", { text: "Left " + when(person.unsubscribed_at) }) : null),
      el("div", { class: "row" }, el("button", { class: "btn small danger", type: "button", text: "Remove completely",
        onclick: (event) => busy(event.currentTarget, async () => {
          if (!confirm("Remove " + person.email + " and its record?")) return;
          await api("POST", "subscribers/remove", { email: person.email }); toast("Removed"); refresh();
        }) }))));
    return el("div", { class: "stack" },
      el("div", { class: "card" }, el("h2", { text: "Write news" }),
        el("p", { class: "hint", text: "Goes only to people who agreed to get news. Every email gets a one-click unsubscribe " +
          "link at the bottom. Sender: " + (data.from || "not set") + "." }),
        data.ready ? null : el("p", { class: "msg bad", text: "Email is not set up on the server, so nothing can be sent yet." }),
        el("div", {}, el("label", { for: "news-subject", text: "Subject" }), subject),
        el("div", {}, el("label", { for: "news-body", text: "Message (plain text)" }), body),
        el("div", { class: "row" }, test, send), progress,
        el("p", { class: "hint", text: "Tip: send a test to " + (data.test_to || "yourself") + " first and read it on your phone." })),
      el("div", { class: "card" }, el("h2", { text: "People (" + data.active + " subscribed)" }),
        el("form", { onsubmit: (event) => { event.preventDefault(); busy(add, async () => {
          await api("POST", "subscribers/add", { email: email.value, consent: consent.checked });
          toast("Added"); refresh(); }); } },
        el("div", {}, el("label", { for: "sub-email", text: "Add a person who asked for news" }), email),
        el("label", { class: "check" }, consent, el("span", { text: "They agreed to get ScadaBench news by email" })),
        el("div", { class: "row" }, add)),
        el("div", { class: "list" }, people.length ? people : el("p", { class: "muted", text: "Nobody yet. Buyers can tick " +
          "“Email me about new versions” at checkout." }))));
  };

  views.activity = async () => {
    const data = await api("GET", "activity");
    const all = el("button", { class: "btn danger", type: "button", text: "Sign out all browsers",
      onclick: (event) => busy(event.currentTarget, async () => {
        if (!confirm("Sign out every browser, including this one?")) return;
        await api("POST", "logout-all", {});
        location.reload();
      }) });
    return el("div", { class: "card" }, el("h2", { text: "Owner activity" }),
      el("p", { class: "hint", text: data.sessions + " browser(s) signed in now. Every sign-in, failed try and change " +
        "is recorded here and cannot be edited from this page." }),
      el("div", { class: "row" }, all),
      el("table", { class: "wide" }, el("thead", {}, el("tr", {}, ["When", "What", "Result", "From"].map((t) => el("th", { text: t })))),
        el("tbody", {}, data.entries.map((entry) => el("tr", {}, el("td", { text: when(entry.time) }),
          el("td", { text: entry.action + (entry.detail ? " · " + entry.detail : "") }),
          el("td", {}, el("span", { class: "pill " + (entry.ok ? "ok" : "bad"), text: entry.ok ? "Done" : "Refused" })),
          el("td", { class: "mono", text: entry.ip || "—" }))))));
  };
})();
