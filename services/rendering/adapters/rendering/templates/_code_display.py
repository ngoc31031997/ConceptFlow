"""Shared code-display helper used by every template (Business Rule 3-4).

Not a template itself — implements the "if code_snippet is present, show
it fixed in a corner for the whole scene" placement rule once, so each
template (algorithm_visualization, concept_illustration, ...) doesn't
duplicate the logic.
"""

from __future__ import annotations

import logging

from manim import LEFT, Code

from domain.models import SceneRenderRequest

logger = logging.getLogger(__name__)

_KNOWN_PYGMENTS_LANGUAGES = None  # populated lazily to avoid import cost when unused


def _is_known_language(language: str | None) -> bool:
    global _KNOWN_PYGMENTS_LANGUAGES
    if not language:
        return False
    if _KNOWN_PYGMENTS_LANGUAGES is None:
        from pygments.lexers import get_all_lexers

        _KNOWN_PYGMENTS_LANGUAGES = {
            alias.lower() for _, aliases, _, _ in get_all_lexers() for alias in aliases
        }
    return language.lower() in _KNOWN_PYGMENTS_LANGUAGES


def build_code_mobject(request: SceneRenderRequest) -> Code | None:
    """Returns a Code mobject positioned in the left column, or None if
    the scene has no code_snippet.

    Falls back to plain text (no syntax highlight) when code_language is
    missing or not recognized by Pygments (Business Rule 4) — this never
    raises, it only logs a warning, so an unrecognized language never
    blocks rendering.
    """
    if not request.code_snippet:
        return None

    language = request.code_language
    if not _is_known_language(language):
        if language:
            logger.warning(
                "Unrecognized code_language=%r for project_id=%s scene_index=%s — "
                "falling back to plain text",
                language,
                request.project_id,
                request.scene_index,
            )
        language = "text"

    code = Code(code=request.code_snippet, language=language)
    code.to_edge(LEFT)
    return code
