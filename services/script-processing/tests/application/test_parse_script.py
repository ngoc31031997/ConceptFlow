"""Unit tests for ParseScriptUseCase — thin wrapper, verifies delegation."""

from __future__ import annotations

import pytest

from application.parse_script import ParseScriptUseCase
from domain.errors import ScriptSyntaxError
from domain.models import ParsedScript, Scene
from domain.ports import ScriptParserPort


class FakeScriptParser(ScriptParserPort):
    def __init__(self, result: ParsedScript | None = None, error: ScriptSyntaxError | None = None) -> None:
        self._result = result
        self._error = error
        self.received_raw_script: str | None = None

    def parse(self, raw_script: str) -> ParsedScript:
        self.received_raw_script = raw_script
        if self._error is not None:
            raise self._error
        return self._result


def test_parse_delegates_to_parser_and_returns_result():
    expected = ParsedScript(scenes=[Scene(0, "hello", None, None, None)])
    parser = FakeScriptParser(result=expected)
    use_case = ParseScriptUseCase(parser)

    result = use_case.parse("## Scene 1\nhello")

    assert result is expected
    assert parser.received_raw_script == "## Scene 1\nhello"


def test_parse_propagates_syntax_error():
    parser = FakeScriptParser(error=ScriptSyntaxError(1, "no scenes found"))
    use_case = ParseScriptUseCase(parser)

    with pytest.raises(ScriptSyntaxError):
        use_case.parse("not a script")
