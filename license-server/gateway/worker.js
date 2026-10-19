// Owner page gateway (ADR-0072). Cloudflare runs this only for the owner path on
// scadabench.com (the route is set at deploy time and is not written in this file).
// It forwards the request to the license server and adds the shared secret that
// proves the request came through here; the server answers 404 to anything else.
// Cloudflare Access, when set up, checks the owner's email before this code runs.
export default {
  async fetch(request, env) {
    // Trimmed: a secret piped in from Windows can carry a trailing line ending.
    const secret = (env.ADMIN_GATEWAY_SECRET || "").trim();
    if (secret.length < 32 || !env.UPSTREAM) {
      return new Response("Not available", { status: 503, headers: { "Cache-Control": "no-store" } });
    }
    const incoming = new URL(request.url);
    const target = new URL(incoming.pathname + incoming.search, env.UPSTREAM);
    const headers = new Headers(request.headers);
    headers.delete("X-Admin-Gateway");
    headers.delete("X-Admin-Client-IP");
    headers.set("X-Admin-Gateway", secret);
    headers.set("X-Admin-Client-IP", request.headers.get("CF-Connecting-IP") || "");
    const init = { method: request.method, headers, redirect: "manual" };
    if (request.method !== "GET" && request.method !== "HEAD") init.body = request.body;
    let response;
    try {
      response = await fetch(target, init);
    } catch (error) {
      return new Response("The license server is not reachable right now. Try again in a minute.",
        { status: 502, headers: { "Cache-Control": "no-store" } });
    }
    return new Response(response.body, response);
  },
};
