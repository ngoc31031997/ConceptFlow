"""Đo bố cục một script Remotion trong Chromium headless.

Giống TypeScriptChecker: một tiến trình Node sống lâu
(`remotion_project/layout_check.mjs`) đóng gói trang đo và mở trình duyệt MỘT
lần; mỗi lần kiểm tra chỉ tốn một trang mới, biên dịch script và đo. Một yêu
cầu một lúc (một trình duyệt, bộ nhớ phẳng).

Nó chỉ trả số đo (JSON của layout probe). Luật nằm ở `domain/layout_rules.py`.
Trình duyệt không mở được, tiến trình chết, quá giờ hay trả lời không đọc
được → `LayoutCheckError`: người gọi báo "không kiểm tra được bố cục", KHÔNG
bao giờ coi là đạt.
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
from pathlib import Path

from adapters.rendering.remotion_renderer import _sanitize_script
from adapters.rendering.typescript_checker import _pump


class LayoutCheckError(RuntimeError):
    """Không đo được bố cục (khác với bố cục sai)."""


class LayoutScriptError(LayoutCheckError):
    """Chính script không nạp được vào trang đo (lỗi cú pháp, ném lỗi khi nạp,
    không gọi registerRoot) — tsc đã qua nên hiếm; vẫn không phải "đạt"."""


class LayoutChecker:
    def __init__(self, project_dir: str | Path, timeout_seconds: int = 60, node: str = "node") -> None:
        self._dir = Path(project_dir)
        self._timeout = timeout_seconds
        self._node = node
        self._lock = threading.Lock()
        self._proc: subprocess.Popen[str] | None = None
        self._replies: queue.Queue[str | None] = queue.Queue()
        self._stderr: deque[str] = deque(maxlen=40)
        self.browser_version = ""

    # -- tiến trình ------------------------------------------------------------

    def _start(self, deadline: float) -> None:
        script = self._dir / "layout_check.mjs"
        if not script.exists():
            raise LayoutCheckError(f"thiếu {script}")
        for dep in ("playwright-core", "@remotion/player", "esbuild", "typescript"):
            if not (self._dir / "node_modules" / dep).exists():
                raise LayoutCheckError(
                    f"thiếu gói {dep} trong {self._dir / 'node_modules'} — remotion_project chưa cài?")
        node = shutil.which(self._node)
        if node is None:
            raise LayoutCheckError(f"không tìm thấy {self._node!r} trên PATH")
        try:
            proc = subprocess.Popen(
                [node, str(script)], cwd=self._dir,
                stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                text=True, encoding="utf-8", bufsize=1, env={**os.environ, "NO_COLOR": "1"},
            )
        except OSError as exc:
            raise LayoutCheckError(f"không khởi động được bộ đo bố cục: {exc}") from exc
        self._replies = queue.Queue()
        self._stderr.clear()
        threading.Thread(target=_pump, args=(proc.stdout, self._replies.put), daemon=True).start()
        threading.Thread(
            target=_pump, args=(proc.stderr, lambda s: s is not None and self._stderr.append(s)), daemon=True,
        ).start()
        self._proc = proc
        hello = self._await(lambda r: "ready" in r or "fatal" in r, deadline, "khởi động")
        if hello.get("fatal"):
            self._stop()
            raise LayoutCheckError(f"bộ đo bố cục không mở được trình duyệt: {hello['fatal']}")
        self.browser_version = str((hello.get("browser") or {}).get("version", ""))

    def _stop(self) -> None:
        proc, self._proc = self._proc, None
        if proc is None:
            return
        try:
            proc.kill()
            proc.wait(timeout=5)
        except (OSError, subprocess.TimeoutExpired):
            pass

    def close(self) -> None:
        with self._lock:
            self._stop()

    def warm(self) -> None:
        """Mở trình duyệt ngay, để lần kiểm tra đầu không phải chờ. Lỗi (nếu có)
        sẽ hiện ra ở lần kiểm tra đó."""
        with self._lock:
            if self._proc is None or self._proc.poll() is not None:
                try:
                    self._start(time.monotonic() + self._timeout)
                except LayoutCheckError:
                    self._stop()

    def _await(self, match, deadline: float, what: str) -> dict:
        while True:
            try:
                line = self._replies.get(timeout=max(0.0, deadline - time.monotonic()))
            except queue.Empty:
                self._stop()
                raise LayoutCheckError(f"đo bố cục quá {self._timeout}s ({what})") from None
            if line is None:
                tail = "".join(self._stderr).strip()[-1500:]
                self._stop()
                raise LayoutCheckError(f"bộ đo bố cục đã thoát ({what}): {tail or 'không có output'}")
            try:
                reply = json.loads(line)
            except ValueError as exc:
                self._stop()
                raise LayoutCheckError(f"bộ đo bố cục trả một dòng không đọc được: {line[:300]!r}") from exc
            if isinstance(reply, dict) and match(reply):
                return reply

    # -- đo --------------------------------------------------------------------

    def measure(self, script: str, font: str = "") -> dict:
        """Số đo của mọi shot (JSON của layout probe, version 1)."""
        with self._lock:
            deadline = time.monotonic() + self._timeout
            if self._proc is None or self._proc.poll() is not None:
                self._start(deadline)
            assert self._proc is not None and self._proc.stdin is not None
            rid = uuid.uuid4().hex
            msg: dict = {"id": rid, "code": _sanitize_script(script)}
            if font:
                msg["font"] = font
            try:
                self._proc.stdin.write(json.dumps(msg) + "\n")
                self._proc.stdin.flush()
            except OSError as exc:
                self._stop()
                raise LayoutCheckError(f"bộ đo bố cục không nhận input: {exc}") from exc
            reply = self._await(lambda r: r.get("id") == rid, deadline, "đo")
        return self._read(reply)

    @staticmethod
    def _read(reply: dict) -> dict:
        if reply.get("error"):
            stage = reply.get("stage", "")
            text = str(reply["error"])[:1500]
            if stage in ("compile", "load"):
                raise LayoutScriptError(f"script không nạp được vào trang đo: {text}")
            raise LayoutCheckError(f"đo bố cục thất bại ({stage or 'không rõ'}): {text}")
        result = reply.get("result")
        if (not isinstance(result, dict) or result.get("version") != 1
                or not isinstance(result.get("shots"), list)):
            raise LayoutCheckError(
                "bộ đo bố cục trả kết quả không đúng dạng (cần version 1 với danh sách shots)")
        return result
