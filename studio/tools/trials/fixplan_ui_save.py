"""Fix plan A1 and A2 live, in the real app page (headless Chrome over CDP).

A1  Drag a breaker, press Ctrl+S and drag again while that save and its project read are in
    flight, ROUNDS times. When the page says "All changes saved", the engine's drawing must be
    exactly what the page shows (every node position).
A2  (a) Every diagram save fails as if the engine were unreachable (CDP Fetch fails the PUT):
    after a drag the page says "Not saved" and keeps the moved breaker; when the engine answers
    again, Ctrl+S saves it. (b) The engine process is ended: a drag still stays on screen and the
    page says "Not saved".

Usage: python studio/tools/trials/fixplan_ui_save.py [--rounds 10]
"""
import argparse
import random
import time

from fixplan_cdp import Browser
from fixplan_engine import FixEngine, check, finish

PORT, BROWSER_PORT = 23020, 23902
STATUS = "(document.querySelector('.topbar span.small') || {}).textContent || ''"
POSITIONS = ("Object.fromEntries([...document.querySelectorAll('.react-flow__node')].map(n => {"
             "const m = /translate\\(([-\\d.]+)px, ?([-\\d.]+)px\\)/.exec(n.style.transform);"
             "return [n.dataset.id, [Math.round(+m[1]), Math.round(+m[2])]];}))")


def layout():
    nodes = [{"id": f"cb{index}", "type": "equipment", "position": {"x": 220 * index, "y": 0},
              "data": {"label": f"CB {index}", "kind": "CB", "rotation": 0}} for index in range(3)]
    return {"nodes": nodes, "edges": []}


def engine_positions(engine):
    nodes = engine.project()["diagram"]["nodes"]
    return {item["id"]: [round(item["position"]["x"]), round(item["position"]["y"])] for item in nodes}


def centre(browser, node_id):
    box = browser.evaluate(f"(() => {{const r = document.querySelector('.react-flow__node[data-id=\"{node_id}\"]')"
                           ".getBoundingClientRect(); return [r.x + r.width / 2, r.y + 20];})()")
    return box[0], box[1]


def drag_any(browser, dx, dy):
    """Drag the first breaker whose centre is really under the mouse (earlier rounds may have moved
    one under the toolbar); returns the positions after the drag."""
    before = browser.evaluate(POSITIONS)
    for node_id in ("cb0", "cb1", "cb2"):
        x, y = centre(browser, node_id)
        if browser.evaluate(f"(document.elementFromPoint({x}, {y}) || document.body).closest("
                            f"'.react-flow__node')?.dataset.id === '{node_id}'"):
            browser.drag((x, y), (x + dx, y + dy))
            moved = browser.evaluate(POSITIONS)
            if moved != before:
                return moved
    raise RuntimeError("no breaker could be dragged")


def open_page(engine, browser):
    browser.call("Page.enable")
    browser.call("Page.navigate", url=f"{engine.url}/#session={engine.token}")
    browser.wait("document.querySelector('.nav-item') ? 1 : 0", 30)
    skip = ("(() => { const b = [...document.querySelectorAll('button')].find(b => b.textContent.trim() === "
            "'Skip tour'); if (b) b.click(); return !!b; })()")
    browser.wait(skip, 10)  # the welcome tour opens a moment after the page
    browser.wait(f"!{skip}", 5)
    browser.evaluate("[...document.querySelectorAll('.nav-item')].find(n => n.textContent.includes('SLD')).click()")
    browser.wait("document.querySelectorAll('.react-flow__node').length === 3", 20)
    x, y = centre(browser, "cb0")
    hit = f"!!(document.elementFromPoint({x}, {y}) || document.body).closest('.react-flow__node')"
    if not browser.wait(f"{skip} || {hit}", 10):  # a tour that opened late is skipped too
        raise RuntimeError("something covers the drawing: " + browser.evaluate(
            f"(document.elementFromPoint({x}, {y}) || document.body).outerHTML.slice(0, 200)"))
    browser.wait(hit, 5)


def settle(browser, words="All changes saved", timeout=10):
    return browser.wait(f"{STATUS} === '{words}' ? {STATUS} : ''", timeout)


def a1(engine, browser, rounds):
    for number in range(rounds):
        first, second = random.sample(["cb0", "cb1", "cb2"], 2)
        start = centre(browser, first)
        browser.drag(start, (start[0] + random.randint(-60, 60), start[1] + random.randint(30, 90)))
        browser.ctrl_s()
        time.sleep(random.uniform(0, 0.15))
        start = centre(browser, second)
        browser.drag(start, (start[0] + random.randint(-60, 60), start[1] + random.randint(-90, -30)))
        time.sleep(random.uniform(0, 0.3))
        browser.ctrl_s()
        saved = settle(browser)
        time.sleep(0.6)  # any late project read has arrived
        page, stored = browser.evaluate(POSITIONS), engine_positions(engine)
        check("A1", f"drag + Ctrl+S round {number + 1}", bool(saved) and page == stored,
              {"status": browser.evaluate(STATUS), "page": page, "engine": stored})


def handle_paused(browser):
    """Fail every paused diagram save as 'connection refused' (the engine is unreachable)."""
    browser.pump(0.05)
    failed = 0
    for event in [item for item in browser.events if item.get("method") == "Fetch.requestPaused"]:
        browser.events.remove(event)
        browser.call("Fetch.failRequest", requestId=event["params"]["requestId"], errorReason="ConnectionRefused")
        failed += 1
    return failed


def a2_unreachable(engine, browser):
    before = engine_positions(engine)
    browser.call("Fetch.enable", patterns=[{"urlPattern": "*/api/v1/project/diagram*", "requestStage": "Request"}])
    moved = drag_any(browser, 80, 60)
    failed, deadline, status = 0, time.time() + 6, ""
    while time.time() < deadline and status != "Not saved":
        failed += handle_paused(browser)
        status = browser.evaluate(STATUS)
    for _ in range(20):  # 2 s more: nothing puts the old drawing back
        failed += handle_paused(browser)
    kept = browser.evaluate(POSITIONS)
    stored = engine_positions(engine)
    check("A2", "save fails: edit kept, 'Not saved'", status == "Not saved" and kept == moved and stored == before
          and moved != before, {"status": status, "failed_saves": failed, "moved": moved, "on_screen_after_2s": kept,
                                "engine": stored})
    browser.call("Fetch.disable")
    browser.ctrl_s()
    saved = settle(browser)
    time.sleep(0.6)
    page, stored = browser.evaluate(POSITIONS), engine_positions(engine)
    check("A2", "engine answers again: Ctrl+S saves the kept edit", bool(saved) and page == stored == kept,
          {"status": browser.evaluate(STATUS), "page": page, "engine": stored})


def a2_engine_gone(engine, browser):
    engine.kill()
    moved = drag_any(browser, -70, 50)
    status = browser.wait(f"{STATUS} === 'Not saved' ? 'Not saved' : ''", 8) or browser.evaluate(STATUS)
    time.sleep(2)
    kept = browser.evaluate(POSITIONS)
    banner = browser.evaluate("document.body.innerText.includes('Failed to fetch') ? 'Failed to fetch' : ''")
    check("A2", "engine process ended: edit kept, 'Not saved'", status == "Not saved" and kept == moved,
          {"status": status, "moved": moved, "on_screen_after_2s": kept, "error_text": banner})


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--rounds", type=int, default=10)
    options = parser.parse_args()
    engine = FixEngine(PORT).start()
    browser = None
    try:
        project = engine.wait_project()
        engine.json("PUT", "/api/v1/project/diagram", {"diagram": layout(), "revision": project["revision"],
                                                        "tab_id": "fixplan"})
        browser = Browser(BROWSER_PORT)
        open_page(engine, browser)
        a1(engine, browser, options.rounds)
        a2_unreachable(engine, browser)
        a2_engine_gone(engine, browser)
    finally:
        if browser:
            browser.close()
        engine.close()
    return finish("fixplan_ui_save")


if __name__ == "__main__":
    raise SystemExit(main())
