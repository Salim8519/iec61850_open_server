# ADR-0056: Verify the license server with Windows' own certificate check

Accepted: 2026-10-17. Owner report: the installed app said "Cannot reach the license
server. Check the internet connection, or use offline activation", also for the 7-day
trial, while the internet and the server were fine.

**Cause (evidence)**

- `https://license.scadabench.com/api/v1/health` answered 200 to curl and browsers.
- Through the app's client the error was `SSLCertVerificationError: certificate has
  expired`. The Cloudflare edge certificate was renewed on 2026-10-16 onto Let's
  Encrypt's new chain `scadabench.com → YE2 → Root YE → ISRG Root X2` (all valid).
- Python's OpenSSL only trusts roots already in the Windows store. This PC (and most
  PCs) has ISRG Root X1 as a root but ISRG Root X2 only as the old X1-signed
  cross-certificate in the intermediate store, **expired 2025-09-15**. OpenSSL built the
  chain through it and failed. Windows itself (CryptoAPI, used by Edge and curl) fetches
  missing roots on demand, so it succeeds.
- Every installed build (0.1.0–0.1.2) is affected for trial, activation and the weekly
  check; nothing is wrong with the network or the server.

**Decision**

- `licensing/client.py` opens HTTPS with `truststore.SSLContext` (`https_context()`),
  which verifies the server's chain with the Windows certificate verifier, exactly like
  the browser. Host name checking and `CERT_REQUIRED` stay on.
- New pinned dependency `truststore==0.10.5` (pure Python, MIT, maintained by a Python
  core developer; no native code). Added to `collect_imports.THIRD_PARTY` so the
  protected build bundles it (the packaging guard test enforces this).
- Rejected: shipping our own CA bundle (must be kept up to date by hand and breaks again
  when Cloudflare changes CA); disabling verification (unsafe).

**Compatibility and limits**

- Takes effect from the next build. Installed 0.1.x copies keep failing to reach the
  server until updated; their existing licenses keep working (an unreachable weekly
  check changes nothing). Until then, affected users can use the offline activation
  page in their browser.
- A possible server-side mitigation for old copies (switching the Cloudflare edge CA)
  is an outward production change and is not made here.

**Evidence**: change record [0080](../changes/2026/0080-windows-tls-verification.md).
