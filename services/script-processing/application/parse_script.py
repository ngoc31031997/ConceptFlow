"""ParseScriptUseCase — business-logic-model.md."""

from __future__ import annotations

from domain.models import ParsedScript
from domain.ports import ScriptParserPort


class ParseScriptUseCase:
    def __init__(self, parser: ScriptParserPort) -> None:
        self._parser = parser

    def parse(self, raw_script: str) -> ParsedScript:
        return self._parser.parse(raw_script)
