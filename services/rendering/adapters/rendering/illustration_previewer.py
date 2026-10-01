"""Ảnh xem trước cho một hình của thư viện minh hoạ.

Giống TypeScriptChecker: một tiến trình Node sống lâu
(`remotion_project/illustration_preview.mjs`) đóng gói host và mở trình duyệt
MỘT lần, rồi mỗi yêu cầu chỉ tốn một lần dựng. Một yêu cầu một lúc.
"""

from __future__ import annotations

import json
import os
import queue
import shutil
import subprocess
import threading
import time
import uuid
from collections import deque
from dataclasses import dataclass
from pathlib import Path

from adapters.rendering.typescript_checker import _pump


class IllustrationPreviewError(RuntimeError):
    """Bộ dựng xem trước không chạy được (khác với hình vẽ sai)."""


@dataclass(frozen=True)
class PreviewResult:
    ok: bool
    png: str = ""  # base64
    gif: str = ""  # base64
    error: str = ""


class IllustrationPreviewer:
    def __init__(self, project_dir: str | Path, timeout_seconds: int = 180, node: str = "node") -> None:
        self._dir = Path(project_dir)
        self._timeout = timeout_seconds
        self._node = node
        self._lock = threading.Lock()
        self._proc: subprocess.Popen[str] | None = None
        self._replies: queue.Queue[str | None] = queue.Queue()
        self._stderr: deque[str] = deque(maxlen=40)

    def _start(self) -> None:
        script = self._dir / "illustration_preview.mjs"
        node = shutil.which(self._node)
        if node is None or not script.exists():
            raise IllustrationPreviewError(f"không chạy được {script} (node={node})")
        self._replies = queue.Queue()
        self._stderr.clear()
        self._proc = subprocess.Popen(
            [node, str(script)], cwd=self._dir,
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            text=True, encoding="utf-8", bufsize=1, env={**os.environ, "NO_COLOR": "1"},
        )
        threading.Thread(target=_pump, args=(self._proc.stdout, self._replies.put), daemon=True).start()
        threading.Thread(
            target=_pump,
            args=(self._proc.stderr, lambda s: s is not None and self._stderr.append(s)),
            daemon=True,
        ).start()
        self._await(lambda reply: reply.get("ready") is True)

    def _stop(self) -> None:
        proc, self._proc = self._proc, None
        if proc is not None:
            try:
                proc.kill()
                proc.wait(timeout=5)
            except (OSError, subprocess.TimeoutExpired):
                pass

    def close(self) -> None:
        with self._lock:
            self._stop()

    def _await(self, match) -> dict:
        deadline = time.monotonic() + self._timeout
        while True:
            try:
                line = self._replies.get(timeout=max(0.0, deadline - time.monotonic()))
            except queue.Empty:
                self._stop()
                raise IllustrationPreviewError(f"xem trước quá {self._timeout}s") from None
            if line is None:
                tail = "".join(self._stderr).strip()[-1500:]
                self._stop()
                raise IllustrationPreviewError(f"bộ dựng xem trước đã thoát: {tail or 'không có output'}")
            try:
                reply = json.loads(line)
            except ValueError:
                continue  # Remotion đôi khi in log ra stdout
            if match(reply):
                return reply

    def preview(
        self, name: str, code: str = "", props: dict | None = None, gif: bool = True, kind: str = "figure",
    ) -> PreviewResult:
        with self._lock:
            if self._proc is None or self._proc.poll() is not None:
                self._start()
            assert self._proc is not None and self._proc.stdin is not None
            rid = uuid.uuid4().hex
            msg = {"id": rid, "name": name, "props": props or {}, "gif": gif, "kind": kind}
            if code:
                msg["code"] = code
            try:
                self._proc.stdin.write(json.dumps(msg) + "\n")
                self._proc.stdin.flush()
            except OSError as exc:
                self._stop()
                raise IllustrationPreviewError(f"bộ dựng xem trước không nhận input: {exc}") from exc
            reply = self._await(lambda r: r.get("id") == rid)
        if reply.get("ok"):
            return PreviewResult(True, png=reply.get("png", ""), gif=reply.get("gif", ""))
        return PreviewResult(False, error=str(reply.get("error", "lỗi không rõ")))
