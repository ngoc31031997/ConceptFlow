"""Reading the JSON object a model was asked to write.

A model often wraps the object in a ```json fence or adds a note after it, and
a note can hold a stray `}`. The object is read from the first `{` and stops where
it closes, so whatever follows is ignored.
"""

from __future__ import annotations

import json

_DECODER = json.JSONDecoder()


def first_json_object(text: str) -> tuple[dict | None, str]:
    """(object, "") for the first JSON object in `text`, else (None, why it could not be read)."""
    start = text.find("{")
    if start < 0:
        return None, "không có khối JSON"
    try:
        value, _ = _DECODER.raw_decode(text, start)
    except json.JSONDecodeError as exc:
        return None, str(exc)
    return value, ""


def repair_prompt(reply: str, problem: str) -> str:
    return (
        "Câu trả lời dưới đây đáng lẽ là một khối JSON nhưng không đọc được "
        f"({problem}). Viết lại đúng nội dung đó thành một khối JSON hợp lệ, "
        "giữ nguyên các khoá và giá trị, không thêm chữ nào ngoài khối JSON.\n\n"
        f"Câu trả lời:\n{reply}"
    )
