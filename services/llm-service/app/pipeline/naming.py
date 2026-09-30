"""Identifier helpers shared by the mergers."""

from __future__ import annotations

import re
import unicodedata


def ascii_words(text: str) -> list[str]:
    text = text.replace("đ", "d").replace("Đ", "D")
    text = unicodedata.normalize("NFKD", text)
    text = "".join(c for c in text if not unicodedata.combining(c))
    return re.findall(r"[A-Za-z0-9]+", text)


def camel(text: str, fallback: str = "color") -> str:
    """camelCase that keeps the capitals already inside a word, so a role the
    storyboard wrote as `conNguoi` stays `conNguoi` (CR-056: lowering it made the
    model write a key PALETTE did not have). A word in all capitals is lowered."""
    words = [w.lower() if len(w) > 1 and w.isupper() else w for w in ascii_words(text)]
    if not words:
        return fallback
    out = words[0][0].lower() + words[0][1:] + "".join(w[0].upper() + w[1:] for w in words[1:])
    return out if not out[0].isdigit() else "c" + out


def pascal(text: str, fallback: str = "Video") -> str:
    words = ascii_words(text)
    if not words:
        return fallback
    out = "".join(w.capitalize() for w in words)
    return out if not out[0].isdigit() else "V" + out


def unique(names: list[str]) -> list[str]:
    seen: dict[str, int] = {}
    out = []
    for n in names:
        seen[n] = seen.get(n, 0) + 1
        out.append(n if seen[n] == 1 else f"{n}{seen[n]}")
    return out
