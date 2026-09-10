"""Runtime lời thoại hai lượt (CR-018).

Bộ test này chứng minh đúng thứ mà `# NARRATION` + `self.wait(AUTO)` không làm
được: lời thoại nằm trong vòng lặp, trong nhánh điều kiện và trong hàm helper.
"""

import json

import pytest

pytest.importorskip("manim", reason="cần Scene thật để chạy construct()")

from conceptflow import narration  # noqa: E402
from conceptflow.scene import ConceptFlowScene  # noqa: E402


@pytest.fixture
def dry(tmp_path, monkeypatch):
    """Chạy ở chế độ dry và trả về các bản ghi mà script ghi ra."""
    marks = tmp_path / "cf_marks.jsonl"
    monkeypatch.setenv(narration.ENV_MARKS_PATH, str(marks))
    monkeypatch.setenv(narration.ENV_MODE, narration.MODE_DRY)
    narration.reset()

    def run(scene_cls):
        scene_cls().construct()
        if not marks.exists():
            return []
        return [json.loads(line) for line in marks.read_text(encoding="utf-8").splitlines() if line]

    return run


def _texts(records):
    return [r["text"] for r in records if r["kind"] == "narration"]


def test_loi_thoai_trong_vong_lap(dry):
    """Điều `self.wait(AUTO)` không bao giờ làm được: số lần chạy phụ thuộc
    vòng lặp, nên không thể đếm bằng cách đọc text của script."""

    class S(ConceptFlowScene):
        def construct(self):
            for i in range(3):
                self.narrate(f"Ví dụ số {i + 1}")

    assert _texts(dry(S)) == ["Ví dụ số 1", "Ví dụ số 2", "Ví dụ số 3"]


def test_loi_thoai_trong_nhanh_dieu_kien(dry):
    class S(ConceptFlowScene):
        def construct(self):
            if True:
                self.narrate("nhánh đúng")
            else:
                self.narrate("nhánh sai")

    assert _texts(dry(S)) == ["nhánh đúng"]


def test_loi_thoai_trong_ham_helper(dry):
    """Đây là điều kiện để hook/CTA trở thành component thật (CR-019)."""

    class S(ConceptFlowScene):
        def hook(self):
            self.narrate("câu mở đầu")

        def construct(self):
            self.hook()
            self.narrate("nội dung chính")

    assert _texts(dry(S)) == ["câu mở đầu", "nội dung chính"]


def test_beat_va_chapter_gan_vao_loi_thoai_ke_tiep(dry):
    """Timestamp của beat/chapter là mốc thật lượt render đo được cho lời thoại
    mở đầu nó, không phải ước lượng (giữ nguyên cách CR-006 FR15 làm)."""

    class S(ConceptFlowScene):
        def construct(self):
            self.beat("hook")
            self.narrate("một")
            self.chapter("Phần hai")
            self.narrate("hai")

    records = dry(S)
    beat = next(r for r in records if r["kind"] == "beat")
    chapter = next(r for r in records if r["kind"] == "chapter")
    assert beat["index"] == 0
    assert chapter["index"] == 1


def test_loi_thoai_rong_bi_tu_choi(dry):
    class S(ConceptFlowScene):
        def construct(self):
            self.narrate("   ")

    with pytest.raises(narration.NarrationError):
        dry(S)


def test_che_do_render_cho_dung_thoi_luong_audio(tmp_path, monkeypatch):
    marks = tmp_path / "cf_marks.jsonl"
    durations = tmp_path / "cf_durations.json"
    durations.write_text(json.dumps([2.0, 3.5]))
    monkeypatch.setenv(narration.ENV_MARKS_PATH, str(marks))
    monkeypatch.setenv(narration.ENV_DURATIONS_PATH, str(durations))
    monkeypatch.setenv(narration.ENV_MODE, narration.MODE_RENDER)
    narration.reset()

    waited: list[float] = []

    class S(ConceptFlowScene):
        def wait(self, duration=None, **kwargs):  # noqa: D102
            waited.append(duration)

        def construct(self):
            self.narrate("một")
            self.narrate("hai")

    S().construct()

    assert waited == [2.0, 3.5]
    kinds = [json.loads(line)["kind"] for line in marks.read_text().splitlines() if line]
    assert kinds == ["mark", "mark"]


def test_render_thieu_thoi_luong_bao_loi_ro_rang(tmp_path, monkeypatch):
    """Lượt dry và lượt render lệch nhau chỉ xảy ra khi script không tất định."""
    marks = tmp_path / "cf_marks.jsonl"
    durations = tmp_path / "cf_durations.json"
    durations.write_text(json.dumps([1.0]))
    monkeypatch.setenv(narration.ENV_MARKS_PATH, str(marks))
    monkeypatch.setenv(narration.ENV_DURATIONS_PATH, str(durations))
    monkeypatch.setenv(narration.ENV_MODE, narration.MODE_RENDER)
    narration.reset()

    class S(ConceptFlowScene):
        def wait(self, duration=None, **kwargs):
            pass

        def construct(self):
            self.narrate("một")
            self.narrate("hai")

    with pytest.raises(narration.NarrationError, match="tất định"):
        S().construct()


def test_hook_recap_cta_tu_mang_beat_va_loi_thoai(dry):
    """CR-019 FR53: ba beat này chỉ trở thành method được sau CR-018.

    Trước đó lời thoại là comment phải đếm khớp theo thứ tự dòng, nên không thể
    nằm trong một hàm — đúng lý do CR-006 §Quyết định #2 phải lùi FR17 xuống
    thành snippet Creator tự chép.
    """

    class S(ConceptFlowScene):
        def wait(self, duration=None, **kwargs):
            pass

        def construct(self):
            self.hook("Vì sao vòng lặp này chạy mãi không dừng?")
            self.recap(["for gồm bốn phần", "Quên bước nhảy là lặp vô hạn"])
            self.call_to_action("Đăng ký để xem phần sau")

    records = dry(S)
    beats = [r["id"] for r in records if r["kind"] == "beat"]
    assert beats == ["hook", "recap", "cta"]

    narrations = [r["text"] for r in records if r["kind"] == "narration"]
    assert len(narrations) == 3
    assert narrations[0].startswith("Vì sao")


def test_ghi_lai_khung_hinh_tai_moi_loi_thoai(dry):
    """CR-024 FR68.5: duyệt dàn ý mà chỉ đọc lời thoại là duyệt nửa ít quan
    trọng hơn, với một kênh đặt trọng tâm vào ví dụ trực quan."""
    from manim import Square, Text

    class S(ConceptFlowScene):
        def construct(self):
            self.add(Text("a"), Text("b"), Square())
            self.narrate("có hình")

    record = next(r for r in dry(S) if r["kind"] == "narration")
    assert "Text×2" in record["visual"]
    assert "Square" in record["visual"]


def test_khung_trong_duoc_noi_ro(dry):
    """Một beat không có gì trên màn hình là thứ Creator cần thấy ngay."""

    class S(ConceptFlowScene):
        def construct(self):
            self.narrate("chỉ có tiếng, không có hình")

    record = next(r for r in dry(S) if r["kind"] == "narration")
    assert record["visual"] == "khung trống"
