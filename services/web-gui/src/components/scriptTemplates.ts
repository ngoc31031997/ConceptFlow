/**
 * Starter scripts offered by the "Dùng script mẫu" button.
 *
 * Sau CR-017 các template này viết bằng design system `conceptflow`, không phải
 * API thô của Manim: `from conceptflow import *`, kế thừa `ConceptFlowScene`, và
 * dựng cảnh bằng component thay vì tự ghép mobject. Nhờ đó mọi video mang cùng
 * bảng màu, cùng thang cỡ chữ và cùng nhịp chuyển cảnh mà script không phải khai
 * báo gì — bản sắc nằm trong thư viện, không nằm trong lời dặn ở prompt.
 *
 * Sau CR-018, lời thoại là `self.narrate("...")` — một lời gọi hàm bình
 * thường, không phải comment `# NARRATION` ghép với `self.wait(AUTO)`. Bản
 * template này từng bị bỏ sót khi CR-018 gỡ chuẩn cũ (phát hiện lúc Creator
 * bấm "dùng script mẫu" và nhận về một video không có giọng đọc nào —
 * narration_segments rỗng vì không có lời gọi self.narrate() nào cả).
 *
 * Bản gốc của template tiếng Việt sống ở
 * `services/rendering/tests/fixtures/conceptflow_template.py`, nơi lint và test
 * chạy được trên nó (kể cả render thật qua Manim). Một template mà chính lint
 * từ chối là cách chắc chắn nhất để Creator mất niềm tin vào cả hệ thống.
 *
 * One per content language (CR-008 FR21.4). A Creator running an English
 * channel used to get the Vietnamese template and have to translate every
 * narration line by hand before the script was usable.
 */

const VIETNAMESE_TEMPLATE = `"""
Video minh hoạ vòng lặp \`for\` trong Java, viết bằng design system ConceptFlow.
"""

from conceptflow import *


class ForLoopIntroScene(ConceptFlowScene):
    def construct(self):
        # --- Mở đầu ---------------------------------------------------------
        card = TitleCard("Vòng lặp for trong Java", "Khởi tạo · Điều kiện · Bước nhảy")
        self.reveal(card)
        self.narrate(
            "Vòng lặp for trong Java gồm ba phần: khởi tạo biến đếm, điều kiện lặp, và bước tăng giảm."
        )
        self.dismiss(card)

        # --- Ví dụ cụ thể trước, định nghĩa sau ------------------------------
        panel = CodePanel(
            "for (int i = 0; i < 5; i++) {\\n"
            "    System.out.println(i);\\n"
            "}",
            "java",
        )
        self.reveal(panel)
        self.narrate("Nhìn vào đoạn code này trước. Nó in ra các số từ không đến bốn.")

        note = Callout("i chạy từ 0 đến 4, không tới 5", tone="warning")
        note.next_to(panel, DOWN, buff=0.5)
        self.reveal(note)
        self.narrate(
            "Điều kiện là i nhỏ hơn năm, nên vòng lặp dừng ngay khi i bằng năm. "
            "Số năm không bao giờ được in ra."
        )
        self.dismiss(panel, note)

        # --- Rút ra quy luật -------------------------------------------------
        steps = StepList([
            "Khởi tạo: chạy đúng một lần, trước mọi thứ",
            "Điều kiện: kiểm tra trước mỗi vòng",
            "Thân vòng lặp: chạy khi điều kiện còn đúng",
            "Bước nhảy: chạy sau mỗi vòng",
        ])
        self.reveal(steps)
        self.narrate(
            "Quy luật chung là bốn bước lặp lại: khởi tạo một lần, rồi kiểm tra điều kiện, "
            "chạy thân vòng lặp, và tăng biến đếm."
        )
        self.emphasize(steps)
        self.narrate("Thứ tự này quan trọng: điều kiện luôn được kiểm tra trước khi thân vòng lặp chạy.")
        self.dismiss(steps)

        # --- So sánh ---------------------------------------------------------
        compare = ComparisonSplit(
            "for", "Biết trước số vòng lặp",
            "while", "Lặp tới khi điều kiện sai",
        )
        self.reveal(compare)
        self.narrate("Khi biết trước cần lặp bao nhiêu lần thì dùng for. Khi không biết trước thì while tự nhiên hơn.")
        self.dismiss(compare)

        # --- Tóm tắt ---------------------------------------------------------
        recap = Recap([
            "for gồm bốn phần chạy theo thứ tự cố định",
            "Điều kiện kiểm tra TRƯỚC mỗi vòng",
            "Quên bước nhảy là lặp vô hạn",
        ])
        self.reveal(recap)
        self.narrate(
            "Tóm lại: for gồm bốn phần chạy theo thứ tự cố định, điều kiện luôn kiểm tra trước, "
            "và quên bước nhảy sẽ khiến vòng lặp chạy mãi."
        )
`;

const ENGLISH_TEMPLATE = `"""
A short explainer on Java's \`for\` loop, written with the ConceptFlow design system.
"""

from conceptflow import *


class ForLoopIntroScene(ConceptFlowScene):
    def construct(self):
        # --- Opening ---------------------------------------------------------
        card = TitleCard("The for loop in Java", "Init · Condition · Step")
        self.reveal(card)
        self.narrate(
            "A for loop in Java has three parts: initialising a counter, a loop condition, and a step."
        )
        self.dismiss(card)

        # --- Concrete example first, definition after ------------------------
        panel = CodePanel(
            "for (int i = 0; i < 5; i++) {\\n"
            "    System.out.println(i);\\n"
            "}",
            "java",
        )
        self.reveal(panel)
        self.narrate("Look at this snippet first. It prints the numbers zero through four.")

        note = Callout("i runs 0 to 4, never reaching 5", tone="warning")
        note.next_to(panel, DOWN, buff=0.5)
        self.reveal(note)
        self.narrate(
            "The condition is i less than five, so the loop stops the moment i becomes five. "
            "Five is never printed."
        )
        self.dismiss(panel, note)

        # --- Draw out the pattern --------------------------------------------
        steps = StepList([
            "Init: runs exactly once, before anything else",
            "Condition: checked before every pass",
            "Body: runs while the condition holds",
            "Step: runs after every pass",
        ])
        self.reveal(steps)
        self.narrate(
            "The general pattern is four steps: initialise once, then check the condition, "
            "run the body, and advance the counter."
        )
        self.emphasize(steps)
        self.narrate("The order matters: the condition is always checked before the body runs.")
        self.dismiss(steps)

        # --- Compare ----------------------------------------------------------
        compare = ComparisonSplit(
            "for", "You know the count up front",
            "while", "Loop until a condition fails",
        )
        self.reveal(compare)
        self.narrate("When you know how many passes you need, reach for a for loop. When you do not, a while loop reads more naturally.")
        self.dismiss(compare)

        # --- Recap ------------------------------------------------------------
        recap = Recap([
            "A for loop has four parts in a fixed order",
            "The condition is checked BEFORE each pass",
            "Forget the step and you loop forever",
        ])
        self.reveal(recap)
        self.narrate(
            "To recap: a for loop has four parts in a fixed order, the condition is always "
            "checked first, and forgetting the step gives you an infinite loop."
        )
`;

export const SCRIPT_TEMPLATES: Record<"vi" | "en", string> = {
  vi: VIETNAMESE_TEMPLATE,
  en: ENGLISH_TEMPLATE,
};

/**
 * Snippet hook và end screen (CR-006 FR17), viết bằng các method dựng sẵn của
 * CR-019 (`self.hook()`/`self.call_to_action()`) — chúng tự mở đúng beat
 * (`hook`/`cta`), tự dựng TitleCard, tự gọi self.narrate() và tự dismiss.
 * Trước CR-019 những method này chưa tồn tại (lời thoại còn là comment không
 * đóng gói được vào hàm), nên hai đoạn này từng phải tự ghép tay bằng
 * TitleCard + `# NARRATION` + `self.wait(AUTO)` — nay chỉ còn một dòng.
 */
const HOOK_VI = `        # ---------- HOOK (5 giây đầu — giữ chân người xem) ----------
        self.hook(
            "Viết ở đây câu hỏi hoặc nghịch lý mở đầu, đủ cụ thể để người xem muốn biết câu trả lời.",
        )
`;

const HOOK_EN = `        # ---------- HOOK (first 5 seconds — earn the watch) ----------
        self.hook(
            "Put the opening question or paradox here, concrete enough that the viewer wants the answer.",
        )
`;

const END_SCREEN_VI = `        # ---------- END SCREEN (~20s cuối, chừa chỗ cho element của YouTube) ----------
        self.call_to_action(
            "Nếu thấy hữu ích, hãy đăng ký kênh để không bỏ lỡ phần tiếp theo.",
        )
`;

const END_SCREEN_EN = `        # ---------- END SCREEN (last ~20s, leaving room for YouTube elements) ----------
        self.call_to_action(
            "If this helped, subscribe so you do not miss the next part.",
        )
`;

export const HOOK_SNIPPETS: Record<"vi" | "en", string> = { vi: HOOK_VI, en: HOOK_EN };
export const END_SCREEN_SNIPPETS: Record<"vi" | "en", string> = {
  vi: END_SCREEN_VI,
  en: END_SCREEN_EN,
};
