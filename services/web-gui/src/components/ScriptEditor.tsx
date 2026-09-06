import { useState, type ChangeEvent } from "react";
import glass from "../styles/glass.module.css";
import styles from "./ScriptEditor.module.css";

interface ScriptEditorProps {
  value: string;
  onChange: (value: string) => void;
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

function CopyIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="12" height="12" rx="2" />
      <path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1" />
    </svg>
  );
}

const AI_PROMPT_TEMPLATE = `Tôi có một script Manim (Python) dùng để tạo video giải thích lập trình.
Hãy chỉnh sửa script này để tương thích với hệ thống render tự động của tôi,
theo đúng các quy tắc sau — KHÔNG được thay đổi bất kỳ logic animation nào khác:

1. Hệ thống chỉ render CLASS SCENE ĐẦU TIÊN xuất hiện trong file. Nếu script
   có nhiều class Scene, hãy hỏi tôi muốn giữ class nào, hoặc giữ lại class
   đầu tiên và báo cho tôi biết các class còn lại sẽ bị bỏ qua.

2. Trước MỖI đoạn animation cần có lời thoại/giọng đọc (voice-over), thêm một
   dòng comment ngay phía trên đúng định dạng:
       # NARRATION: "Nội dung lời thoại tiếng Việt cho đoạn này"
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

Script gốc:
<dán script Manim của bạn vào đây>`;

const SCRIPT_TEMPLATE = `"""
Video minh họa vòng lặp \`for\` trong Java bằng Manim (Community Edition).
"""

from manim import *

# Bảng màu dùng chung, gợi cảm giác theme editor tối
BG_DARK = "#1e1e2e"
CODE_BLUE = "#89b4fa"
CODE_GREEN = "#a6e3a1"
CODE_YELLOW = "#f9e2af"
CODE_RED = "#f38ba8"
CODE_TEXT = "#cdd6f4"
CONSOLE_BG = "#11111b"


class ForLoopIntroScene(Scene):
    def construct(self):
        self.camera.background_color = BG_DARK

        # ---------- Tiêu đề ----------
        title = Text("Vòng lặp for trong Java", font_size=44, color=CODE_TEXT, weight=BOLD)
        subtitle = Text(
            "Khởi tạo • Điều kiện • Bước tăng/giảm",
            font_size=26, color=CODE_YELLOW
        ).next_to(title, DOWN, buff=0.3)

        self.play(Write(title))
        self.play(FadeIn(subtitle, shift=UP * 0.2))
        # NARRATION: "Giới thiệu vòng lặp for trong Java. Vòng lặp for gồm ba phần: khởi tạo biến đếm, điều kiện lặp, và bước tăng giảm."
        self.wait(AUTO)
        self.play(FadeOut(title), FadeOut(subtitle))

        # ---------- Hiển thị đoạn code ----------
        code_str = (
            "for (int i = 0; i < 5; i++) {\\n"
            "    System.out.println(i);\\n"
            "}"
        )
        code = Code(
            code=code_str,
            language="java",
            style="monokai",
            background="rectangle",
            background_stroke_color=CODE_BLUE,
            corner_radius=0.15,
        ).scale(0.9)
        code.to_edge(UP, buff=1.0)

        self.play(FadeIn(code, shift=UP * 0.3))
        # NARRATION: "Đây là cú pháp vòng lặp for: khởi tạo, điều kiện, và bước nhảy được viết gọn trên cùng một dòng."
        self.wait(AUTO)

        # ---------- Chú thích 3 phần của for ----------
        labels = VGroup(
            Text("① Khởi tạo: int i = 0", font_size=24, color=CODE_GREEN),
            Text("② Điều kiện: i < 5", font_size=24, color=CODE_BLUE),
            Text("③ Bước nhảy: i++", font_size=24, color=CODE_RED),
        ).arrange(DOWN, aligned_edge=LEFT, buff=0.25)
        labels.next_to(code, DOWN, buff=0.6)

        for line in labels:
            self.play(FadeIn(line, shift=RIGHT * 0.3), run_time=0.5)
        # NARRATION: "Ba phần của vòng lặp for là khởi tạo, điều kiện, và bước nhảy."
        self.wait(AUTO)
        self.play(FadeOut(labels))

        # ---------- Hoạt hình biến i chạy 0 -> 4 + in ra console ----------
        i_label = Text("i =", font_size=32, color=CODE_TEXT)
        i_value = Integer(0, font_size=36, color=CODE_YELLOW)
        i_group = VGroup(i_label, i_value).arrange(RIGHT, buff=0.2)
        i_group.next_to(code, DOWN, buff=0.8).shift(LEFT * 3)

        console_box = RoundedRectangle(
            width=4.5, height=3.2, corner_radius=0.15,
            color=GRAY, fill_color=CONSOLE_BG, fill_opacity=1
        ).next_to(code, DOWN, buff=0.8).shift(RIGHT * 3)
        console_title = Text("Console", font_size=20, color=GRAY).next_to(
            console_box, UP, buff=0.15
        ).align_to(console_box, LEFT)

        self.play(FadeIn(i_group), Create(console_box), FadeIn(console_title))

        printed_lines = VGroup()
        for value in range(5):
            self.play(i_value.animate.set_value(value), run_time=0.4)
            self.play(Indicate(code.code[1], color=CODE_YELLOW, scale_factor=1.05), run_time=0.4)

            line = Text(str(value), font_size=28, color=CODE_GREEN)
            printed_lines.add(line)
            printed_lines.arrange(DOWN, aligned_edge=LEFT, buff=0.15)
            printed_lines.move_to(console_box.get_top() + DOWN * 0.5, aligned_edge=UP).align_to(
                console_box, LEFT
            ).shift(RIGHT * 0.3)

            self.play(FadeIn(line, shift=UP * 0.15), run_time=0.4)

        # NARRATION: "Biến i chạy từ 0 đến 4, mỗi vòng lặp in ra một giá trị ra console."
        self.wait(AUTO)
        end_text = Text(
            "i = 5 → điều kiện sai → vòng lặp dừng",
            font_size=26, color=CODE_RED
        ).to_edge(DOWN, buff=0.5)
        self.play(FadeIn(end_text))
        # NARRATION: "Khi i bằng 5, điều kiện sai, vòng lặp dừng lại."
        self.wait(AUTO)
        self.play(*[FadeOut(m) for m in self.mobjects])
`;

export function ScriptEditor({ value, onChange }: ScriptEditorProps) {
  const [showPrompt, setShowPrompt] = useState(false);
  const [copied, setCopied] = useState(false);

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
      await navigator.clipboard.writeText(AI_PROMPT_TEMPLATE);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className={glass.card}>
      <div className={glass.cardHeader}>
        <div className={glass.cardTitle}>Script Manim (.py)</div>
        <div className={styles.headerActions}>
          <button
            type="button"
            data-testid="script-editor-ai-prompt-toggle"
            className={glass.ghostBtn}
            onClick={() => setShowPrompt((v) => !v)}
          >
            <SparkleIcon />
            Prompt mẫu (AI)
          </button>
          <button type="button" className={glass.ghostBtn} onClick={() => onChange(SCRIPT_TEMPLATE)}>
            <TemplateIcon />
            Dùng mẫu
          </button>
          <label className={glass.ghostBtn}>
            <UploadIcon />
            Nhập từ file
            <input type="file" accept=".py" onChange={handleFileImport} className={styles.hiddenFileInput} />
          </label>
        </div>
      </div>

      {showPrompt && (
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
            value={AI_PROMPT_TEMPLATE}
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
      <div className={glass.cardHint}>
        Viết script Manim bình thường. Đặt <code>{'# NARRATION: "..."'}</code> ngay trước mỗi{" "}
        <code>self.wait(AUTO)</code> — hệ thống sẽ tạo giọng đọc cho từng đoạn và tự thay <code>AUTO</code> bằng
        thời lượng thật của giọng đọc trước khi render.
      </div>
    </div>
  );
}
