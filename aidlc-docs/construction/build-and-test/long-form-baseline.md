# Pha 0 — Baseline & Benchmark cho video dài (CR-002 / CR-003)

## Date
2026-09-07

## Mục đích
Có số đo thật trên máy của Creator trước khi chốt giá trị timeout/RAM của CR-003, thay vì đoán. Kết quả **đã bác bỏ nhiều ước lượng ghi trong bản CR đầu tiên** — xem mục "Đính chính".

## Môi trường đo
| Hạng mục | Giá trị |
|---|---|
| Host | macOS (Apple Silicon), Docker Desktop |
| Container `rendering` | 10 vCPU, **7 GiB RAM tổng** |
| Manim | 0.18.1 |
| ffmpeg | 7.1.5 |

⚠️ **7 GiB là trần cứng của Docker VM** — mọi đề xuất giới hạn bộ nhớ phải nằm dưới mức này.

## Fixture
`tests/fixtures/long_form_reference.py` — 20 marker `# NARRATION:`, 20 `self.wait(AUTO)`, animation xen kẽ.
Harness: `tests/benchmark_render.py`.

Video sinh ra: **214.9s** (~3.6 phút) = 151.3s narration + **63.6s animation**.

## Kết quả 1 — Render (fixture 215s)

| Chất lượng | Wall-clock | CPU-time | CPU/wall | RAM đỉnh | Dung lượng |
|---|---|---|---|---|---|
| `-qm` 720p30 | 27.2s | 62.2s | **2.21×** | 370 MB | 1.10 MB |
| `-qh` 1080p60 | **98.9s** | — | — | **764 MB** | 2.55 MB |

**Tỉ lệ suy ra**: ở 1080p60, thời gian render ≈ **0.46× thời lượng video**.
Ngoại suy cho video 10 phút (600s) cùng độ phức tạp: **≈ 276s (4.6 phút)**, RAM ~0.8–1 GB.

## Kết quả 2 — Assembly (burn phụ đề, video 215s @1080p60)

| Cấu hình | Wall-clock | CPU/wall | RAM đỉnh | Dung lượng |
|---|---|---|---|---|
| A — `medium`/`crf 23` (hiện tại) | 24.8s | 5.61× | 923 MB | 2.75 MB |
| B — `slow`/`crf 18` (CR-004 đề xuất) | **25.9s** | 5.68× | 1002 MB | 3.52 MB |
| C — `-c:v copy` (không phụ đề) | **0.1s** | 0.99× | — | 2.55 MB |

**Ba kết luận quan trọng**:
1. Burn phụ đề ≈ **0.115× thời lượng video**. Video 10 phút ⇒ ~70s.
2. **Nâng chất lượng lên `slow`/`crf 18` gần như miễn phí về thời gian** (+1.1s, tức +4%), đổi lấy +28% dung lượng. Đề xuất của CR-004 §FR12.2 rẻ hơn nhiều so với dự đoán ban đầu.
3. `-c:v copy` nhanh hơn **~250 lần**. CR-004 §FR12.3 (tránh re-encode thừa) là tối ưu có giá trị rất cao.

## Kết quả 3 — Xác nhận lỗi đồng bộ của CR-002

Đo trực tiếp offset thật (qua `scene.renderer.time`) so với offset mà pipeline hiện tại giả định (cộng dồn thời lượng narration):

| Narration # | Offset thật | Pipeline giả định | **Lệch** |
|---|---|---|---|
| 1 | 0.00s | 0.00s | 0.00s |
| 5 | 37.43s | 27.00s | 10.43s |
| 10 | 93.47s | 65.14s | 28.32s |
| 15 | 145.37s | 101.14s | 44.22s |
| 20 | 206.07s | 144.43s | **61.64s** |

**Lỗi được xác nhận và định lượng.** Sai số tăng đơn điệu và bằng đúng thời gian animation tích luỹ. Trên video chỉ 3.6 phút, câu narration cuối đã phát **sớm hơn hình hơn 1 phút**. Video 10 phút sẽ tệ hơn tương ứng.

## Kết quả 4 — Spike 1B.1: `scene.renderer.time` HOẠT ĐỘNG ✅

Kiểm chứng trên Manim 0.18.1 bằng scene tối giản:

```
wait(2.0) → play(3.0) → wait(1.5) → play(2.5) → play(1.0) → wait(2.0)
marks thu được: 0.0, 5.0, 10.0   (dự kiến: 0, 5, 10)  ✅
ffprobe thời lượng: 12.0s        (dự kiến: 12.0)      ✅
```

Cơ chế đề xuất ở CR-002 §C1 — thay `self.wait(AUTO)` bằng `(_cf_mark(self, i), self.wait(D))` + preamble ghi `marks.jsonl` — **chạy đúng**. Giả định rủi ro nhất của CR-002 đã được gỡ bỏ.

Lưu ý: tuple được đánh giá trái→phải, nên `_cf_mark` chạy **trước** `self.wait`, tức mark = **thời điểm bắt đầu** khoảng chờ. Đây đúng là giá trị cần cho `adelay`.

---

## Đính chính các CR (số đo bác bỏ ước lượng ban đầu)

Bản CR đầu tiên viết trước khi đo, dựa trên ước lượng chung về Manim. Ba con số đã sai:

| Ghi trong CR ban đầu | Số đo thật | Đính chính |
|---|---|---|
| "Manim render 10 phút @1080p mất **20–60+ phút**" | ~4.6 phút (ngoại suy) | **Sai ~10 lần.** Fixture này nhẹ (Text/shape); script nặng (3D, nhiều MathTex, updater) sẽ chậm hơn nhiều, nhưng không tới mức đó. |
| "Burn phụ đề 1080p 10 phút mất **5–15 phút**" | ~70s (ngoại suy) | **Sai ~6 lần.** `ASSEMBLY_TIMEOUT_SECONDS=180` hiện tại thực ra **đủ dùng** cho fixture này, chỉ thiếu biên an toàn. |
| Đề xuất `RENDER_MEMORY_LIMIT_GB = 8` | Đỉnh 764 MB; VM chỉ có 7 GiB | **Bất khả thi.** 8 GiB vượt trần Docker VM. Giá trị đúng là **4 GiB**. |

### Ngược lại, hai vấn đề được xác nhận và **nghiêm trọng hơn** mô tả ban đầu

1. **`RLIMIT_CPU` là bug đang hoạt động, không phải rủi ro lý thuyết.**
   Đo được **CPU/wall = 2.21×**. Với `RLIMIT_CPU = RENDER_TIMEOUT_SECONDS = 300`, tiến trình bị `SIGXCPU` ở wall-clock ≈ **300 / 2.21 ≈ 136s**, chứ không phải 300s. Nghĩa là giới hạn render thực tế hiện nay **chỉ bằng 45% con số ghi trong config** — và đây là nguyên nhân sẽ đánh sập trước cả timeout.

2. **Lệch đồng bộ (CR-002)** — xác nhận 61.6s trên video 3.6 phút, xem Kết quả 3.

### Hạ mức ưu tiên

**RabbitMQ `consumer_timeout`** (CR-003 §C2): mặc định 30 phút. Với render ~4.6 phút cho video 10 phút, ngưỡng này **chưa bị chạm** ở workload thường. Vẫn nên đặt tường minh để có biên cho script nặng, nhưng **không còn là bug sắp nổ** như CR-003 mô tả ban đầu.

---

## Giá trị CHỐT cho CR-003 (thay cho phỏng đoán ban đầu)

| Biến | Cũ | **Mới** | Căn cứ |
|---|---|---|---|
| `RENDER_TIMEOUT_SECONDS` | 300 | **1800** | 6.5× biên an toàn trên 276s đo được cho video 10 phút |
| `RENDER_MEMORY_LIMIT_GB` | 2 (hardcode) | **4** (env) | Đỉnh 764 MB; trần VM 7 GiB nên không thể đặt 8 |
| `RLIMIT_CPU` | = timeout | **bỏ** | CPU/wall = 2.21× khiến nó luôn cắt sớm; timeout wall-clock của `subprocess.run` đã đủ chặn |
| `ASSEMBLY_TIMEOUT_SECONDS` | 180 | **900** | 12× biên trên 70s đo được; timeout chỉ có tác dụng khi hỏng nên đặt rộng là an toàn |
| RabbitMQ `consumer_timeout` | mặc định 30ph | **đặt tường minh 3600s** | Biên cho script nặng, không còn cấp bách |

## Cách chạy lại

```bash
docker cp tests/benchmark_render.py rendering:/tmp/
docker cp tests/fixtures/long_form_reference.py rendering:/tmp/
docker exec rendering python /tmp/benchmark_render.py \
    /tmp/long_form_reference.py LongFormReferenceScene -q h
```

Harness in ra JSON gồm thời gian, RAM, CPU/wall, thời lượng video, và **bảng lệch đồng bộ** — dùng chính nó làm bài kiểm tra nghiệm thu CR-002 (tiêu chí: `max_desync_seconds` < 0.2).
