"""Check that the ScadaBench interface loads nothing from the internet (ADR-0133).

The page itself forbids it (Content-Security-Policy in httpapi/offline_policy.go, the same
rule in vite.config.ts for development). This check keeps the source honest too:

- every web address in the interface source is a link the user clicks, listed below;
- no stylesheet, font, script or image is loaded from an address (url(), @import, src=, CDN);
- both policies allow only the engine itself.
"""
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
FRONTEND = ROOT / "studio" / "frontend"
POLICY = ROOT / "studio" / "engine" / "internal" / "httpapi" / "offline_policy.go"
VITE = FRONTEND / "vite.config.ts"

# Pages the user may open by clicking (they open outside the app, nothing is loaded).
CLICK_LINKS = {
    "https://scadabench.com/tutorials/",  # Help: tutorials (copy/tour.ts)
    "https://scadabench.com/changelog/",  # What's new: all updates (copy/whatsNew.ts)
    "https://npcap.com/#download",  # GOOSE: the Npcap download (network/GooseReach.tsx)
}
ADDRESS = re.compile(r"https?://[^\s'\"`)<>]+")
LOADS = [
    re.compile(r"url\(\s*['\"]?https?:", re.IGNORECASE),
    re.compile(r"@import\s+(url\()?\s*['\"]?https?:", re.IGNORECASE),
    re.compile(r"\b(src|href)\s*=\s*['\"{]?\s*['\"`]?https?:", re.IGNORECASE),
    re.compile(r"\bimport\s*(\(|.*from\s*)['\"`]https?:", re.IGNORECASE),
    re.compile(r"fonts\.googleapis|fonts\.gstatic|cdnjs|jsdelivr|unpkg\.com|cdn\.tailwindcss", re.IGNORECASE),
]
SOURCES = ("*.ts", "*.tsx", "*.css", "*.html")


def _files():
    yield FRONTEND / "index.html"
    for pattern in SOURCES:
        yield from sorted((FRONTEND / "src").rglob(pattern))


def check():
    failures = []
    for path in _files():
        text = path.read_text(encoding="utf-8")
        where = path.relative_to(ROOT) if path.is_relative_to(ROOT) else path
        for number, line in enumerate(text.splitlines(), 1):
            for pattern in LOADS:
                if pattern.search(line):
                    failures.append(f"{where}:{number}: loads something from the internet: {line.strip()[:120]}")
            for address in ADDRESS.findall(line):
                if address not in CLICK_LINKS and not address.startswith("http://www.w3.org/"):
                    failures.append(f"{where}:{number}: web address {address} is not a listed click link "
                                    "(studio/tools/check_offline_ui.py)")
    policy = POLICY.read_text(encoding="utf-8")
    if "default-src 'self'" not in policy or re.search(r"https?:|\*", policy.split("const OfflinePolicy", 1)[-1]):
        failures.append(f"{POLICY.relative_to(ROOT)}: the page policy must allow only 'self'")
    vite = VITE.read_text(encoding="utf-8")
    rules = vite.split("const offlinePolicy", 1)[-1].split("].join", 1)[0]
    if ("'Content-Security-Policy': offlinePolicy" not in vite or "default-src 'self'" not in rules
            or re.search(r"https?:", rules)):
        failures.append(f"{VITE.name}: the development server must send the offline policy")
    return failures


if __name__ == "__main__":
    problems = check()
    print("\n".join(problems) or "Offline interface: PASS")
    raise SystemExit(bool(problems))
