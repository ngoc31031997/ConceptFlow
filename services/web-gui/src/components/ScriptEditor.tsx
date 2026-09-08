import { useMemo, useState, type ChangeEvent } from "react";
import { END_SCREEN_SNIPPETS, HOOK_SNIPPETS, SCRIPT_TEMPLATES } from "./scriptTemplates";
import glass from "../styles/glass.module.css";
import styles from "./ScriptEditor.module.css";
import { validateScript } from "../utils/scriptValidation";

interface ScriptEditorProps {
  value: string;
  onChange: (value: string) => void;
  /** Picks which starter script "Dùng script mẫu" inserts (CR-008 FR21.4). */
  contentLanguage: "vi" | "en";
}

function UploadIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 16V4M12 4L7 9M12 4L17 9" />
      <path d="M4 17V19a2 2 0 002 2h12a2 2 0 002-2v-2" />
    </svg>
  );
}

function TemplateIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M4 9h16M9 9v11" />
    </svg>
  );
}

function SparkleIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3v4M12 17v4M3 12h4M17 12h4M6.3 6.3l2.1 2.1M15.6 15.6l2.1 2.1M6.3 17.7l2.1-2.1M15.6 8.4l2.1-2.1" />
    </svg>
  );
}

function WandIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M15 4V2M15 16v-2M8 9h2M20 9h2M17.8 11.8L19 13M17.8 6.2L19 5M12.2 6.2L11 5M12.2 11.8L11 13" />
      <path d="M3 21l9-9M12.2 15.8l4-4" />
    </svg>
  );
}

function CheckCircleIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="9" />
      <path d="M8.5 12.5l2.3 2.3L16 10" />
    </svg>
  );
}

function WarningIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3l10 18H2L12 3z" />
      <path d="M12 10v4M12 17.5v.01" />
    </svg>
  );
}

function CopyIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="12" height="12" rx="2" />
      <path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1" />
    </svg>
  );
}

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

const buildAiPromptTemplate = (language: "vi" | "en") => `Tôi có một script Manim (Python) dùng để tạo video giải thích lập trình.
Hãy chỉnh sửa script này để tương thích với hệ thống render tự động của tôi,
theo đúng các quy tắc sau — KHÔNG được thay đổi bất kỳ logic animation nào khác:

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

const buildGenerationSystemPrompt = (language: "vi" | "en") => `Bạn là một NHÀ SÁNG TẠO NỘI DUNG giáo dục kiêm đạo diễn hoạt hình, chuyên viết video giải thích bằng Manim (Community Edition v0.18). Bạn không chỉ viết code — bạn TỰ NGHĨ RA kịch bản, cách ví von, thứ tự trình bày và hình ảnh minh họa sao cho người xem hiểu nhanh nhất, giống như một video trên kênh YouTube giáo dục chất lượng cao (kiểu 3Blue1Brown/ đơn giản dễ hiểu).

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

1. Dòng đầu tiên luôn là:
   from manim import *

2. Định nghĩa đúng MỘT class Scene chính, kế thừa Scene (hoặc subclass như MovingCameraScene), tên mô tả đúng chủ đề, hậu tố "Scene":
   class <TênMôTảChủĐề>Scene(Scene):
       def construct(self):
           ...

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

## RÀNG BUỘC KỸ THUẬT

- Chỉ dùng API có sẵn của \`manim\` v0.18.x (Text, MathTex, Tex, Table, Code, VGroup, các animation Create/Write/FadeIn/FadeOut/Transform/Indicate...). Không import thư viện ngoài, không I/O file, không network, không subprocess/exec/eval.
- Script chạy trong subprocess giới hạn tài nguyên (timeout 1800s, RAM 4 GiB) — tránh vòng lặp/animation quá nặng, nhưng không cần cắt ngắn nội dung vì lo timeout.
- Output cuối là video .mp4 khi render bằng: manim -qm <file> <TênScene>.
- Nền tối mặc định của Manim, chọn màu chữ/hình có độ tương phản tốt, bố cục nằm gọn trong khung an toàn 16:9, không để chữ/hình tràn hoặc chồng lấp.

## TRƯỚC KHI TRẢ LỜI, BẮT BUỘC TỰ KIỂM TRA (làm từng bước, đừng bỏ qua)

1. Đếm thủ công: đánh số thứ tự 1, 2, 3... cho từng \`# NARRATION:\` xuất hiện trong script, sau đó đếm riêng số lệnh \`self.wait(AUTO)\`. Hai con số này PHẢI bằng nhau. Nếu lệch, tìm và sửa (thường do wait(AUTO) bị đặt trong vòng lặp, hoặc marker/wait bị mồ côi).
2. Rà lại toàn bộ vòng lặp for/while trong script — đảm bảo không có self.wait(AUTO) nào nằm bên trong.
3. Kịch bản có mạch lạc, đúng trọng tâm chủ đề, không lan man không?
4. Mỗi cảnh có hình ảnh minh họa RIÊNG, không lặp lại animation nhàm chán?
5. Class Scene có đúng hậu tố "Scene"? Không còn self.wait(số cụ thể) ở chỗ có lời thoại?

## OUTPUT

Chỉ trả lời bằng đúng một khối code Python hoàn chỉnh (bọc trong \\\`\\\`\\\`python ... \\\`\\\`\\\`), không giải thích thêm ở ngoài code.`;


type PromptPanel = "system" | "adjust" | null;

export function ScriptEditor({ value, onChange, contentLanguage }: ScriptEditorProps) {
  // Rebuilt when the Creator switches content language, so the prompt they copy
  // always asks for narration in the language the video is actually for.
  const aiPromptTemplate = useMemo(() => buildAiPromptTemplate(contentLanguage), [contentLanguage]);
  const generationSystemPrompt = useMemo(
    () => buildGenerationSystemPrompt(contentLanguage),
    [contentLanguage],
  );
  const [activePanel, setActivePanel] = useState<PromptPanel>(null);
  const [copied, setCopied] = useState(false);
  const [systemPromptCopied, setSystemPromptCopied] = useState(false);

  function togglePanel(panel: PromptPanel) {
    setActivePanel((current) => (current === panel ? null : panel));
  }

  const validation = useMemo(() => validateScript(value), [value]);

  function handleFileImport(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => onChange(String(reader.result ?? ""));
    reader.readAsText(file);
    event.target.value = "";
  }

  async function handleCopyPrompt() {
    try {
      await navigator.clipboard.writeText(aiPromptTemplate);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  async function handleCopySystemPrompt() {
    try {
      await navigator.clipboard.writeText(generationSystemPrompt);
      setSystemPromptCopied(true);
      setTimeout(() => setSystemPromptCopied(false), 2000);
    } catch {
      setSystemPromptCopied(false);
    }
  }

  return (
    <div className={glass.card}>
      <div className={glass.cardHeader}>
        <div className={glass.cardTitle}>Script Manim (.py)</div>
        <div className={styles.headerActions}>
          <details className={styles.toolsMenu}>
            <summary className={glass.ghostBtn}>
              <SparkleIcon />
              Công cụ AI
            </summary>
            <div className={styles.toolsDropdown}>
              <button
                type="button"
                data-testid="script-editor-system-prompt-toggle"
                className={styles.toolsDropdownItem}
                onClick={() => togglePanel("system")}
              >
                <WandIcon />
                System Prompt (tạo mới)
              </button>
              <button
                type="button"
                data-testid="script-editor-ai-prompt-toggle"
                className={styles.toolsDropdownItem}
                onClick={() => togglePanel("adjust")}
              >
                <SparkleIcon />
                Prompt điều chỉnh script
              </button>
              <button type="button" className={styles.toolsDropdownItem} onClick={() => onChange(SCRIPT_TEMPLATES[contentLanguage])}>
                <TemplateIcon />
                Dùng script mẫu
              </button>
              <button
                type="button"
                className={styles.toolsDropdownItem}
                data-testid="script-editor-insert-hook"
                onClick={() => onChange(`${value}\n${HOOK_SNIPPETS[contentLanguage]}`)}
              >
                <TemplateIcon />
                Chèn hook mở đầu
              </button>
              <button
                type="button"
                className={styles.toolsDropdownItem}
                data-testid="script-editor-insert-end-screen"
                onClick={() => onChange(`${value}\n${END_SCREEN_SNIPPETS[contentLanguage]}`)}
              >
                <TemplateIcon />
                Chèn end screen
              </button>
            </div>
          </details>
          <label className={glass.ghostBtn}>
            <UploadIcon />
            Nhập từ file
            <input type="file" accept=".py" onChange={handleFileImport} className={styles.hiddenFileInput} />
          </label>
        </div>
      </div>

      {activePanel === "system" && (
        <div className={styles.promptPanel} data-testid="script-editor-system-prompt-panel">
          <div className={styles.promptPanelHeader}>
            <span>System prompt để nhờ AI viết script Manim MỚI từ đầu (đúng chuẩn NARRATION/wait(AUTO) của hệ thống)</span>
            <button type="button" className={glass.ghostBtn} onClick={handleCopySystemPrompt}>
              <CopyIcon />
              {systemPromptCopied ? "Đã copy!" : "Copy"}
            </button>
          </div>
          <textarea
            className={`${glass.textArea} ${styles.promptTextarea}`}
            data-testid="script-editor-system-prompt-textarea"
            value={generationSystemPrompt}
            readOnly
            rows={10}
          />
        </div>
      )}

      {activePanel === "adjust" && (
        <div className={styles.promptPanel} data-testid="script-editor-ai-prompt-panel">
          <div className={styles.promptPanelHeader}>
            <span>Dán prompt này kèm script gốc của bạn vào một AI bất kỳ để tự động gắn marker NARRATION</span>
            <button type="button" className={glass.ghostBtn} onClick={handleCopyPrompt}>
              <CopyIcon />
              {copied ? "Đã copy!" : "Copy"}
            </button>
          </div>
          <textarea
            className={`${glass.textArea} ${styles.promptTextarea}`}
            data-testid="script-editor-ai-prompt-textarea"
            value={aiPromptTemplate}
            readOnly
            rows={10}
          />
        </div>
      )}

      <textarea
        className={`${glass.textArea} ${styles.textarea}`}
        data-testid="new-project-script-textarea"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={
          'class DemoScene(Scene):\n    def construct(self):\n        # NARRATION: "Loi thoai cho canh nay"\n        self.wait(AUTO)'
        }
        rows={16}
      />

      {value.trim().length > 0 && (
        <div
          className={validation.isValid ? styles.validationOk : styles.validationError}
          data-testid="script-editor-validation"
        >
          {validation.isValid ? (
            <>
              <CheckCircleIcon />
              Hợp lệ: {validation.narrationCount} đoạn NARRATION khớp {validation.autoWaitCount} self.wait(AUTO).
            </>
          ) : (
            <>
              <WarningIcon />
              {validation.message}
            </>
          )}
        </div>
      )}

      <div className={glass.cardHint}>
        Viết script Manim bình thường. Đặt <code>{'# NARRATION: "..."'}</code> ngay trước mỗi{" "}
        <code>self.wait(AUTO)</code> — hệ thống sẽ tạo giọng đọc cho từng đoạn và tự thay <code>AUTO</code> bằng
        thời lượng thật của giọng đọc trước khi render.
      </div>
    </div>
  );
}
