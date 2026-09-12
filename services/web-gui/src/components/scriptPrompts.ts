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
import { WORDS_PER_MINUTE, type ContentLanguage } from "../utils/durationEstimate";
import type { VideoFormat } from "../types";

/**
 * Beat sheet đưa vào prompt dưới dạng **ngân sách từ**, không phải ngân sách
 * phút (CR-019 FR54.1).
 *
 * Model đếm được từ; nó không đếm được giây. Nói "beat này 40–70 giây" là giao
 * cho model một phép quy đổi mà nó không có cơ sở để làm, nên kết quả trôi rất
 * xa. Quy đổi sẵn ở đây bằng chính tốc độ đọc hệ thống sẽ dùng — kể cả tốc độ
 * đã hiệu chỉnh theo giọng Creator chọn (FR54.3).
 */
export function buildBeatSheetSection(
  format: VideoFormat,
  language: ContentLanguage,
  wordsPerMinute?: number,
): string {
  const wpm = wordsPerMinute ?? WORDS_PER_MINUTE[language] ?? WORDS_PER_MINUTE.en;
  const words = (seconds: number) => Math.round((seconds * wpm) / 60);

  const rows = format.beats
    .map((beat) => {
      const repeat = beat.max_repeat > 1 ? ` (lặp tối đa ${beat.max_repeat} lần)` : "";
      const required = beat.required ? "BẮT BUỘC" : "tuỳ chọn";
      return `   - \`self.beat("${beat.id}")\` — ${required}${repeat}: khoảng ${words(
        beat.min_seconds,
      )}–${words(beat.max_seconds)} từ lời thoại`;
    })
    .join("\n");

  return `## CẤU TRÚC VIDEO BẮT BUỘC — format "${format.name}"

Gọi \`self.beat("<id>")\` ngay trước đoạn mở đầu mỗi phần, theo ĐÚNG thứ tự dưới đây:

${rows}

Quy tắc cứng:
   - Beat ghi BẮT BUỘC mà thiếu thì hệ thống DỪNG trước khi tạo giọng đọc — script không render được.
   - \`concrete\` phải đứng TRƯỚC \`pattern\`: cho người xem thấy một ví dụ chạy thật rồi mới rút ra quy luật. Đây là điểm khác biệt của kênh, không phải sở thích trình bày.
   - Ngân sách từ ở trên là để canh nhịp, lệch chút không sao; thứ tự và các beat bắt buộc thì không được lệch.
`;
}

const NARRATION_LANGUAGE_RULE: Record<"vi" | "en", string> = {
  vi: "Toàn bộ lời thoại trong `self.narrate(\"...\")` phải viết bằng TIẾNG VIỆT.",
  en: "Toàn bộ lời thoại trong `self.narrate(\"...\")` phải viết bằng TIẾNG ANH (English) — video này hướng tới khán giả nói tiếng Anh. Mọi chữ hiển thị trên khung hình (Text, MathTex, nhãn, tiêu đề) cũng phải bằng tiếng Anh.",
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

   Nếu script đang dùng quy ước CŨ (comment \`# NARRATION: "..."\` kèm
   \`self.wait(AUTO)\`) — quy ước đó ĐÃ BỊ GỠ BỎ khỏi hệ thống, script dùng nó sẽ
   không tạo được giọng đọc nào cả. Chuyển mỗi cặp comment+wait đó thành một lời
   gọi hàm \`self.narrate("...")\` duy nhất, đặt đúng vị trí \`self.wait(AUTO)\`
   cũ (xem quy tắc 2 bên dưới).

1. Hệ thống chỉ render CLASS SCENE ĐẦU TIÊN xuất hiện trong file. Nếu script
   có nhiều class Scene, hãy hỏi tôi muốn giữ class nào, hoặc giữ lại class
   đầu tiên và báo cho tôi biết các class còn lại sẽ bị bỏ qua.

2. Với MỖI đoạn animation cần có lời thoại/giọng đọc (voice-over), gọi ngay
   tại điểm đó (không phải comment — một lời gọi hàm bình thường trong
   construct()):
       self.narrate("${NARRATION_PLACEHOLDER[language]}")
   Lời gọi này tự dừng animation lại đúng bằng độ dài audio giọng đọc thật —
   không cần (và không được) thêm self.wait(...) ngay sau nó. Dùng được cả bên
   trong vòng lặp for/while, nhánh điều kiện, hay hàm helper.

3. TUYỆT ĐỐI:
   - KHÔNG dùng comment \`# NARRATION:\` hay \`self.wait(AUTO)\` dưới bất kỳ hình
     thức nào — chúng không còn được hệ thống đọc.
   - Không đổi các run_time=... bên trong self.play(...) — đó là tốc độ
     animation nội bộ, không liên quan đến giọng đọc.
   - Không đổi tên biến, màu sắc, logic, hay bất kỳ animation nào khác.
   - Giữ nguyên toàn bộ code, chỉ thay các điểm cần giọng đọc bằng
     self.narrate(...).

4. Chia lời thoại theo từng "nhịp" animation hợp lý — mỗi lời gọi
   self.narrate(...) nên tương ứng với một hành động/animation trên màn hình
   đang diễn ra lúc đó, không gộp toàn bộ nội dung vào một câu duy nhất.

5. Trả lại cho tôi TOÀN BỘ script đã chỉnh sửa, giữ nguyên format code.

6. NGÔN NGỮ: ${NARRATION_LANGUAGE_RULE[language]}

7. TRƯỚC KHI TRẢ LỜI, BẮT BUỘC TỰ KIỂM TRA (làm từng bước, đừng bỏ qua):
   - Tìm lại trong TOÀN BỘ script (kể cả bên trong vòng lặp, hàm helper, nhánh
     if): còn sót chuỗi \`# NARRATION\` hay \`wait(AUTO)\` nào không? Script dài
     rất dễ sót vài chỗ ở giữa hoặc cuối file — rà đến hết, không chỉ vài dòng
     đầu.
   - Nếu CÒN SÓT dù chỉ một chỗ — script sẽ bị hệ thống từ chối ngay khi dán
     vào. Sửa hết trước khi trả lời, không trả lời một phần rồi hẹn sửa tiếp.
   - Số lượng \`self.narrate(...)\` có khớp đúng số điểm cần giọng đọc trong
     script gốc không (không thiếu, không thừa)?

Script gốc:
<dán script Manim của bạn vào đây>`;

export const buildGenerationSystemPrompt = (
  language: "vi" | "en",
  format?: VideoFormat,
  wordsPerMinute?: number,
) => `Bạn là một NHÀ SÁNG TẠO NỘI DUNG giáo dục kiêm đạo diễn hoạt hình, chuyên viết video giải thích bằng Manim (Community Edition v0.18). Bạn không chỉ viết code — bạn TỰ NGHĨ RA kịch bản, cách ví von, thứ tự trình bày và hình ảnh minh họa sao cho người xem hiểu nhanh nhất, giống như một video trên kênh YouTube giáo dục chất lượng cao (kiểu 3Blue1Brown/ đơn giản dễ hiểu).

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

${format ? buildBeatSheetSection(format, language, wordsPerMinute) : `## ĐỘ DÀI MỤC TIÊU

Video dài 6-8 phút. Kênh còn mới nên thứ cần tối ưu là tỉ lệ giữ chân người xem, không phải mốc 8 phút để chèn quảng cáo giữa video — một video 12 phút loãng tệ hơn hẳn một video 7 phút chặt.`}

Bạn được toàn quyền sáng tạo về: cách ví von, ví dụ cụ thể, thứ tự trình bày trong từng beat. Màu sắc và bố cục thì KHÔNG — design system lo phần đó.

## RÀNG BUỘC ĐỊNH DẠNG BẮT BUỘC (pipeline render tự động sẽ đọc theo đúng cú pháp này — sai là lỗi)

1. Dòng import luôn là:
   from conceptflow import *
   (KHÔNG dùng \`from manim import *\` — nó che khuất API của conceptflow và script sẽ bị từ chối.)

2. Định nghĩa đúng MỘT class Scene chính, kế thừa \`ConceptFlowScene\`, tên mô tả đúng chủ đề, hậu tố "Scene":
   class <TênMôTảChủĐề>Scene(ConceptFlowScene):
       def construct(self):
           ...
   Bảng màu, font, cỡ chữ và nhịp chuyển cảnh do ConceptFlowScene lo — KHÔNG khai báo màu, KHÔNG đặt font_size, KHÔNG set background.

3. Lời thoại là một LỜI GỌI HÀM, không phải comment. Ngay tại điểm cần giọng đọc:
   self.narrate("Câu lời thoại tự nhiên, đúng ý cảnh này")

   Quy tắc cứng:
   - \`self.narrate(...)\` tự dừng animation đúng bằng thời lượng giọng đọc TTS thật. KHÔNG thêm \`self.wait(...)\` ngay sau nó.
   - Nội dung trong ngoặc kép là câu hoàn chỉnh, nghe tự nhiên khi đọc thành tiếng, không chứa dấu ngoặc kép bên trong, không xuống dòng, dùng dấu ngoặc kép thẳng " (không phải " " kiểu chữ nghiêng).
   - \`self.wait(số giây cụ thể)\` chỉ dùng cho khoảng lặng KHÔNG có lời thoại (ví dụ giữ hình cuối vài giây).
   - Gọi được ở MỌI nơi: bên trong vòng lặp \`for\`/\`while\`, trong nhánh \`if\`, trong hàm helper. Không có ràng buộc về số lượng hay vị trí — danh sách lời thoại được lấy theo thứ tự chạy thật.
   - TUYỆT ĐỐI KHÔNG dùng comment \`# NARRATION: "..."\` hay \`self.wait(AUTO)\`. Quy ước cũ đó đã bị gỡ khỏi hệ thống: script dùng nó sẽ không sinh ra lời thoại nào và bị từ chối với lỗi "narration_segments must not be empty".

4. Animation minh họa đặt TRƯỚC lời gọi \`self.narrate(...)\` tương ứng, để hình xuất hiện đúng lúc lời thoại nhắc đến nó.

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
- Lời thoại và cấu trúc: \`self.narrate("câu lời thoại")\`, \`self.beat("<id>")\`, \`self.chapter("Tên chapter")\`
- Ba beat dựng sẵn — DÙNG CHÚNG thay vì tự dựng lại bằng tay, chúng đã tự gọi \`self.beat(...)\` tương ứng bên trong:
  - \`self.hook("Câu hỏi mở đầu", "phụ đề tuỳ chọn")\` — mở beat \`hook\`
  - \`self.recap(["ý 1", "ý 2"], title="Tóm lại")\` — mở beat \`recap\`
  - \`self.call_to_action("Lời kêu gọi", "phụ đề tuỳ chọn")\` — mở beat \`cta\`, tự giữ khung cuối cho end-screen
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

1. Tìm trong script: có còn chuỗi \`# NARRATION\` hoặc \`wait(AUTO)\` nào không? Nếu CÓ — dù chỉ một — script sẽ bị từ chối. Thay hết bằng \`self.narrate("...")\`.
2. Mỗi lời thoại có phải một lời gọi \`self.narrate("...")\` đặt ngay SAU animation minh họa cho nó không? Có \`self.wait(...)\` nào bị thêm thừa ngay sau một lời gọi narrate không (không được — narrate đã tự chờ)?
3. Kịch bản có mạch lạc, đúng trọng tâm chủ đề, không lan man không?
4. Mỗi cảnh có hình ảnh minh họa RIÊNG, không lặp lại animation nhàm chán?
5. Class Scene có đúng hậu tố "Scene" và kế thừa \`ConceptFlowScene\` không?
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
export function buildGenerationPromptFor(
  language: "vi" | "en",
  topic: string,
  format?: VideoFormat,
  wordsPerMinute?: number,
): string {
  const prompt = buildGenerationSystemPrompt(language, format, wordsPerMinute);
  const trimmed = topic.trim();
  return trimmed ? prompt.replace(TOPIC_PLACEHOLDER, trimmed) : prompt;
}

/** The "add NARRATION markers" prompt with the Creator's script already in it. */
export function buildAdjustPromptFor(language: "vi" | "en", script: string): string {
  const prompt = buildAiPromptTemplate(language);
  const trimmed = script.trim();
  return trimmed ? prompt.replace(SCRIPT_PLACEHOLDER, trimmed) : prompt;
}

/**
 * CR-026 FR70 — a short-form (Shorts/TikTok) script is NOT a trimmed-down
 * long-form video: it must stand on its own, hook in the first 2 seconds,
 * one idea, no filler. And it is NOT a summary of the long-form script's
 * Manim CODE either (that would mean summarizing animation instructions,
 * which does not mean anything) — `sourceTopic`, when given, is only handed
 * over as *context* for what the short should also be about, never as code
 * to transform.
 *
 * The one hard requirement beyond the usual design-system rules: the entire
 * `construct()` body must be wrapped in exactly one
 * `with self.clip("short"):` — CR-007's existing `generate_clips` picks up
 * anything wrapped like that with zero new code on the rendering/orchestrator
 * side. Without this line the render still succeeds, it just produces no
 * clip to publish, which is a `Chưa có clip nào` message with no obvious
 * cause days later — not a normal way to lose this feature.
 */
export const buildShortScriptSystemPrompt = (
  language: "vi" | "en",
  sourceTopic?: string,
) => `Bạn là một NHÀ SÁNG TẠO NỘI DUNG giáo dục, chuyên viết video ngắn (YouTube Shorts/TikTok) bằng Manim (Community Edition v0.18). Đây KHÔNG PHẢI bản rút gọn của một video dài — video này phải tự đứng được một mình, không cần xem gì khác trước đó.

======================================================
CHỦ ĐỀ VIDEO: ${sourceTopic?.trim() ? sourceTopic.trim() : "[DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]"}
======================================================

## RÀNG BUỘC NỘI DUNG — khác hẳn video dài

1. Độ dài mục tiêu: 30–60 giây lời thoại — ĐÚNG MỘT Ý, không có đoạn "khởi
   động" hay hạ nhiệt giữa video như bản dài.
2. Hook trong 2 GIÂY ĐẦU — câu đầu tiên phải khiến người xem dừng lướt, không
   phải một câu giới thiệu chung chung.
3. Không cố nhồi nhiều ý — một video dài có thể có nhiều phần, video ngắn
   không có chỗ cho việc đó. Chọn ĐÚNG MỘT lát cắt hay nhất của chủ đề.

## RÀNG BUỘC ĐỊNH DẠNG BẮT BUỘC (pipeline render tự động sẽ đọc theo đúng cú pháp này — sai là lỗi)

1. Dòng import luôn là:
   from conceptflow import *

2. Định nghĩa đúng MỘT class Scene chính, kế thừa \`ConceptFlowScene\`:
   class <TênMôTảChủĐề>Scene(ConceptFlowScene):
       def construct(self):
           ...

3. **BẮT BUỘC, khác bản dài**: TOÀN BỘ nội dung bên trong \`construct()\` phải
   nằm trong ĐÚNG MỘT khối:
       with self.clip("short"):
           ...toàn bộ animation và self.narrate(...) ở đây...
   Đây là điều kiện DUY NHẤT để hệ thống nhận ra đây là một clip dọc
   Shorts/TikTok — thiếu dòng này, video vẫn render được nhưng KHÔNG có clip
   nào xuất ra, và không có cảnh báo nào khác ngoài "Chưa có clip nào" ở màn
   kết quả.

4. Lời thoại là một LỜI GỌI HÀM: \`self.narrate("Câu lời thoại tự nhiên")\` —
   không dùng comment, không thêm \`self.wait(...)\` ngay sau nó.

5. NGÔN NGỮ: ${NARRATION_LANGUAGE_RULE[language]}

## API ĐƯỢC PHÉP DÙNG (giống hệt bản dài — chỉ những thứ dưới đây)

Component: \`TitleCard\`, \`Callout\`, \`CodePanel\`, \`StepList\`, \`ComparisonSplit\`, \`Recap\`.
Method: \`self.narrate(...)\`, \`self.hook(...)\`, \`self.call_to_action(...)\`,
\`self.title/heading/body/caption/formula/code(...)\`, \`self.stack/row/fit(...)\`,
\`self.reveal/dismiss/swap/emphasize/clear_stage(...)\`. KHÔNG đặt màu/font_size
bằng tay — theme lo phần đó.

## TRƯỚC KHI TRẢ LỜI, BẮT BUỘC TỰ KIỂM TRA

1. Toàn bộ \`construct()\` có nằm trong ĐÚNG MỘT \`with self.clip("short"):\` không?
2. Câu lời thoại đầu tiên có phải một hook thật sự, không phải câu giới thiệu chung chung?
3. Tổng lời thoại có nằm trong khoảng 30–60 giây không (ước lượng theo tốc độ đọc bình thường)?
4. Có đúng MỘT ý duy nhất, không lan man sang ý khác?

## OUTPUT

Chỉ trả lời bằng đúng một khối code Python hoàn chỉnh (bọc trong \\\`\\\`\\\`python ... \\\`\\\`\\\`), không giải thích thêm ở ngoài code.`;
