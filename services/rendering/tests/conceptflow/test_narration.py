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


def test_clip_ghi_mot_ban_ghi_o_luot_dry_voi_t_null(dry):
    """CR-007 FR19.2: ở lượt dry chưa có thời gian thật — Creator chỉ cần thấy
    clip nào được định cắt, để CR-024 duyệt dàn ý trước khi tốn TTS."""

    class S(ConceptFlowScene):
        def construct(self):
            with self.clip("ví dụ chạy thật"):
                self.narrate("bên trong clip")

    records = dry(S)
    clips = [r for r in records if r["kind"] == "clip"]
    assert len(clips) == 1
    assert clips[0]["name"] == "ví dụ chạy thật"
    assert clips[0]["index"] == 0
    assert clips[0]["t_start"] is None
    assert clips[0]["t_end"] is None


def test_clip_khong_pha_index_cua_narrate_ben_trong(dry):
    """`clip()` phải dùng chung `_recorder` singleton với `narrate()`, không
    được tạo state đếm riêng — nếu không thứ tự/số lượng lời thoại sẽ lệch."""

    class S(ConceptFlowScene):
        def construct(self):
            self.narrate("trước clip")
            with self.clip("đoạn giữa"):
                self.narrate("một")
                self.narrate("hai")
            self.narrate("sau clip")

    records = dry(S)
    assert _texts(records) == ["trước clip", "một", "hai", "sau clip"]
    clip = next(r for r in records if r["kind"] == "clip")
    # index gắn vào lời thoại SẮP chạy khi clip mở ra (upcoming_index), giống
    # beat()/chapter() — ở đây là "một", tức lời thoại thứ 1 (0-based).
    assert clip["index"] == 1


def test_clip_long_nhau_bi_tu_choi(dry):
    class S(ConceptFlowScene):
        def construct(self):
            with self.clip("ngoài"):
                with self.clip("trong"):
                    pass

    with pytest.raises(narration.NarrationError):
        dry(S)


def test_clip_dong_du_thi_mo_clip_khac_duoc(dry):
    """Không lồng nhau, nhưng hai clip liên tiếp (đóng xong mới mở) là hợp lệ."""

    class S(ConceptFlowScene):
        def construct(self):
            with self.clip("một"):
                self.narrate("a")
            with self.clip("hai"):
                self.narrate("b")

    records = dry(S)
    clips = [r for r in records if r["kind"] == "clip"]
    assert [c["name"] for c in clips] == ["một", "hai"]


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
    # Mỗi mốc kèm đúng một bản ghi bố cục cho QC (CR-021 FR58.1).
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


def test_clip_o_luot_render_ghi_t_start_nho_hon_t_end(tmp_path, monkeypatch):
    """CR-007 FR19.2: ở lượt render, t_start/t_end là mốc thật đọc từ
    `scene.renderer.time`, không phải null như lượt dry.

    Gọi thẳng `narration.clip()` với một scene giả có `renderer.time` tăng dần
    — tránh phải dựng lại toàn bộ máy render thật của Manim chỉ để chứng minh
    hai lần đọc `renderer.time` (lúc mở và lúc đóng) được ghi đúng thứ tự.
    """
    marks = tmp_path / "cf_marks.jsonl"
    monkeypatch.setenv(narration.ENV_MARKS_PATH, str(marks))
    monkeypatch.setenv(narration.ENV_MODE, narration.MODE_RENDER)
    narration.reset()

    class _FakeRenderer:
        def __init__(self) -> None:
            self._t = 0.0

        @property
        def time(self) -> float:
            self._t += 1.0
            return self._t

    class _FakeScene:
        renderer = _FakeRenderer()

    with narration.clip(_FakeScene(), "ví dụ chạy thật"):
        pass

    records = [json.loads(line) for line in marks.read_text().splitlines() if line]
    assert len(records) == 1
    clip = records[0]
    assert clip["kind"] == "clip"
    assert clip["name"] == "ví dụ chạy thật"
    assert clip["t_start"] is not None
    assert clip["t_end"] is not None
    assert clip["t_start"] < clip["t_end"]


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


# --- CR-021 FR58: thu dữ liệu bố cục ở lượt render ---------------------------


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
    # luật tương phản FR59.4 sống hay chết ở con số này.
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
    """FR58.2: dữ liệu QC không bao giờ được đắt hơn cái video nó đang chấm."""

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
