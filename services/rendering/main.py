"""Composition root for the Rendering Service.

AMQP consumer with a PostgreSQL-backed Outbox/Inbox (ADR-0013), plus the
internal HTTP compile check. No public REST endpoint — readiness is
signaled via a sentinel file (Infrastructure Design).
"""

from __future__ import annotations

import asyncio
import logging
import os
import signal
from pathlib import Path

import aio_pika
import uvicorn

from adapters.http.check_metrics import CheckMetrics
from adapters.http.check_server import create_check_app
from adapters.messaging.cancellation import CancelAwareOutbox, listen_for_cancels
from adapters.messaging.consumer import (
    RenderingCommandDispatcher,
    RenderScriptCommandHandler,
    ValidateScriptCommandHandler,
)
from adapters.messaging.producer import EVENTS_EXCHANGE, EVENTS_ROUTING_KEY
from adapters.messaging.progress import PROGRESS_EXCHANGE, ProgressPublisher
from adapters.messaging.purge import PurgeProjectArtifactsCommandHandler
from adapters.persistence.db import create_pool
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from adapters.persistence.relay import OutboxRelay
from adapters.rendering.illustration_previewer import IllustrationPreviewer
from adapters.rendering.layout_checker import LayoutChecker
from adapters.rendering.remotion_renderer import (
    CACHE_ROOT,
    DEFAULT_RENDER_TIMEOUT_SECONDS,
    RemotionScriptRenderer,
)
from adapters.rendering.typescript_checker import TypeScriptChecker
from adapters.storage.artifact_paths import purge_project_artifacts
from application.check_script import CheckScriptUseCase
from application.preview_illustration import PreviewIllustrationUseCase
from application.stills import StillsUseCase
from domain.kits import load_kits, load_skeletons
from application.render_script import RenderScriptUseCase
from application.validate_script import ValidateScriptUseCase
from domain import lottie_catalog

logging.basicConfig(level=logging.WARNING)
logger = logging.getLogger(__name__)

COMMANDS_QUEUE = "rendering.commands"
RABBITMQ_URL = os.environ["RABBITMQ_URL"]
READY_SENTINEL_PATH = "/tmp/ready"



LOTTIE_MANIFEST = Path(__file__).parent / "remotion_project" / "lottie" / "manifest.json"


def env_flag(name: str, default: bool) -> bool:
    raw = os.environ.get(name, "").strip().lower()
    if raw == "":
        return default
    if raw in ("1", "true", "yes", "on"):
        return True
    if raw in ("0", "false", "no", "off"):
        return False
    raise ValueError(f"{name} must be true or false, got {raw!r}")


def approved_lottie_ids() -> set[str]:
    """Ids a Remotion script may pass to <LottieClip>; read per call so a
    manifest fixed by hand is picked up without restarting the consumer."""
    return {a.id for a in lottie_catalog.approved(lottie_catalog.load_manifest(LOTTIE_MANIFEST))}

async def run() -> None:
    timeout_seconds = int(os.environ.get("RENDER_TIMEOUT_SECONDS", DEFAULT_RENDER_TIMEOUT_SECONDS))
    renderer = RemotionScriptRenderer(timeout_seconds=timeout_seconds)
    use_case = RenderScriptUseCase(renderer)

    pool = await create_pool()
    inbox = InboxRepository(pool)
    outbox = CancelAwareOutbox(OutboxRepository())

    connection = await aio_pika.connect_robust(RABBITMQ_URL)
    channel = await connection.channel()
    # One command at a time. RenderingCommandDispatcher's docstring states the
    # commands "cùng phải xếp hàng sau nhau" — but that only holds if the
    # broker stops handing us the next one before the current render is done.
    # Without this, aio-pika prefetches the whole queue and `queue.consume()`
    # schedules every delivery as its own task, so several renders would run
    # at once and fight over the container's CPU and memory.
    await channel.set_qos(prefetch_count=1)
    exchange = await channel.get_exchange(EVENTS_EXCHANGE)
    progress_exchange = await channel.get_exchange(PROGRESS_EXCHANGE)
    queue = await channel.get_queue(COMMANDS_QUEUE)

    def make_persistent_message(body: bytes) -> aio_pika.Message:
        return aio_pika.Message(body, delivery_mode=aio_pika.DeliveryMode.PERSISTENT)

    command_handler = RenderingCommandDispatcher(
        # Cổng kiểm tra chạy trước TTS: script sai bị chặn trước khi
        # tiêu quota giọng đọc.
        ValidateScriptCommandHandler(
            ValidateScriptUseCase(renderer, approved_lottie_ids), pool, inbox, outbox
        ),
        RenderScriptCommandHandler(
            use_case, pool, inbox, outbox, ProgressPublisher(progress_exchange)
        ),
        # Dọn video đã dựng và cache Remotion của project bị xoá.
        purge_project_artifacts=PurgeProjectArtifactsCommandHandler(
            lambda project_id: purge_project_artifacts(project_id, CACHE_ROOT), pool, inbox, outbox
        ),
    )
    relay = OutboxRelay(pool, exchange, make_persistent_message, EVENTS_ROUTING_KEY)
    relay.start()

    # How many render commands are in flight when a check starts.
    check_metrics = CheckMetrics()

    async def handle_counted(message):
        with check_metrics.command_running():
            await command_handler.handle(message)

    consumer_tag = await queue.consume(handle_counted)
    # Cancel requests arrive on their own fanout, not behind the running render.
    await listen_for_cancels(channel)

    # The compile check the llm-service calls before a generated
    # script is saved. Same process, same event loop; internal network only.
    typescript = TypeScriptChecker(
        Path(__file__).parent / "remotion_project",
        timeout_seconds=int(os.environ.get("TSC_TIMEOUT_SECONDS", "90")),
    )
    # Load React/Remotion's type declarations now, not on the first check.
    warm_typescript = asyncio.create_task(asyncio.to_thread(typescript.warm))
    # After tsc passes, every shot is drawn in headless Chromium and
    # its measured layout held to the layout rules. Off = each check says so.
    layout = None
    warm_layout = None
    if env_flag("LAYOUT_CHECK_ENABLED", True):
        layout = LayoutChecker(
            Path(__file__).parent / "remotion_project",
            timeout_seconds=int(os.environ.get("LAYOUT_CHECK_TIMEOUT_SECONDS", "60")),
        )
        # Bundle the probe page and start the browser now, not on the first check.
        warm_layout = asyncio.create_task(asyncio.to_thread(layout.warm))
    else:
        logger.warning("LAYOUT_CHECK_ENABLED=false — code checks will not look at the layout")
    check_use_case = CheckScriptUseCase(
        ValidateScriptUseCase(renderer, approved_lottie_ids), typescript, layout)
    # Xem trước hình của thư viện minh hoạ (tiến trình Node khởi động lúc cần).
    previewer = IllustrationPreviewer(
        Path(__file__).parent / "remotion_project",
        timeout_seconds=int(os.environ.get("ILLUSTRATION_PREVIEW_TIMEOUT_SECONDS", "180")),
    )
    remotion_dir = Path(__file__).parent / "remotion_project"
    kits = load_kits(remotion_dir)
    stills = StillsUseCase(previewer, kits, os.environ.get("STILLS_DIR", "/shared/stills"))
    check_server = uvicorn.Server(uvicorn.Config(
        create_check_app(
            check_use_case, concurrency=int(os.environ.get("CHECK_CONCURRENCY", "2")), metrics=check_metrics,
            illustrations=PreviewIllustrationUseCase(typescript, previewer, kits, load_skeletons(remotion_dir)),
            stills=stills, kits=kits,
        ),
        host="0.0.0.0", port=int(os.environ.get("CHECK_HTTP_PORT", "8000")), log_level="warning",
    ))
    check_server_task = asyncio.create_task(check_server.serve())

    with open(READY_SENTINEL_PATH, "w") as f:
        f.write("ready")
    logger.info("Rendering Service ready — consuming '%s'", COMMANDS_QUEUE)

    # SIGTERM's default disposition terminates the process immediately,
    # unlike SIGINT (KeyboardInterrupt) — without a handler, `docker stop`
    # kills us before the `finally` below ever runs, and the AMQP connection
    # / DB pool are torn down uncleanly instead of closed.
    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, stop_event.set)

    try:
        await stop_event.wait()
        logger.info("Shutdown signal received — draining Rendering Service")
    finally:
        check_server.should_exit = True
        await check_server_task
        await warm_typescript
        typescript.close()
        if layout is not None:
            await warm_layout
            layout.close()
        previewer.close()
        await queue.cancel(consumer_tag)
        await relay.stop()
        await connection.close()
        await pool.close()


if __name__ == "__main__":
    asyncio.run(run())
