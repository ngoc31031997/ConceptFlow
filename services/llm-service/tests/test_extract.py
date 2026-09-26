import re

import pytest

from app.pipeline import extract

TSX = re.compile(r"^function Shot(\d+)_(\d+)\s*\(")
PY = re.compile(r"^def shot_(\d+)_(\d+)\s*\(self")


def test_splits_functions_with_their_leading_comment():
    code = """// Shot 1.1 — MÁY: a | HÌNH: b
function Shot1_1({duration}: ShotProps) {
  return null;
}

// Shot 1.2 — x
function Shot1_2({duration}: ShotProps) {
  return null;
}
"""
    blocks = extract.split_shots(code, TSX)
    assert [b.shot_id for b in blocks] == ["1.1", "1.2"]
    assert blocks[0].code.startswith("// Shot 1.1") and "Shot1_2" not in blocks[0].code
    assert blocks[1].code.startswith("// Shot 1.2")


def test_python_methods_and_two_digit_ids():
    code = "def shot_10_2(self):\n    self.narrate('x')\n\ndef shot_1_1(self):\n    pass\n"
    assert [b.shot_id for b in extract.split_shots(code, PY)] == ["10.2", "1.1"]


def test_rejects_empty_leading_code_and_duplicates():
    with pytest.raises(extract.ExtractError, match="no shot function"):
        extract.split_shots("const x = 1;", TSX)
    with pytest.raises(extract.ExtractError, match="unexpected code before"):
        extract.split_shots("const x = 1;\nfunction Shot1_1() {}\n", TSX)
    dup = extract.split_shots("function Shot1_1() {}\nfunction Shot1_1() {}\n", TSX)
    with pytest.raises(extract.ExtractError, match="defined twice"):
        extract.shot_map(dup)


def test_strip_fence_takes_one_block_and_refuses_several():
    assert extract.strip_fence("```tsx\nA\n```") == "A"
    assert extract.strip_fence("plain") == "plain"
    with pytest.raises(extract.ExtractError):
        extract.strip_fence("```\nA\n```\ntext\n```\nB\n```")
