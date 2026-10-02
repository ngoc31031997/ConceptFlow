"""Runtime lời thoại hai lượt.

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
    """Đây là điều kiện để hook/CTA trở thành component thật."""

    class S(ConceptFlowScene):
        def hook(self):
            self.narrate("câu mở đầu")

        def construct(self):
            self.hook()
            self.narrate("nội dung chính")

    assert _texts(dry(S)) == ["câu mở đầu", "nội dung chính"]


def test_beat_va_chapter_gan_vao_loi_thoai_ke_tiep(dry):
    """Timestamp của beat/chapter là mốc thật lượt render đo được cho lời thoại
    mở đầu nó, không phải ước lượng."""

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
    # Mỗi mốc kèm đúng một bản ghi bố cục cho QC.
    assert kinds == ["mark", "layout", "mark", "layout"]


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
    """Hook/recap/CTA là method: lời thoại nằm được trong một hàm, và mỗi beat
    mang đúng beat id cùng lời thoại của nó.
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
    """Duyệt dàn ý mà chỉ đọc lời thoại là duyệt nửa ít quan
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


# --- thu dữ liệu bố cục ở lượt render ----------------------------------------


def _render_records(tmp_path, monkeypatch, scene_cls, count=1):
    marks = tmp_path / "cf_marks.jsonl"
    durations = tmp_path / "cf_durations.json"
    durations.write_text(json.dumps([1.0] * count))
    monkeypatch.setenv(narration.ENV_MARKS_PATH, str(marks))
    monkeypatch.setenv(narration.ENV_DURATIONS_PATH, str(durations))
    monkeypatch.setenv(narration.ENV_MODE, narration.MODE_RENDER)
    narration.reset()
    scene_cls().construct()
    return [json.loads(line) for line in marks.read_text(encoding="utf-8").splitlines() if line]


def test_ban_ghi_layout_di_kem_moc_render(tmp_path, monkeypatch):
    """Bản ghi bố cục phải khớp index và mốc thời gian của bản ghi `mark`:
    QC chấm hình tại đúng giây Video Assembly đặt lời thoại."""
    from manim import YELLOW, Text

    class S(ConceptFlowScene):
        def wait(self, duration=None, **kwargs):
            pass

        def construct(self):
            self.add(Text("xin chào", color=YELLOW))
            self.narrate("một")

    records = _render_records(tmp_path, monkeypatch, S)
    mark = next(r for r in records if r["kind"] == "mark")
    layout = next(r for r in records if r["kind"] == "layout")

    assert layout["index"] == mark["index"] == 0
    assert layout["t"] == mark["t"]
    assert len(layout["mobjects"]) == 1
    entry = layout["mobjects"][0]
    assert entry["cls"] == "Text"
    assert len(entry["bbox"]) == 4
    assert entry["bbox"][0] < entry["bbox"][1]  # left < right
    assert entry["bbox"][3] < entry["bbox"][2]  # bottom < top
    # Màu thật của chữ, không phải #000000 mà `Text.get_color()` luôn trả về:
    # luật tương phản của QC sống hay chết ở con số này.
    assert entry["color"] == "#FFFF00"
    assert entry["font_size"] > 0


def test_luot_dry_khong_thu_bo_cuc(dry):
    """Lượt dry là cổng chặn trước TTS — chỗ Creator đang ngồi chờ, không phải
    chỗ để thêm việc cho QC."""
    from manim import Text

    class S(ConceptFlowScene):
        def construct(self):
            self.add(Text("a"))
            self.narrate("một")

    assert all(r["kind"] != "layout" for r in dry(S))


def test_mobject_khong_phai_chu_co_font_size_none():
    from manim import Square

    class _Scene:
        mobjects = [Square()]

    (entry,) = narration._describe_layout(_Scene())
    assert entry["cls"] == "Square"
    assert entry["font_size"] is None
    assert entry["color"].startswith("#")


def test_khung_trong_cho_danh_sach_rong():
    class _Scene:
        mobjects = []

    assert narration._describe_layout(_Scene()) == []


def test_loi_thu_bo_cuc_khong_lam_hong_luot_render():
    """Dữ liệu QC không bao giờ được đắt hơn cái video nó đang chấm."""

    class _Broken:
        def get_left(self):
            raise RuntimeError("mobject lạ")

    class _Scene:
        mobjects = [_Broken()]

    assert narration._describe_layout(_Scene()) == []

    class _NoMobjects:
        @property
        def mobjects(self):
            raise RuntimeError("scene lạ")

    assert narration._describe_layout(_NoMobjects()) == []


# --- hình chuyển động trong lúc đọc thoại ------------------------------------


@pytest.fixture
def render(tmp_path, monkeypatch):
    """Lượt render giả: thời lượng audio cố định, ghi lại các lần play/wait."""
    marks = tmp_path / "cf_marks.jsonl"
    durations = tmp_path / "d.json"
    durations.write_text(json.dumps([2.0, 3.0]))
    monkeypatch.setenv(narration.ENV_MARKS_PATH, str(marks))
    monkeypatch.setenv(narration.ENV_MODE, narration.MODE_RENDER)
    monkeypatch.setenv(narration.ENV_DURATIONS_PATH, str(durations))
    narration.reset()
    return marks


class _FakeScene:
    class renderer:
        time = 0.0

    def __init__(self):
        self.calls = []

    def play(self, *animations, run_time=None):
        self.calls.append(("play", animations, run_time))

    def wait(self, seconds):
        self.calls.append(("wait", seconds))

    class camera:
        class frame:
            class animate:
                @staticmethod
                def scale(factor):
                    return ("scale", factor)


def test_animation_chay_dung_bang_thoi_luong_cau(render, monkeypatch):
    monkeypatch.setattr(narration, "_describe_layout", lambda s: [])
    monkeypatch.setattr(narration, "_describe_frame", lambda s: {})
    scene = _FakeScene()
    narration.narrate(scene, "câu một", "anim")
    assert scene.calls == [("play", ("anim",), 2.0)]


def test_khong_animation_van_dung_yen_va_drift_tat_mac_dinh(render, monkeypatch):
    monkeypatch.setattr(narration, "_describe_layout", lambda s: [])
    monkeypatch.setattr(narration, "_describe_frame", lambda s: {})
    scene = _FakeScene()
    narration.narrate(scene, "câu một")
    assert scene.calls == [("wait", 2.0)]


def test_drift_day_may_vao_nhe_khi_khong_co_animation(render, monkeypatch):
    monkeypatch.setattr(narration, "_describe_layout", lambda s: [])
    monkeypatch.setattr(narration, "_describe_frame", lambda s: {})
    scene = _FakeScene()
    narration.narrate(scene, "câu một", drift=True)
    assert scene.calls == [("play", (("scale", narration.DRIFT_SCALE),), 2.0)]


def test_dry_van_choi_animation_de_trang_thai_khop_luot_render(dry):
    played = []

    class S(ConceptFlowScene):
        def construct(self):
            self.narrate("một hai ba", "anim")

        def play(self, *a, **k):
            played.append((a, k))

    records = dry(S)
    assert _texts(records) == ["một hai ba"]
    assert played and played[0][0] == ("anim",) and played[0][1]["run_time"] >= 1.0


def test_recap_khong_hien_bang_va_recap_card_thi_co(dry):
    class S(ConceptFlowScene):
        def construct(self):
            self.recap(narration="Tóm lại một nửa khả năng đã mất")

    assert _texts(dry(S)) == ["Tóm lại một nửa khả năng đã mất"]
