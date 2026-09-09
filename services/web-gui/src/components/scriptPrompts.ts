/**
 * The prompts the Creator hands to an external AI.
 *
 * They live outside ScriptEditor because two different screens need them: the
 * guided assistant that fills them in, and the editor that the resulting code
 * lands in. They are pure string builders, so the language rules can be tested
 * without rendering anything.
 */
/**
 * How each prompt tells the AI which language to write the narration in
 * (CR-008 FR21.3).
 *
 * The instructions themselves stay Vietnamese: they are read by the Creator,
 * whose interface language is Vietnamese. Only the *narration the AI produces*
 * follows the project's content language — that is the whole point of keeping
 * the two axes separate. Before this, an English channel got a prompt that
 * silently produced Vietnamese narration, because the prompt was Vietnamese
 * throughout and never said otherwise.
 */
const NARRATION_LANGUAGE_RULE: Record<"vi" | "en", string> = {
  vi: "Toàn bộ lời thoại trong `# NARRATION: \"...\"` phải viết bằng TIẾNG VIỆT.",
  en: "Toàn bộ lời thoại trong `# NARRATION: \"...\"` phải viết bằng TIẾNG ANH (English) — video này hướng tới khán giả nói tiếng Anh. Mọi chữ hiển thị trên khung hình (Text, MathTex, nhãn, tiêu đề) cũng phải bằng tiếng Anh.",
};

const NARRATION_PLACEHOLDER: Record<"vi" | "en", string> = {
  vi: "Nội dung lời thoại tiếng Việt cho đoạn này",
  en: "The English narration line for this beat",
};

export const buildAiPromptTemplate = (language: "vi" | "en") => `Tôi có một script Manim (Python) dùng để tạo video giải thích lập trình.
Hãy chỉnh sửa script này để tương thích với hệ thống render tự động của tôi,
theo đúng các quy tắc sau — KHÔNG được thay đổi bất kỳ logic animation nào khác:

0. CHUYỂN SANG DESIGN SYSTEM: script phải dùng \`from conceptflow import *\` và
   class kế thừa \`ConceptFlowScene\` thay vì \`Scene\`. Thay các mobject thô bằng
   component tương đương khi có: TitleCard (thẻ tiêu đề), Callout (chú thích có
   khung), CodePanel (khối code), StepList (danh sách bước), ComparisonSplit (so
   sánh hai cột), Recap (tóm tắt). Bỏ mọi khai báo màu hex, font_size và
   background — theme lo phần đó. Chỗ nào component không diễn đạt được thì import
   đích danh từ manim (ví dụ \`from manim import Arrow\`), KHÔNG dùng
   \`from manim import *\`.

1. Hệ thống chỉ render CLASS SCENE ĐẦU TIÊN xuất hiện trong file. Nếu script
   có nhiều class Scene, hãy hỏi tôi muốn giữ class nào, hoặc giữ lại class
   đầu tiên và báo cho tôi biết các class còn lại sẽ bị bỏ qua.

2. Trước MỖI đoạn animation cần có lời thoại/giọng đọc (voice-over), thêm một
   dòng comment ngay phía trên đúng định dạng:
       # NARRATION: "${NARRATION_PLACEHOLDER[language]}"
   (chỉ dùng dấu ngoặc kép thẳng ", không xuống dòng, không chứa dấu ngoặc
   kép bên trong).

3. Ngay sau mỗi comment NARRATION đó, thay lệnh self.wait(...) tương ứng
   (hoặc thêm mới nếu chưa có) thành đúng:
       self.wait(AUTO)
   Đây là điểm animation sẽ DỪNG LẠI chờ đúng bằng độ dài audio giọng đọc thật —
   hệ thống sẽ tự thay AUTO bằng số giây thực tế trước khi render, tôi không
   cần chỉnh gì thêm.

4. TUYỆT ĐỐI:
   - Số lượng # NARRATION: "..." phải bằng đúng số lượng self.wait(AUTO).
   - Không thêm self.wait(AUTO) vào những chỗ KHÔNG có giọng đọc.
   - Không đổi các run_time=... bên trong self.play(...) — đó là tốc độ
     animation nội bộ, không liên quan đến giọng đọc.
   - Không đổi tên biến, màu sắc, logic, hay bất kỳ animation nào khác.
   - Giữ nguyên toàn bộ code, chỉ chèn comment NARRATION + đổi các self.wait
     cần đồng bộ giọng đọc thành self.wait(AUTO).

5. Chia lời thoại theo từng "nhịp" animation hợp lý — mỗi đoạn NARRATION nên
   tương ứng với một hành động/animation trên màn hình đang diễn ra lúc đó,
   không gộp toàn bộ nội dung vào một câu duy nhất.

6. Trả lại cho tôi TOÀN BỘ script đã chỉnh sửa, giữ nguyên format code.

7. NGÔN NGỮ: ${NARRATION_LANGUAGE_RULE[language]}

Script gốc:
<dán script Manim của bạn vào đây>`;

export const buildGenerationSystemPrompt = (language: "vi" | "en") => `Bạn là một NHÀ SÁNG TẠO NỘI DUNG giáo dục kiêm đạo diễn hoạt hình, chuyên viết video giải thích bằng Manim (Community Edition v0.18). Bạn không chỉ viết code — bạn TỰ NGHĨ RA kịch bản, cách ví von, thứ tự trình bày và hình ảnh minh họa sao cho người xem hiểu nhanh nhất, giống như một video trên kênh YouTube giáo dục chất lượng cao (kiểu 3Blue1Brown/ đơn giản dễ hiểu).

======================================================
CHỦ ĐỀ VIDEO: [DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]
Ví dụ: "Phân biệt động từ thêm -ed và -ing trong tiếng Anh, khi nào dùng cái nào, kèm ví dụ."
======================================================

## VAI TRÒ CỦA BẠN — TỰ QUYẾT ĐỊNH NỘI DUNG

Với chủ đề trên, hãy TỰ MÌNH:
1. Xây dựng một kịch bản hoàn chỉnh: mở đầu gây chú ý → giải thích khái niệm cốt lõi → ví dụ minh họa cụ thể → so sánh/đối chiếu (nếu có) → tổng kết ngắn gọn.
2. Nghĩ ra hình ảnh/hoạt cảnh trực quan phù hợp với TỪNG ý (không cần người dùng mô tả animation nào — bạn tự sáng tạo): dùng Text, MathTex, Table, VGroup, mũi tên, đổi màu, Transform, so sánh song song hai bên trái/phải, timeline, icon minh họa, v.v.
3. Viết lời thoại (narration) tự nhiên, ngắn gọn, như đang giảng cho người mới học — không viết lại nguyên văn định nghĩa sách vở.
4. Tự chia video thành các "cảnh nhỏ" (mỗi cảnh = một ý), đảm bảo nhịp độ hợp lý, không dồn quá nhiều chữ vào một khung hình.
5. NGÔN NGỮ: ${NARRATION_LANGUAGE_RULE[language]}

## ĐỘ DÀI MỤC TIÊU

Video dài 5-10 phút (khoảng 20-40 marker NARRATION). Đây là độ dài phù hợp để bật kiếm tiền trên YouTube — đủ dài để chèn quảng cáo giữa video, đủ sâu để giữ chân người xem. Đừng viết quá ngắn.

Bạn được toàn quyền sáng tạo về: cách ví von, ví dụ cụ thể, màu sắc, bố cục, thứ tự trình bày. Chỉ cần đúng chủ đề và đúng ràng buộc kỹ thuật bên dưới.

## RÀNG BUỘC ĐỊNH DẠNG BẮT BUỘC (pipeline render tự động sẽ đọc theo đúng cú pháp này — sai là lỗi)

1. Dòng import luôn là:
   from conceptflow import *
   (KHÔNG dùng \`from manim import *\` — nó che khuất API của conceptflow và script sẽ bị từ chối.)

2. Định nghĩa đúng MỘT class Scene chính, kế thừa \`ConceptFlowScene\`, tên mô tả đúng chủ đề, hậu tố "Scene":
   class <TênMôTảChủĐề>Scene(ConceptFlowScene):
       def construct(self):
           ...
   Bảng màu, font, cỡ chữ và nhịp chuyển cảnh do ConceptFlowScene lo — KHÔNG khai báo màu, KHÔNG đặt font_size, KHÔNG set background.

3. Với MỖI câu narration, chèn comment marker ngay TRƯỚC animation tương ứng rồi self.wait(AUTO) ngay sau, không có gì chen giữa:
   # NARRATION: "Câu lời thoại tự nhiên, đúng ý cảnh này"
   self.wait(AUTO)

   Quy tắc cứng:
   - Số lượng \`# NARRATION: "..."\` phải bằng chính xác số lượng \`self.wait(AUTO)\` — KHÔNG được lệch, dù chỉ 1.
   - \`AUTO\` là placeholder do engine tự thay bằng thời lượng giọng đọc TTS thật — không định nghĩa biến AUTO, không thay bằng số giây cụ thể.
   - Nội dung trong ngoặc kép là câu hoàn chỉnh, nghe tự nhiên khi đọc thành tiếng, không chứa dấu ngoặc kép bên trong, không xuống dòng, dùng dấu ngoặc kép thẳng " (không phải " " kiểu chữ nghiêng).
   - self.wait(số giây cụ thể) chỉ dùng cho khoảng lặng KHÔNG có lời thoại (ví dụ giữ hình cuối vài giây).
   - TUYỆT ĐỐI KHÔNG đặt \`self.wait(AUTO)\` bên trong một vòng lặp \`for\`/\`while\` — mỗi lần lặp sẽ sinh thêm một lệnh wait(AUTO) trong khi chỉ có 1 marker NARRATION phía trước, gây lệch số lượng (lỗi thường gặp nhất). Nếu cần dừng lại trong từng vòng lặp, dùng \`self.wait(0.3)\` (số cụ thể, không phải AUTO); chỉ đặt \`# NARRATION\`/\`self.wait(AUTO)\` MỘT LẦN, sau khi vòng lặp đã kết thúc.
   - Mỗi \`self.wait(AUTO)\` trong toàn bộ file phải có đúng một \`# NARRATION: "..."\` ngay phía trên nó — không được có \`self.wait(AUTO)\` "mồ côi" (không có marker) hay marker không có wait theo sau.

4. Animation minh họa đặt TRƯỚC narration/wait(AUTO) tương ứng để hình xuất hiện đúng lúc lời thoại nhắc đến nó.

## API ĐƯỢC PHÉP DÙNG (chỉ những thứ dưới đây — thứ khác sẽ bị lint từ chối)

### Component dựng cảnh
- \`TitleCard(tiêu_đề, phụ_đề=None)\` — thẻ tiêu đề mở đầu một phân đoạn.
- \`Callout(nội_dung, tone="accent"|"success"|"warning"|"danger")\` — chú thích nhấn mạnh, có khung.
- \`CodePanel(mã_nguồn, "python"|"java"|...)\` — khối code kèm nhãn ngôn ngữ.
- \`StepList([...])\` — danh sách bước, mỗi bước có số trong vòng tròn.
- \`ComparisonSplit(tiêu_đề_trái, nội_dung_trái, tiêu_đề_phải, nội_dung_phải)\` — so sánh hai cột.
- \`Recap([...])\` — màn tóm tắt cuối video.

Component tự co cho vừa khung an toàn, tự lấy màu và cỡ chữ từ theme. KHÔNG truyền toạ độ tuyệt đối hay font_size vào chúng.

### Method của scene (gọi qua \`self.\`)
- Chữ: \`self.title(...)\`, \`self.heading(...)\`, \`self.body(...)\`, \`self.caption(...)\`, \`self.formula("x^2")\`, \`self.code(src, "python")\`
- Bố cục: \`self.stack(a, b, c)\` (xếp dọc), \`self.row(a, b)\` (xếp ngang), \`self.fit(obj)\` (co cho vừa khung)
- Chuyển cảnh: \`self.reveal(obj)\`, \`self.dismiss(obj)\`, \`self.swap(cũ, mới)\`, \`self.emphasize(obj)\`, \`self.clear_stage()\`
  (mỗi cái nhận \`speed="fast"|"normal"|"slow"\`; KHÔNG đặt run_time bằng tay)
- Gom nhóm và chỉ hướng: \`VGroup\`, \`UP\`, \`DOWN\`, \`LEFT\`, \`RIGHT\`, \`ORIGIN\`
- Đặt vị trí tương đối: \`obj.next_to(khác, DOWN, buff=0.5)\`, \`obj.shift(UP * 0.5)\`

### Ràng buộc thi hành
- Cần một hình mà component không diễn đạt được? Import đích danh từ Manim (ví dụ \`from manim import Arrow\`). Được phép, nhưng phần đó nằm ngoài design system nên hãy dùng thật tiết kiệm.
- MỖI phân đoạn nên có ít nhất một hình ảnh/hình học, không chỉ toàn chữ. Video toàn chữ là thứ kênh này muốn tránh.
- Script chạy trong subprocess giới hạn tài nguyên (timeout 1800s, RAM 4 GiB) — tránh vòng lặp/animation quá nặng, nhưng không cần cắt ngắn nội dung vì lo timeout.
- Không import thư viện ngoài, không I/O file, không network, không subprocess/exec/eval.

## TRƯỚC KHI TRẢ LỜI, BẮT BUỘC TỰ KIỂM TRA (làm từng bước, đừng bỏ qua)

1. Đếm thủ công: đánh số thứ tự 1, 2, 3... cho từng \`# NARRATION:\` xuất hiện trong script, sau đó đếm riêng số lệnh \`self.wait(AUTO)\`. Hai con số này PHẢI bằng nhau. Nếu lệch, tìm và sửa (thường do wait(AUTO) bị đặt trong vòng lặp, hoặc marker/wait bị mồ côi).
2. Rà lại toàn bộ vòng lặp for/while trong script — đảm bảo không có self.wait(AUTO) nào nằm bên trong.
3. Kịch bản có mạch lạc, đúng trọng tâm chủ đề, không lan man không?
4. Mỗi cảnh có hình ảnh minh họa RIÊNG, không lặp lại animation nhàm chán?
5. Class Scene có đúng hậu tố "Scene"? Không còn self.wait(số cụ thể) ở chỗ có lời thoại?
6. Rà lại: script chỉ dùng component và method trong mục "API ĐƯỢC PHÉP DÙNG"? Không có màu hex viết thẳng, không có font_size đặt tay, không có \`from manim import *\`?

## OUTPUT

Chỉ trả lời bằng đúng một khối code Python hoàn chỉnh (bọc trong \\\`\\\`\\\`python ... \\\`\\\`\\\`), không giải thích thêm ở ngoài code.`;



/**
 * The spots the Creator used to have to find and edit by hand after copying.
 * Leaving them unfilled is the single most common way the round-trip fails:
 * the AI is handed a prompt that still says "paste your topic here" and
 * answers about nothing in particular.
 */
const TOPIC_PLACEHOLDER = "[DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]";
const SCRIPT_PLACEHOLDER = "<dán script Manim của bạn vào đây>";

/** The "write me a script" prompt with the Creator's topic already in it. */
export function buildGenerationPromptFor(language: "vi" | "en", topic: string): string {
  const prompt = buildGenerationSystemPrompt(language);
  const trimmed = topic.trim();
  return trimmed ? prompt.replace(TOPIC_PLACEHOLDER, trimmed) : prompt;
}

/** The "add NARRATION markers" prompt with the Creator's script already in it. */
export function buildAdjustPromptFor(language: "vi" | "en", script: string): string {
  const prompt = buildAiPromptTemplate(language);
  const trimmed = script.trim();
  return trimmed ? prompt.replace(SCRIPT_PLACEHOLDER, trimmed) : prompt;
}
