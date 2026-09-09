"""RoutingTTSEngine — picks the engine a voice belongs to, falls back to Edge
when a metered engine cannot deliver, and meters what it sends (CR-005 FR13.3 /
FR13.5, ADR-0023, ADR-0024, CR-011).

This sits behind TTSEnginePort like any other engine, so the use case stays
unaware that there is more than one. Keeping the routing here rather than in
the application layer is what let ADR-0023 add a second engine, ADR-0024 swap
the base one, and CR-011 add a third, without touching business logic — the
point of the port in the first place.

Edge is the only engine that is always present: it needs no account, so it is
both the default and the thing every other engine degrades to.
"""

from __future__ import annotations

import json
import logging
import os
from datetime import UTC, datetime

from adapters.tts_engines.voice_registry import (
    ENGINE_AZURE,
    ENGINE_EDGE,
    ENGINE_GOOGLE,
    engine_for,
    fallback_voice_for,
)
from domain.errors import TTSEngineError
from domain.ports import TTSEnginePort

logger = logging.getLogger(__name__)

USAGE_PATH = "/shared/tts_usage.json"

# What each metered engine allows per month before characters start costing
# money. Azure's is a real free tier (F0: 500k neural characters, CR-011).
# Google's was withdrawn (CR-010), so its number is now only a "you are
# spending real money" tripwire. Nothing here blocks synthesis.
FREE_TIER_CHARACTERS = {
    ENGINE_AZURE: 500_000,
    ENGINE_GOOGLE: 4_000_000,
}
WARN_AT_FRACTION = 0.8

CREDENTIAL_HINT = {
    ENGINE_AZURE: "AZURE_SPEECH_KEY and AZURE_SPEECH_REGION",
    ENGINE_GOOGLE: "GOOGLE_APPLICATION_CREDENTIALS",
}


class RoutingTTSEngine(TTSEnginePort):
    def __init__(
        self,
        edge: TTSEnginePort,
        metered: dict[str, TTSEnginePort] | None = None,
        usage_path: str = USAGE_PATH,
    ) -> None:
        self._edge = edge
        # Only holds the engines whose credentials are configured; a voice
        # belonging to any other engine resolves to Edge instead.
        self._metered = metered or {}
        self._usage_path = usage_path

    @property
    def available_engines(self) -> set[str]:
        """The engines that can actually synthesize here. Edge is always in the
        set — it needs no account, which is why everything degrades to it. The
        catalogue is filtered by this so the GUI never offers a voice that would
        quietly come back in a different engine's voice."""
        return {ENGINE_EDGE, *self._metered}

    def synthesize(self, text: str, voice_id: str, output_path: str) -> float:
        engine_name = engine_for(voice_id)
        engine = self._metered.get(engine_name)

        if engine is None:
            if engine_name != ENGINE_EDGE:
                fallback = fallback_voice_for(voice_id)
                logger.warning(
                    "%s TTS is not configured; voice %s falls back to %s. "
                    "Set %s to use the voice that was chosen.",
                    engine_name,
                    voice_id,
                    fallback,
                    CREDENTIAL_HINT.get(engine_name, "its credentials"),
                )
                voice_id = fallback
            return self._edge.synthesize(text, voice_id, output_path)

        try:
            duration = engine.synthesize(text, voice_id, output_path)
        except TTSEngineError as exc:
            # A network blip, an expired credential or a spent quota should cost
            # the chosen voice, not the whole render — but never silently
            # (FR13.3).
            fallback = fallback_voice_for(voice_id)
            logger.warning(
                "%s TTS failed for %s (%s); falling back to %s.",
                engine_name,
                voice_id,
                exc,
                fallback,
            )
            return self._edge.synthesize(text, fallback, output_path)

        self._record_usage(engine_name, len(text))
        return duration

    def _record_usage(self, engine_name: str, characters: int) -> None:
        """Tracks characters sent to each metered engine, per month.

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

            by_engine = usage.get(month)
            if not isinstance(by_engine, dict):
                by_engine = {}
            total = int(by_engine.get(engine_name, 0)) + characters
            by_engine[engine_name] = total
            usage[month] = by_engine

            os.makedirs(os.path.dirname(self._usage_path), exist_ok=True)
            with open(self._usage_path, "w", encoding="utf-8") as f:
                json.dump(usage, f)

            allowance = FREE_TIER_CHARACTERS.get(engine_name)
            if allowance and total > allowance * WARN_AT_FRACTION:
                logger.warning(
                    "%s TTS usage for %s is %d characters, past %d%% of the "
                    "%d free allowance. Further synthesis may be billed.",
                    engine_name,
                    month,
                    total,
                    int(WARN_AT_FRACTION * 100),
                    allowance,
                )
        except (OSError, ValueError):
            logger.exception("Could not record TTS usage at %s", self._usage_path)
