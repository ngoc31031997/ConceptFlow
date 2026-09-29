#!/usr/bin/env python3
"""Prints the "Graph impact" section of the review brief (scripts/review-prep.sh).

    git diff --name-only <base> HEAD | graph_impact.py --graph graphify-out/graph.json --head <sha>

For every changed file, lists the files outside the diff that directly depend on
it (import, call, reference, inheritance...) according to the graphify code graph,
so the review agents know which callers to check for regressions.

The brief is trusted verbatim by record_review.py, so only repository paths go
into it: every listed path must be tracked by git and made of plain path
characters. Node labels and other text taken from the code are never printed.

When the graph is missing, unreadable or was not built at HEAD, the section says
so instead of listing anything: a stale graph would name the wrong callers.
Stdlib only, like the hooks.
"""
import argparse
import json
import re
import subprocess
import sys

# Edge relations meaning "source depends on target" (the set `graphify affected`
# traverses). Structural edges (contains, method, defines, rationale_for) are not
# dependencies.
DEPENDENCY_RELATIONS = frozenset({
    "calls", "indirect_call", "references", "imports", "imports_from",
    "dynamic_import", "re_exports", "inherits", "extends", "implements",
    "uses", "mixes_in", "embeds", "requires",
})
SAFE_PATH = re.compile(r"^[A-Za-z0-9._@+/-]+$")
MAX_PER_FILE = 15
HEADER = "Graph impact (graphify code graph; files outside the diff that directly depend on a changed file):"


def unavailable(reason):
    return f"Graph impact: not available ({reason})."


def tracked_files():
    out = subprocess.run(["git", "ls-files", "-z"], capture_output=True, text=True, check=True).stdout
    return {p for p in out.split("\0") if p}


def go_package(path):
    """The package directory of a non-test Go file ("a/b/" for "a/b/c.go"), else None."""
    if not path.endswith(".go") or path.endswith("_test.go"):
        return None
    return path.rsplit("/", 1)[0] + "/" if "/" in path else "./"


def dependents(graph, changed):
    """Maps each changed file, and each Go package with a changed non-test file,
    to the set of other files with a dependency edge into it.

    A Go import names a whole package, but graphify attaches it to one arbitrary
    file of that package (sometimes a _test.go), so Go import edges are counted
    against the package directory, keyed "pkg:<dir>".
    """
    file_of = {n.get("id"): n.get("source_file") or "" for n in graph.get("nodes", [])}
    changed_packages = {go_package(p) for p in changed} - {None}
    result = {}
    for edge in graph.get("links", graph.get("edges", [])):
        relation = edge.get("relation")
        # INFERRED edges are graphify's guesses by name (e.g. a doc heading that
        # mentions "node" linked to a node() function); only statically extracted
        # edges go into the brief.
        if relation not in DEPENDENCY_RELATIONS or edge.get("confidence", "EXTRACTED") != "EXTRACTED":
            continue
        target_file = file_of.get(edge.get("target"), "")
        source_file = file_of.get(edge.get("source")) or edge.get("source_file") or ""
        if not source_file or source_file == target_file:
            continue
        if relation in ("imports", "imports_from") and target_file.endswith(".go"):
            package = target_file.rsplit("/", 1)[0] + "/" if "/" in target_file else "./"
            if package in changed_packages and not source_file.startswith(package):
                result.setdefault("pkg:" + package, set()).add(source_file)
        elif target_file in changed:
            result.setdefault(target_file, set()).add(source_file)
    return result


def render(graph, head, changed, tracked):
    built = graph.get("built_at_commit") or ""
    if built != head:
        return unavailable(f"graph built at {built[:12] or 'unknown commit'}, HEAD is {head[:12]}")
    changed = [p for p in changed if SAFE_PATH.match(p)]
    changed_set = set(changed)
    deps = dependents(graph, changed_set)
    packages = sorted({go_package(p) for p in changed} - {None})
    keys = [(p, p) for p in changed] + [("pkg:" + d, f"Go package {d} (imported as a whole)") for d in packages]
    lines = [HEADER]
    for key, title in keys:
        users = sorted(p for p in deps.get(key, ()) if p in tracked and p not in changed_set and SAFE_PATH.match(p))
        if not users:
            continue
        shown = ", ".join(users[:MAX_PER_FILE])
        more = f" (+{len(users) - MAX_PER_FILE} more)" if len(users) > MAX_PER_FILE else ""
        lines.append(f"- {title} <- {len(users)} file(s): {shown}{more}")
    if len(lines) == 1:
        lines.append("- no dependents outside the diff found for the changed files")
    return "\n".join(lines)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--graph", required=True)
    parser.add_argument("--head", required=True, help="full commit hash of HEAD")
    args = parser.parse_args(argv)
    changed = [line.strip() for line in sys.stdin if line.strip()]
    try:
        with open(args.graph, encoding="utf-8") as fh:
            graph = json.load(fh)
    except FileNotFoundError:
        print(unavailable(f"no {args.graph}; run `make graph`"))
        return 0
    except (OSError, ValueError) as exc:
        print(unavailable(f"cannot read {args.graph}: {type(exc).__name__}"))
        return 0
    print(render(graph, args.head, changed, tracked_files()))
    return 0


if __name__ == "__main__":
    sys.exit(main())
