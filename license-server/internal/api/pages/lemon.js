// Lemon Squeezy card checkout on the buy page (ADR-0126). The overlay comes from
// Lemon Squeezy's own script; when the payment succeeds the page asks this server for
// the key of that order until the order's webhook has arrived. Without the overlay the
// hosted checkout opens instead and the key arrives by email.
const lemonButton = document.getElementById("lemon-pay");

function findIdentifier(value, depth = 0) {
  if (!value || typeof value !== "object" || depth > 6) return "";
  if (typeof value.identifier === "string" && value.identifier.length >= 32) return value.identifier;
  for (const child of Object.values(value)) {
    const found = findIdentifier(child, depth + 1);
    if (found) return found;
  }
  return "";
}

async function waitForKey(identifier) {
  show("status", "Payment received. Getting your license key…", "");
  for (const seconds of [2, 3, 4, 5, 6, 8, 10, 12, 15, 20]) {
    await new Promise((resolve) => setTimeout(resolve, seconds * 1000));
    try {
      const response = await fetch("/api/v1/checkout/lemon-key?order=" + encodeURIComponent(identifier));
      const data = await response.json();
      if (response.ok && data.license_key) { showKey(data); return; }
    } catch (error) { /* ask again */ }
  }
  show("status", "Your payment went through. Your license key is on its way by email; it can take a few minutes.", "ok");
}

function paid(event) {
  if (event.event !== "Checkout.Success") return;
  window.LemonSqueezy.Url.Close();
  const identifier = findIdentifier(event.data);
  if (identifier) waitForKey(identifier);
  else show("status", "Your payment went through. Your license key is on its way by email.", "ok");
}

lemonButton.onclick = async () => {
  lemonButton.disabled = true;
  try {
    const { url } = await postJSON("/api/v1/checkout/lemon-start", { hwid, news: wantsNews() });
    if (typeof window.createLemonSqueezy === "function") window.createLemonSqueezy();
    if (window.LemonSqueezy) {
      window.LemonSqueezy.Setup({ eventHandler: paid });
      window.LemonSqueezy.Url.Open(url + "&embed=1");
    } else {
      location.href = url;
    }
  } catch (error) {
    show("status", error.message, "bad");
  } finally {
    lemonButton.disabled = false;
  }
};
