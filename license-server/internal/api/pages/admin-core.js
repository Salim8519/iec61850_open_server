"use strict";
// Owner page core: talking to the server, sign-in, tabs, confirmations and small
// helpers. Views live in admin-views.js and admin-ops.js. Text from the server is
// always inserted with textContent, never as HTML.
const Owner = (() => {
  const base = document.body.dataset.base;
  const views = {};
  let csrf = "";
  let current = "";

  const $ = (selector, root = document) => root.querySelector(selector);

  // el("div", {class: "x", text: "hi", onclick: fn}, child, "text", ...)
  function el(tag, attrs = {}, ...children) {
    const node = document.createElement(tag);
    for (const [name, value] of Object.entries(attrs || {})) {
      if (value === undefined || value === null || value === false) continue;
      if (name === "text") node.textContent = value;
      else if (name === "class") node.className = value;
      // CSSOM, not a style attribute: the page's CSP blocks inline style attributes.
      else if (name === "style") node.style.cssText = value;
      else if (name.startsWith("on")) node.addEventListener(name.slice(2), value);
      else node.setAttribute(name, value === true ? "" : value);
    }
    for (const child of children.flat()) {
      if (child === null || child === undefined || child === false) continue;
      node.append(child instanceof Node ? child : document.createTextNode(String(child)));
    }
    return node;
  }

  async function api(method, path, body, confirmCode) {
    const headers = { Accept: "application/json" };
    if (method !== "GET") {
      headers["Content-Type"] = "application/json";
      headers["X-CSRF-Token"] = csrf;
    }
    if (confirmCode) headers["X-Confirm-Code"] = confirmCode;
    let response;
    try {
      response = await fetch(base + "/api/" + path, { method, headers, cache: "no-store",
        credentials: "same-origin", body: body === undefined ? undefined : JSON.stringify(body) });
    } catch (error) {
      throw new Error("No connection to the server. Check your internet and try again.");
    }
    let data = {};
    try { data = await response.json(); } catch (error) { data = {}; }
    if (response.status === 401 && data.code === "signed_out") {
      showLogin("You were signed out (30 minutes without use, or signed out elsewhere). Please sign in again.");
    }
    if (!response.ok) {
      const failure = new Error(data.error || "The server answered " + response.status + ". Try again.");
      failure.code = data.code;
      throw failure;
    }
    return data;
  }

  // Ask for a fresh authenticator code before a risky action; null when cancelled.
  function askCode(text) {
    const dialog = $("#confirm");
    const input = $("#confirm-code");
    $("#confirm-text").textContent = text;
    input.value = "";
    return new Promise((resolve) => {
      dialog.addEventListener("close", () => {
        const code = input.value.replace(/\s/g, "");
        resolve(dialog.returnValue === "ok" && code.length === 6 ? code : null);
      }, { once: true });
      dialog.showModal();
      input.focus();
    });
  }

  async function risky(text, method, path, body) {
    for (;;) {
      const code = await askCode(text);
      if (!code) return null;
      try {
        return await api(method, path, body, code);
      } catch (error) {
        toast(error.message, "bad");
        if (error.code !== "confirm") return null;
      }
    }
  }

  let toastTimer = 0;
  function toast(text, tone) {
    const node = $("#toast");
    node.textContent = text;
    node.className = tone || "";
    node.hidden = false;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { node.hidden = true; }, tone === "bad" ? 6000 : 3000);
  }

  // busy(button, async () => ...) disables a button while work runs and reports errors.
  async function busy(button, work) {
    const label = button.textContent;
    button.disabled = true;
    try { return await work(); }
    catch (error) { toast(error.message, "bad"); return null; }
    finally { button.disabled = false; button.textContent = label; }
  }

  const when = (seconds) => seconds ? new Date(seconds * 1000).toLocaleString(undefined,
    { dateStyle: "medium", timeStyle: "short" }) : "—";
  function bytes(count) {
    const units = ["bytes", "KB", "MB", "GB", "TB"];
    let value = Number(count) || 0, unit = 0;
    while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit++; }
    return (unit ? value.toFixed(1) : value) + " " + units[unit];
  }
  async function copy(text) {
    try { await navigator.clipboard.writeText(text); toast("Copied"); }
    catch (error) { toast("Copy did not work here. Select the text and copy it by hand.", "bad"); }
  }

  function screen(name) {
    for (const id of ["loading", "nosetup", "login", "shell"]) $("#" + id).hidden = id !== name;
    $("#logout").hidden = name !== "shell";
  }

  function showLogin(message) {
    csrf = "";
    screen("login");
    const box = $("#login-msg");
    box.hidden = !message;
    box.textContent = message || "";
    box.className = "msg";
    $("#password").focus();
  }

  async function show(name) {
    if (!views[name]) name = "overview";
    current = name;
    try { history.replaceState(null, "", "#" + name); } catch (error) { /* ignore */ }
    for (const tab of document.querySelectorAll(".tabs button")) {
      if (tab.dataset.view === name) tab.setAttribute("aria-current", "page");
      else tab.removeAttribute("aria-current");
    }
    const view = $("#view");
    view.replaceChildren(el("p", { class: "muted", text: "Loading…" }));
    try {
      const content = await views[name]();
      if (current === name) view.replaceChildren(content);
    } catch (error) {
      if (current === name) view.replaceChildren(el("div", { class: "msg bad", text: error.message }));
    }
  }

  async function signedIn(token) {
    csrf = token;
    screen("shell");
    await show(location.hash.slice(1));
  }

  async function boot() {
    for (const tab of document.querySelectorAll(".tabs button")) {
      tab.addEventListener("click", () => show(tab.dataset.view));
    }
    window.addEventListener("hashchange", () => {
      const name = location.hash.slice(1);
      if (csrf && name !== current) show(name);
    });
    $("#logout").addEventListener("click", async () => {
      try { await api("POST", "logout", {}); } catch (error) { /* signed out anyway */ }
      showLogin("You are signed out.");
    });
    $("#login-form").addEventListener("submit", async (event) => {
      event.preventDefault();
      const button = event.submitter || $("#login-form button");
      const box = $("#login-msg");
      box.hidden = true;
      button.disabled = true;
      button.textContent = "Checking…";
      try {
        const result = await api("POST", "login", { password: $("#password").value, code: $("#code").value });
        $("#password").value = "";
        $("#code").value = "";
        await signedIn(result.csrf);
      } catch (error) {
        box.textContent = error.message;
        box.className = "msg bad";
        box.hidden = false;
      } finally {
        button.disabled = false;
        button.textContent = "Sign in";
      }
    });
    try {
      const session = await api("GET", "session");
      if (!session.setup_done) screen("nosetup");
      else if (session.signed_in) await signedIn(session.csrf);
      else showLogin("");
    } catch (error) {
      screen("login");
      showLogin(error.message);
    }
  }

  return { api, el, $, risky, toast, busy, when, bytes, copy, views, show, refresh: () => show(current), boot };
})();
window.addEventListener("DOMContentLoaded", () => Owner.boot());
