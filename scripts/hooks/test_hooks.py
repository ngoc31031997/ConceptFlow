"""Tests for the Claude Code hooks in scripts/hooks/. Run: python3 -m unittest discover scripts/hooks

Each git test builds a throwaway repository, so nothing touches this checkout.
Stdlib only, like the hooks themselves.
"""
import json
import os
import shutil
import subprocess
import tempfile
import unittest

HOOKS = os.path.dirname(os.path.abspath(__file__))
GUARD = os.path.join(HOOKS, "guard_bash.py")


def run_guard(command, cwd):
    payload = json.dumps({"tool_name": "Bash", "tool_input": {"command": command}, "cwd": cwd})
    result = subprocess.run(["python3", GUARD], input=payload, capture_output=True, text=True)
    return result.returncode, result.stderr


class SecretGuardTest(unittest.TestCase):
    def assertBlocked(self, command):
        code, err = run_guard(command, HOOKS)
        self.assertEqual(code, 2, f"expected block: {command!r}")
        self.assertIn("Blocked", err)

    def assertAllowed(self, command):
        code, err = run_guard(command, HOOKS)
        self.assertEqual(code, 0, f"expected allow: {command!r}: {err}")

    def test_blocks_env_files(self):
        for cmd in ["cat .env", "grep KEY ./.env", "head -3 /repo/.env", "cat .env.window",
                    "source .env && echo $X", 'python3 -c "open(\'.env\').read()"',
                    "docker compose --env-file .env up"]:
            self.assertBlocked(cmd)

    def test_allows_env_lookalikes(self):
        for cmd in ["cat .env.example", "cat services/web-gui/.env.test", "cat .envrc",
                    "git add .env.example", "echo $ENV", "grep -r env_file docker-compose.yml"]:
            self.assertAllowed(cmd)

    def test_blocks_secrets_dir_but_not_readme(self):
        self.assertBlocked("cat secrets/client.json")
        self.assertBlocked("ls secrets/")
        self.assertBlocked("cat ./secrets/foo && echo hi")
        self.assertAllowed("cat secrets/README.md")
        self.assertAllowed("grep -rn secrets_manager services/")

    def test_blocks_client_secret_json(self):
        self.assertBlocked("cp ~/Downloads/client_secret_123.json /tmp/x")

    def test_compose_config(self):
        self.assertBlocked("docker compose config")
        self.assertBlocked("docker compose -f docker-compose.yml config")
        self.assertAllowed("docker compose config --services")
        self.assertAllowed("docker compose ps")


def sh(cwd, *args):
    subprocess.run(args, cwd=cwd, check=True, capture_output=True)


class MergeGateTest(unittest.TestCase):
    def setUp(self):
        self.repo = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, self.repo)
        sh(self.repo, "git", "init", "-q", "-b", "main")
        sh(self.repo, "git", "config", "user.email", "t@example.com")
        sh(self.repo, "git", "config", "user.name", "t")
        self.commit("a.txt", "1")
        sh(self.repo, "git", "checkout", "-q", "-b", "feature/x")
        self.commit("b.txt", "2")

    def commit(self, name, content):
        with open(os.path.join(self.repo, name), "w") as f:
            f.write(content)
        sh(self.repo, "git", "add", name)
        sh(self.repo, "git", "commit", "-q", "-m", name)

    def mark(self, rev):
        tree = subprocess.run(["git", "rev-parse", rev + "^{tree}"], cwd=self.repo,
                              capture_output=True, text=True, check=True).stdout.strip()
        marks = os.path.join(self.repo, ".git", "conceptflow", "checked-trees")
        os.makedirs(marks, exist_ok=True)
        open(os.path.join(marks, tree), "w").close()

    def guard(self, command):
        return run_guard(command, self.repo)

    def test_merge_of_unchecked_branch_is_blocked(self):
        sh(self.repo, "git", "checkout", "-q", "main")
        code, err = self.guard("git merge --no-ff feature/x -m 'Merge'")
        self.assertEqual(code, 2)
        self.assertIn("merge gate", err)

    def test_merge_of_checked_branch_is_allowed(self):
        self.mark("feature/x")
        sh(self.repo, "git", "checkout", "-q", "main")
        self.assertEqual(self.guard("git merge --no-ff feature/x -m 'Merge'")[0], 0)

    def test_checkout_main_then_merge_in_one_line_is_gated(self):
        code, _ = self.guard("git checkout main && git pull -q origin main && git merge feature/x")
        self.assertEqual(code, 2)

    def test_merge_on_feature_branch_is_not_gated(self):
        sh(self.repo, "git", "checkout", "-q", "-b", "feature/y", "main")
        self.assertEqual(self.guard("git merge main")[0], 0)
        self.assertEqual(self.guard("git merge feature/x")[0], 0)

    def test_branch_behind_main_is_blocked_even_if_checked(self):
        self.mark("feature/x")
        sh(self.repo, "git", "checkout", "-q", "main")
        self.commit("c.txt", "3")
        code, err = self.guard("git merge feature/x")
        self.assertEqual(code, 2)
        self.assertIn("does not contain the current main", err)

    def test_merge_abort_is_allowed(self):
        sh(self.repo, "git", "checkout", "-q", "main")
        self.assertEqual(self.guard("git merge --abort")[0], 0)

    def test_push_of_unchecked_main_is_blocked(self):
        sh(self.repo, "git", "checkout", "-q", "main")
        self.commit("direct.txt", "on main")
        for cmd in ["git push origin main", "git push", "git push origin HEAD:main",
                    "git push --all origin"]:
            self.assertEqual(self.guard(cmd)[0], 2, cmd)

    def test_push_of_checked_merge_is_allowed(self):
        self.mark("feature/x")
        sh(self.repo, "git", "checkout", "-q", "main")
        sh(self.repo, "git", "merge", "-q", "--no-ff", "feature/x", "-m", "Merge")
        self.assertEqual(self.guard("git push origin main")[0], 0)

    def test_push_in_same_line_as_merge_is_blocked(self):
        self.mark("feature/x")
        sh(self.repo, "git", "checkout", "-q", "main")
        code, err = self.guard("git merge --no-ff feature/x -m M && git push origin main")
        self.assertEqual(code, 2)
        self.assertIn("separate command", err)

    def test_push_of_feature_branch_is_not_gated(self):
        self.assertEqual(self.guard("git push -u origin feature/x")[0], 0)
        self.assertEqual(self.guard("git push")[0], 0)

    def test_pull_of_other_branch_on_main_is_blocked(self):
        sh(self.repo, "git", "checkout", "-q", "main")
        self.assertEqual(self.guard("git pull origin feature/x")[0], 2)
        self.assertEqual(self.guard("git pull origin main")[0], 0)
        self.assertEqual(self.guard("git pull")[0], 0)


if __name__ == "__main__":
    unittest.main()
