# Chạy ConceptFlow trên máy Windows làm server (`.env.window`)

Hướng dẫn này dành cho việc host toàn bộ pipeline trên một máy Windows cấu
hình phổ thông (tham khảo: Asus FX503VD — i5-7300HQ 2 lõi/4 luồng, 16GB RAM,
GTX 1050 4GB, 225GB SSD + ~932GB HDD), khi **chất lượng video quan trọng hơn
thời gian render**.

## Vì sao cần `.env.window` riêng

Máy dev gốc benchmark timeout/tài nguyên (`RENDER_TIMEOUT_SECONDS`,
`ASSEMBLY_TIMEOUT_SECONDS`, `RENDER_MEMORY_LIMIT_GB`, các `cpus`/`memory`
limit trong `docker-compose.yml`) trên CPU nhiều luồng hơn. Trên CPU 2
lõi/4 luồng, các mốc mặc định dễ khiến render/assembly bị kill giữa chừng
trước khi xong.

`.env.window` chỉ nới **timeout và trần tài nguyên** — không đổi bất kỳ
tham số ảnh hưởng chất lượng nào:

- `RENDER_QUALITY` giữ `1080p60` (đổi sang `4k60` nếu muốn, chấp nhận lâu
  hơn nữa).
- ffmpeg assembly vẫn `-preset slow -crf 18` (hardcode trong
  `services/video-assembly/adapters/assembly/ffmpeg_assembler.py`, không đổi
  theo host).

Các biến trên đã được parameterize hóa trong `docker-compose.yml` (đọc từ
env, có default y hệt cấu hình gốc) nên không ảnh hưởng gì nếu bạn chạy
`docker compose` không kèm `--env-file .env.window`.

## Các biến trong `.env.window`

| Biến | Mặc định gốc | Giá trị trong `.env.window` | Lý do |
|---|---|---|---|
| `RENDER_TIMEOUT_SECONDS` | 1800 | 10800 | CPU 2 lõi render chậm hơn nhiều so với máy đo benchmark (10 phút video @1080p60 ~276s trên máy gốc) |
| `DRY_RUN_TIMEOUT_SECONDS` | 2000 | 7200 | Dry-run chạy toàn bộ animation ở quality thấp nhất, vẫn tốn CPU tương đương |
| `RENDER_MEMORY_LIMIT_GB` | 4 | 4 | Giữ nguyên — RLIMIT_AS của tiến trình render, không liên quan CPU |
| `RENDER_CPU_LIMIT` | 6.0 | 4.0 | Máy chỉ có 4 luồng thật, đặt đúng thực tế tránh Docker lập lịch sai kỳ vọng |
| `RENDER_MEMORY_LIMIT` | 5G | 4.5G | Trần cgroup Docker cho container rendering |
| `ASSEMBLY_TIMEOUT_SECONDS` | 900 | 3600 | `-preset slow` vốn đã chậm hơn `medium`, cộng CPU yếu hơn |
| `ASSEMBLY_CPU_LIMIT` | 6.0 | 4.0 | Tương tự rendering |
| `ASSEMBLY_MEMORY_LIMIT` | 3G | 3G | Giữ nguyên |
| `OLLAMA_MODEL` | llama3.2 | llama3.2 | Model nhỏ, chạy CPU-only vẫn chấp nhận được (GTX 1050 4GB không được pass GPU vào container Ollama trong compose hiện tại) |

`RABBITMQ_USER/PASS`, `POSTGRES_USER/PASS`, `GRAFANA_USER/PASS` và các key
OAuth/TTS/Hive vẫn cần điền như `.env.example` — `.env.window` chỉ thêm các
biến tài nguyên/timeout ở trên.

## Cách chạy

1. Cài Docker Desktop (bật WSL2 backend) hoặc Docker Engine trong WSL2 trên
   máy Windows — WSL2 đỡ overhead RAM hơn Hyper-V thuần trên máy 16GB.
2. Copy/clone repo sang máy đó.
3. Mở `.env.window`, điền giá trị thật cho `RABBITMQ_USER/PASS`,
   `POSTGRES_USER/PASS`, `GRAFANA_USER/PASS` (và OAuth/TTS/Hive key nếu
   dùng).
4. Build và chạy, chỉ định `.env.window` bằng cờ `--env-file`:
   ```powershell
   docker compose --env-file .env.window up -d --build
   ```
5. Kiểm tra trạng thái, đợi `rendering` và `video-assembly` báo `healthy`
   (build lần đầu và render lần đầu sẽ lâu hơn máy dev gốc):
   ```powershell
   docker compose --env-file .env.window ps
   ```
6. **Mọi lệnh compose sau này** (`logs`, `down`, `restart <service>`, ...)
   đều phải kèm `--env-file .env.window` — thiếu cờ này, compose đọc `.env`
   mặc định (rỗng hoặc không tồn tại) và quay về timeout ngắn của máy gốc.
7. Muốn khỏi gõ cờ mỗi lần: trên máy Windows đó, copy/đổi tên
   `.env.window` thành `.env` (không đụng `.env` gốc trên máy dev khác).

## Nếu vẫn bị timeout

Thu log của service bị lỗi rồi nới thêm timeout tương ứng:

```powershell
docker compose --env-file .env.window logs rendering
docker compose --env-file .env.window logs video-assembly
```

Không hạ `RENDER_QUALITY` hay sửa `crf`/`preset` trong
`ffmpeg_assembler.py` để "chữa" timeout — đó là đánh đổi chất lượng, chỉ nên
làm nếu chủ động muốn giảm chất lượng, không phải cách vá lỗi timeout.
