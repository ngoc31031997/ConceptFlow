/**
 * Starter scripts offered by the "Dùng script mẫu" button.
 *
 * Sau CR-017 các template này viết bằng design system `conceptflow`, không phải
 * API thô của Manim: `from conceptflow import *`, kế thừa `ConceptFlowScene`, và
 * dựng cảnh bằng component thay vì tự ghép mobject. Nhờ đó mọi video mang cùng
 * bảng màu, cùng thang cỡ chữ và cùng nhịp chuyển cảnh mà script không phải khai
 * báo gì — bản sắc nằm trong thư viện, không nằm trong lời dặn ở prompt.
 *
 * Bản gốc của template tiếng Việt sống ở
 * `services/rendering/tests/fixtures/conceptflow_template.py`, nơi lint và test
 * chạy được trên nó. Một template mà chính lint từ chối là cách chắc chắn nhất
 * để Creator mất niềm tin vào cả hệ thống.
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
        # NARRATION: "Vòng lặp for trong Java gồm ba phần: khởi tạo biến đếm, điều kiện lặp, và bước tăng giảm."
        self.wait(AUTO)
        self.dismiss(card)

        # --- Ví dụ cụ thể trước, định nghĩa sau ------------------------------
        panel = CodePanel(
            "for (int i = 0; i < 5; i++) {\\n"
            "    System.out.println(i);\\n"
            "}",
            "java",
        )
        self.reveal(panel)
        # NARRATION: "Nhìn vào đoạn code này trước. Nó in ra các số từ không đến bốn."
        self.wait(AUTO)

        note = Callout("i chạy từ 0 đến 4, không tới 5", tone="warning")
        note.next_to(panel, DOWN, buff=0.5)
        self.reveal(note)
        # NARRATION: "Điều kiện là i nhỏ hơn năm, nên vòng lặp dừng ngay khi i bằng năm. Số năm không bao giờ được in ra."
        self.wait(AUTO)
        self.dismiss(panel, note)

        # --- Rút ra quy luật -------------------------------------------------
        steps = StepList([
            "Khởi tạo: chạy đúng một lần, trước mọi thứ",
            "Điều kiện: kiểm tra trước mỗi vòng",
            "Thân vòng lặp: chạy khi điều kiện còn đúng",
            "Bước nhảy: chạy sau mỗi vòng",
        ])
        self.reveal(steps)
        # NARRATION: "Quy luật chung là bốn bước lặp lại: khởi tạo một lần, rồi kiểm tra điều kiện, chạy thân vòng lặp, và tăng biến đếm."
        self.wait(AUTO)
        self.emphasize(steps)
        # NARRATION: "Thứ tự này quan trọng: điều kiện luôn được kiểm tra trước khi thân vòng lặp chạy."
        self.wait(AUTO)
        self.dismiss(steps)

        # --- So sánh ---------------------------------------------------------
        compare = ComparisonSplit(
            "for", "Biết trước số vòng lặp",
            "while", "Lặp tới khi điều kiện sai",
        )
        self.reveal(compare)
        # NARRATION: "Khi biết trước cần lặp bao nhiêu lần thì dùng for. Khi không biết trước thì while tự nhiên hơn."
        self.wait(AUTO)
        self.dismiss(compare)

        # --- Tóm tắt ---------------------------------------------------------
        recap = Recap([
            "for gồm bốn phần chạy theo thứ tự cố định",
            "Điều kiện kiểm tra TRƯỚC mỗi vòng",
            "Quên bước nhảy là lặp vô hạn",
        ])
        self.reveal(recap)
        # NARRATION: "Tóm lại: for gồm bốn phần chạy theo thứ tự cố định, điều kiện luôn kiểm tra trước, và quên bước nhảy sẽ khiến vòng lặp chạy mãi."
        self.wait(AUTO)
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
        # NARRATION: "A for loop in Java has three parts: initialising a counter, a loop condition, and a step."
        self.wait(AUTO)
        self.dismiss(card)

        # --- Concrete example first, definition after ------------------------
        panel = CodePanel(
            "for (int i = 0; i < 5; i++) {\\n"
            "    System.out.println(i);\\n"
            "}",
            "java",
        )
        self.reveal(panel)
        # NARRATION: "Look at this snippet first. It prints the numbers zero through four."
        self.wait(AUTO)

        note = Callout("i runs 0 to 4, never reaching 5", tone="warning")
        note.next_to(panel, DOWN, buff=0.5)
        self.reveal(note)
        # NARRATION: "The condition is i less than five, so the loop stops the moment i becomes five. Five is never printed."
        self.wait(AUTO)
        self.dismiss(panel, note)

        # --- Draw out the pattern --------------------------------------------
        steps = StepList([
            "Init: runs exactly once, before anything else",
            "Condition: checked before every pass",
            "Body: runs while the condition holds",
            "Step: runs after every pass",
        ])
        self.reveal(steps)
        # NARRATION: "The general pattern is four steps: initialise once, then check the condition, run the body, and advance the counter."
        self.wait(AUTO)
        self.emphasize(steps)
        # NARRATION: "The order matters: the condition is always checked before the body runs."
        self.wait(AUTO)
        self.dismiss(steps)

        # --- Compare ----------------------------------------------------------
        compare = ComparisonSplit(
            "for", "You know the count up front",
            "while", "Loop until a condition fails",
        )
        self.reveal(compare)
        # NARRATION: "When you know how many passes you need, reach for a for loop. When you do not, a while loop reads more naturally."
        self.wait(AUTO)
        self.dismiss(compare)

        # --- Recap ------------------------------------------------------------
        recap = Recap([
            "A for loop has four parts in a fixed order",
            "The condition is checked BEFORE each pass",
            "Forget the step and you loop forever",
        ])
        self.reveal(recap)
        # NARRATION: "To recap: a for loop has four parts in a fixed order, the condition is always checked first, and forgetting the step gives you an infinite loop."
        self.wait(AUTO)
`;

export const SCRIPT_TEMPLATES: Record<"vi" | "en", string> = {
  vi: VIETNAMESE_TEMPLATE,
  en: ENGLISH_TEMPLATE,
};

/**
 * Snippet hook và end screen (CR-006 FR17).
 *
 * Vẫn là snippet Creator tự chèn, vì bất biến "số `# NARRATION` bằng số
 * `self.wait(AUTO)`" chưa được gỡ — CR-018 sẽ gỡ, và CR-019 sẽ biến hai đoạn này
 * thành component thật. Mỗi snippet tự mang đúng một cặp marker/wait nên giữ
 * đúng số đếm theo cấu trúc.
 */
const HOOK_VI = `        # ---------- HOOK (5 giây đầu — giữ chân người xem) ----------
        hook = TitleCard("Câu hỏi khiến người xem ở lại", "Đặt vấn đề, chưa trả lời")
        self.reveal(hook)
        # NARRATION: "Viết ở đây câu hỏi hoặc nghịch lý mở đầu, đủ cụ thể để người xem muốn biết câu trả lời."
        self.wait(AUTO)
        self.dismiss(hook)
`;

const HOOK_EN = `        # ---------- HOOK (first 5 seconds — earn the watch) ----------
        hook = TitleCard("The question that keeps them watching", "Pose it, do not answer yet")
        self.reveal(hook)
        # NARRATION: "Put the opening question or paradox here, concrete enough that the viewer wants the answer."
        self.wait(AUTO)
        self.dismiss(hook)
`;

const END_SCREEN_VI = `        # ---------- END SCREEN (~20s cuối, chừa chỗ cho element của YouTube) ----------
        outro = TitleCard("Cảm ơn bạn đã xem", "Đăng ký để xem tiếp phần sau")
        self.reveal(outro)
        # NARRATION: "Nếu thấy hữu ích, hãy đăng ký kênh để không bỏ lỡ phần tiếp theo."
        self.wait(AUTO)
        # Giữ khung cuối để YouTube có chỗ hiện end-screen element. Số cụ thể,
        # không phải AUTO: đây là khoảng lặng không lời thoại.
        self.wait(8)
`;

const END_SCREEN_EN = `        # ---------- END SCREEN (last ~20s, leaving room for YouTube elements) ----------
        outro = TitleCard("Thanks for watching", "Subscribe for the next one")
        self.reveal(outro)
        # NARRATION: "If this helped, subscribe so you do not miss the next part."
        self.wait(AUTO)
        # Hold the closing frame so YouTube has room for its end-screen
        # elements. A literal, not AUTO: this is silence, not narration.
        self.wait(8)
`;

export const HOOK_SNIPPETS: Record<"vi" | "en", string> = { vi: HOOK_VI, en: HOOK_EN };
export const END_SCREEN_SNIPPETS: Record<"vi" | "en", string> = {
  vi: END_SCREEN_VI,
  en: END_SCREEN_EN,
};
