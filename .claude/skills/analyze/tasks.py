#!/usr/bin/env python3
"""Render, read and update a task plan stored as JSON inside a self-contained tasks.html.

The plan lives in a <script type="application/json" id="plan"> block, so the HTML is both
the page the Creator reviews in the browser and the file /run-tasks reads back.

Usage:
  tasks.py render <plan.json> <out.html>      build tasks.html from a plan JSON
  tasks.py show <tasks.html>                  print the plan JSON
  tasks.py next <tasks.html>                  print ids of pending tasks whose deps are all done
  tasks.py set <tasks.html> <id> <field> <value>   set a task field (status, model, effort, note)
"""
import html
import json
import re
import sys
from pathlib import Path

MODELS = ["haiku", "sonnet", "opus", "fable"]
EFFORTS = ["low", "medium", "high", "xhigh"]
STATUSES = ["pending", "running", "done", "failed", "skipped"]
DEFAULT_MODEL = {1: "haiku", 2: "haiku", 3: "sonnet", 4: "opus", 5: "fable"}
DEFAULT_EFFORT = {1: "low", 2: "low", 3: "medium", 4: "medium", 5: "high"}
TEMPLATE = Path(__file__).with_name("template.html")
PLAN_RE = re.compile(r'(<script type="application/json" id="plan">)(.*?)(</script>)', re.S)


def fail(msg: str) -> None:
    print(f"ERROR: {msg}", file=sys.stderr)
    sys.exit(1)


def validate(plan: dict) -> dict:
    tasks = plan.get("tasks")
    if not isinstance(tasks, list) or not tasks:
        fail("plan has no tasks")
    ids = [t.get("id") for t in tasks]
    if len(set(ids)) != len(ids) or None in ids:
        fail("task ids must be present and unique")
    for t in tasks:
        for key in ("title", "goal", "difficulty"):
            if key not in t:
                fail(f"task {t['id']} is missing '{key}'")
        d = t["difficulty"]
        if d not in DEFAULT_MODEL:
            fail(f"task {t['id']} difficulty must be 1-5, got {d}")
        t.setdefault("model", DEFAULT_MODEL[d])
        t.setdefault("effort", DEFAULT_EFFORT[d])
        t["default_model"] = DEFAULT_MODEL[d]
        t.setdefault("depends_on", [])
        t.setdefault("files", [])
        t.setdefault("acceptance", [])
        t.setdefault("status", "pending")
        if t["model"] not in MODELS:
            fail(f"task {t['id']} model must be one of {MODELS}")
        if t["effort"] not in EFFORTS:
            fail(f"task {t['id']} effort must be one of {EFFORTS}")
        for dep in t["depends_on"]:
            if dep not in ids:
                fail(f"task {t['id']} depends on unknown task {dep}")
    return plan


def dump(plan: dict) -> str:
    # "</" is escaped so task text can never close the <script> block early.
    return json.dumps(plan, ensure_ascii=False, indent=2).replace("</", "<\\/")


def read(path: str) -> dict:
    m = PLAN_RE.search(Path(path).read_text(encoding="utf-8"))
    if not m:
        fail(f"{path} has no plan block")
    return json.loads(m.group(2))


def write(path: str, plan: dict) -> None:
    text = Path(path).read_text(encoding="utf-8")
    text = PLAN_RE.sub(lambda m: m.group(1) + "\n" + dump(plan) + "\n" + m.group(3), text, count=1)
    Path(path).write_text(text, encoding="utf-8")


def main() -> None:
    if len(sys.argv) < 3:
        fail(__doc__)
    cmd, args = sys.argv[1], sys.argv[2:]
    if cmd == "render":
        plan = validate(json.loads(Path(args[0]).read_text(encoding="utf-8")))
        page = TEMPLATE.read_text(encoding="utf-8")
        page = page.replace("{{TITLE}}", html.escape(plan.get("title", "Task plan")))
        page = page.replace("{{PLAN}}", dump(plan))
        Path(args[1]).write_text(page, encoding="utf-8")
        print(args[1])
    elif cmd == "show":
        print(json.dumps(read(args[0]), ensure_ascii=False, indent=2))
    elif cmd == "next":
        plan = read(args[0])
        status = {t["id"]: t["status"] for t in plan["tasks"]}
        ready = [t["id"] for t in plan["tasks"]
                 if t["status"] == "pending" and all(status[d] == "done" for d in t["depends_on"])]
        print(" ".join(ready))
    elif cmd == "set":
        if len(args) != 4:
            fail("usage: tasks.py set <tasks.html> <id> <field> <value>")
        _, tid, field, value = args
        allowed = {"status": STATUSES, "model": MODELS, "effort": EFFORTS, "note": None}
        if field not in allowed:
            fail(f"field must be one of {list(allowed)}")
        if allowed[field] and value not in allowed[field]:
            fail(f"{field} must be one of {allowed[field]}")
        plan = read(args[0])
        task = next((t for t in plan["tasks"] if t["id"] == tid), None)
        if not task:
            fail(f"unknown task {tid}")
        task[field] = value
        write(args[0], plan)
    else:
        fail(f"unknown command {cmd}")


if __name__ == "__main__":
    main()
