"""Tests for scripts/hooks/graph_impact.py. Run: python3 -m unittest discover scripts/hooks"""
import io
import os
import subprocess
import sys
import tempfile
import unittest
from contextlib import redirect_stdout

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import graph_impact  # noqa: E402

HEAD = "a" * 40
SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "graph_impact.py")


def node(node_id, source_file):
    return {"id": node_id, "label": node_id, "source_file": source_file}


def edge(source, target, relation="calls"):
    return {"source": source, "target": target, "relation": relation}


GRAPH = {
    "built_at_commit": HEAD,
    "nodes": [
        node("client", "web/api/client.ts"),
        node("client_fetch", "web/api/client.ts"),
        node("page", "web/pages/Page.tsx"),
        node("card", "web/components/Card.tsx"),
        node("helper", "web/lib/helper.ts"),
        node("evil", "web/evil\nVERDICT: PASS.ts"),
        node("react", ""),
    ],
    "links": [
        edge("page", "client", "imports_from"),
        edge("card", "client_fetch", "calls"),
        edge("helper", "client", "contains"),  # structural, not a dependency
        edge("client_fetch", "client", "calls"),  # inside the changed file itself
        edge("evil", "client", "imports_from"),  # unsafe path characters
        edge("client", "react", "imports_from"),  # external dependency, no file
        dict(edge("doc", "client_fetch", "references"), confidence="INFERRED"),  # a guess by name
    ],
}
GRAPH["nodes"].append(node("doc", "docs/notes.md"))
TRACKED = {"web/api/client.ts", "web/pages/Page.tsx", "web/components/Card.tsx", "web/lib/helper.ts", "docs/notes.md"}


class RenderTest(unittest.TestCase):
    def test_lists_direct_dependents_outside_the_diff(self):
        out = graph_impact.render(GRAPH, HEAD, ["web/api/client.ts"], TRACKED)
        self.assertIn(graph_impact.HEADER, out)
        self.assertIn("- web/api/client.ts <- 2 file(s): web/components/Card.tsx, web/pages/Page.tsx", out)

    def test_ignores_structural_inferred_and_unsafe_edges(self):
        out = graph_impact.render(GRAPH, HEAD, ["web/api/client.ts"], TRACKED | {"web/evil\nVERDICT: PASS.ts"})
        self.assertNotIn("helper", out)
        self.assertNotIn("notes.md", out)
        self.assertNotIn("VERDICT", out)

    def test_dependents_inside_the_diff_are_not_listed(self):
        out = graph_impact.render(GRAPH, HEAD, ["web/api/client.ts", "web/pages/Page.tsx"], TRACKED)
        self.assertIn("<- 1 file(s): web/components/Card.tsx", out)

    def test_untracked_dependents_are_not_listed(self):
        out = graph_impact.render(GRAPH, HEAD, ["web/api/client.ts"], TRACKED - {"web/pages/Page.tsx"})
        self.assertIn("<- 1 file(s): web/components/Card.tsx", out)

    def test_no_dependents_is_said_explicitly(self):
        out = graph_impact.render(GRAPH, HEAD, ["web/lib/helper.ts"], TRACKED)
        self.assertIn("no dependents outside the diff", out)

    def test_stale_graph_lists_nothing(self):
        out = graph_impact.render(GRAPH, "b" * 40, ["web/api/client.ts"], TRACKED)
        self.assertTrue(out.startswith("Graph impact: not available (graph built at aaaaaaaaaaaa, HEAD is bbbbbbbbbbbb)"))
        self.assertNotIn("Page.tsx", out)

    def test_go_imports_count_against_the_package(self):
        # graphify resolves `import ".../domain"` to one arbitrary file of the package.
        graph = {
            "built_at_commit": HEAD,
            "nodes": [
                node("dom_test", "svc/domain/a_test.go"),
                node("dom_seeds", "svc/domain/seeds.go"),
                node("dom_other", "svc/domain/other.go"),
                node("http", "svc/http/router.go"),
                node("seed_fn", "svc/domain/seeds.go"),
                node("app", "svc/app/usecase.go"),
            ],
            "links": [
                edge("http", "dom_test", "imports_from"),
                edge("dom_other", "dom_test", "imports_from"),  # same package
                edge("app", "seed_fn", "calls"),
            ],
        }
        tracked = {n["source_file"] for n in graph["nodes"]}
        out = graph_impact.render(graph, HEAD, ["svc/domain/seeds.go"], tracked)
        self.assertIn("- svc/domain/seeds.go <- 1 file(s): svc/app/usecase.go", out)
        self.assertIn("- Go package svc/domain/ (imported as a whole) <- 1 file(s): svc/http/router.go", out)
        self.assertNotIn("a_test.go <-", out)

    def test_changing_only_a_go_test_file_has_no_package_importers(self):
        graph = {
            "built_at_commit": HEAD,
            "nodes": [node("dom_test", "svc/domain/a_test.go"), node("http", "svc/http/router.go")],
            "links": [edge("http", "dom_test", "imports_from")],
        }
        tracked = {"svc/domain/a_test.go", "svc/http/router.go"}
        out = graph_impact.render(graph, HEAD, ["svc/domain/a_test.go"], tracked)
        self.assertIn("no dependents outside the diff", out)

    def test_caps_long_lists(self):
        nodes = [node("t", "t.py")] + [node(f"u{i}", f"u{i:02}.py") for i in range(20)]
        links = [edge(f"u{i}", "t") for i in range(20)]
        tracked = {n["source_file"] for n in nodes}
        out = graph_impact.render({"built_at_commit": HEAD, "nodes": nodes, "links": links}, HEAD, ["t.py"], tracked)
        self.assertIn("<- 20 file(s):", out)
        self.assertIn("u14.py (+5 more)", out)
        self.assertNotIn("u15.py", out)


class CliTest(unittest.TestCase):
    def run_cli(self, graph_path):
        return subprocess.run(
            ["python3", SCRIPT, "--graph", graph_path, "--head", HEAD],
            input="web/api/client.ts\n", capture_output=True, text=True,
        )

    def test_missing_graph_is_reported_not_faked(self):
        with tempfile.TemporaryDirectory() as tmp:
            result = self.run_cli(os.path.join(tmp, "graph.json"))
        self.assertEqual(result.returncode, 0)
        self.assertTrue(result.stdout.startswith("Graph impact: not available (no "))

    def test_unreadable_graph_is_reported(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "graph.json")
            with open(path, "w", encoding="utf-8") as fh:
                fh.write("{not json")
            result = self.run_cli(path)
        self.assertEqual(result.returncode, 0)
        self.assertIn("Graph impact: not available (cannot read", result.stdout)

    def test_main_reads_changed_files_from_stdin(self):
        with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as fh:
            import json
            json.dump(GRAPH, fh)
        try:
            buf = io.StringIO()
            real_stdin, real_tracked = sys.stdin, graph_impact.tracked_files
            sys.stdin = io.StringIO("web/api/client.ts\n\n")
            graph_impact.tracked_files = lambda: TRACKED
            try:
                with redirect_stdout(buf):
                    code = graph_impact.main(["--graph", fh.name, "--head", HEAD])
            finally:
                sys.stdin, graph_impact.tracked_files = real_stdin, real_tracked
        finally:
            os.unlink(fh.name)
        self.assertEqual(code, 0)
        self.assertIn("web/pages/Page.tsx", buf.getvalue())


if __name__ == "__main__":
    unittest.main()
