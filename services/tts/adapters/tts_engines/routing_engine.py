"""RoutingTTSEngine — picks the engine a voice belongs to, falls back to Piper
when the cloud one cannot deliver, and meters what it sends (CR-005 FR13.3 /
FR13.5, ADR-0023).

This sits behind TTSEnginePort like any other engine, so the use case stays
unaware that there is more than one. Keeping the routing here rather than in
the application layer is what let ADR-0023 add a second engine without touching
business logic — the point of the port in the first place.
"""

from __future__ import annotations

import json
import logging
import os
from datetime import UTC, datetime

from adapters.tts_engines.voice_registry import (
    ENGINE_GOOGLE,
    engine_for,
    fallback_voice_for,
)
from domain.errors import TTSEngineError
from domain.ports import TTSEnginePort

logger = logging.getLogger(__name__)

USAGE_PATH = "/shared/tts_usage.json"

# Google's monthly free allowance on the WaveNet tier (ADR-0023). Used only to
# decide when to warn — nothing here blocks synthesis, because at ~8,000
# characters per ten-minute video this is roughly 500 videos a month and
# overage costs about three cents a video.
FREE_TIER_CHARACTERS = 4_000_000
WARN_AT_FRACTION = 0.8


class RoutingTTSEngine(TTSEnginePort):
    def __init__(
        self,
        piper: TTSEnginePort,
        google: TTSEnginePort | None,
        usage_path: str = USAGE_PATH,
    ) -> None:
        self._piper = piper
        # None when no credentials are configured — every voice then resolves
        # to Piper, which is why the service still runs fully offline.
        self._google = google
        self._usage_path = usage_path

    def synthesize(self, text: str, voice_id: str, output_path: str) -> float:
        if engine_for(voice_id) != ENGINE_GOOGLE or self._google is None:
            if engine_for(voice_id) == ENGINE_GOOGLE:
                fallback = fallback_voice_for(voice_id)
                logger.warning(
                    "Google TTS is not configured; voice %s falls back to %s. "
                    "Set GOOGLE_APPLICATION_CREDENTIALS for the higher-quality voice.",
                    voice_id,
                    fallback,
                )
                voice_id = fallback
            return self._piper.synthesize(text, voice_id, output_path)

        try:
            duration = self._google.synthesize(text, voice_id, output_path)
        except TTSEngineError as exc:
            # A network blip or an expired credential should cost quality, not
            # the whole render — but never silently (FR13.3).
            fallback = fallback_voice_for(voice_id)
            logger.warning(
                "Google TTS failed for %s (%s); falling back to the offline voice %s. "
                "This video's narration will be noticeably lower quality.",
                voice_id,
                exc,
                fallback,
            )
            return self._piper.synthesize(text, fallback, output_path)

        self._record_usage(len(text))
        return duration

    def _record_usage(self, characters: int) -> None:
        """Tracks characters sent to the metered engine, per month.

        Best-effort by design: a usage file that cannot be written is a
        reporting gap, not a reason to fail a render the Creator is waiting on.
        """
        month = datetime.now(UTC).strftime("%Y-%m")
        try:
            usage = {}
            if os.path.isfile(self._usage_path):
                with open(self._usage_path, encoding="utf-8") as f:
                    usage = json.load(f)
            if not isinstance(usage, dict):
                usage = {}

            total = int(usage.get(month, 0)) + characters
            usage[month] = total

            os.makedirs(os.path.dirname(self._usage_path), exist_ok=True)
            with open(self._usage_path, "w", encoding="utf-8") as f:
                json.dump(usage, f)

            if total > FREE_TIER_CHARACTERS * WARN_AT_FRACTION:
                logger.warning(
                    "Google TTS usage for %s is %d characters, past %d%% of the "
                    "%d free-tier allowance. Further synthesis may be billed.",
                    month,
                    total,
                    int(WARN_AT_FRACTION * 100),
                    FREE_TIER_CHARACTERS,
                )
        except (OSError, ValueError):
            logger.exception("Could not record TTS usage at %s", self._usage_path)
