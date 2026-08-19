"""Abstract port for the Script Processing Service (module-structure.md, ADR-0002).

The application layer depends only on this abstraction, never on a
concrete grammar — this is what lets a future syntax (e.g. YAML) be
added without touching business logic.
"""

from __future__ import annotations

from abc import ABC, abstractmethod

from domain.models import ParsedScript


class ScriptParserPort(ABC):
    @abstractmethod
    def parse(self, raw_script: str) -> ParsedScript:
        """Parses raw_script into a ParsedScript.

        Raises:
            domain.errors.ScriptSyntaxError: on any grammar violation.
        """
