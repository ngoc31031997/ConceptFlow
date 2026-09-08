/**
 * Starter scripts offered by the "Dùng script mẫu" button.
 *
 * One per content language (CR-008 FR21.4). A Creator running an English
 * channel used to get the Vietnamese template and have to translate every
 * narration line by hand before the script was usable.
 *
 * The two are kept as separate literals rather than one parameterised template:
 * a Manim script is read and edited as a whole, and threading placeholders
 * through it would make both versions harder to read than simply having both.
 */

const VIETNAMESE_TEMPLATE = `"""
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
        self.play(*[FadeOut(m) for m in self.mobjects])`;

const ENGLISH_TEMPLATE = `"""
A Manim (Community Edition) explainer for the \`for\` loop in Java.
"""

from manim import *

# Shared palette, echoing a dark editor theme
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

        # ---------- Title ----------
        title = Text("The for loop in Java", font_size=44, color=CODE_TEXT, weight=BOLD)
        subtitle = Text(
            "Initialiser • Condition • Update",
            font_size=26, color=CODE_YELLOW
        ).next_to(title, DOWN, buff=0.3)

        self.play(Write(title))
        self.play(FadeIn(subtitle, shift=UP * 0.2))
        # NARRATION: "Let's look at the for loop in Java. Every for loop has three parts: the initialiser, the condition, and the update step."
        self.wait(AUTO)
        self.play(FadeOut(title), FadeOut(subtitle))

        # ---------- Show the code ----------
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
        # NARRATION: "Here is the syntax. The initialiser, the condition and the update step all sit compactly on a single line."
        self.wait(AUTO)

        # ---------- Label the three parts ----------
        labels = VGroup(
            Text("1. Initialiser: int i = 0", font_size=24, color=CODE_GREEN),
            Text("2. Condition: i < 5", font_size=24, color=CODE_BLUE),
            Text("3. Update: i++", font_size=24, color=CODE_RED),
        ).arrange(DOWN, aligned_edge=LEFT, buff=0.25)
        labels.next_to(code, DOWN, buff=0.6)

        for line in labels:
            self.play(FadeIn(line, shift=RIGHT * 0.3), run_time=0.5)
        # NARRATION: "So the three parts of a for loop are the initialiser, the condition, and the update."
        self.wait(AUTO)
        self.play(FadeOut(labels))

        # ---------- Animate i running 0 -> 4, printing to the console ----------
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

        # NARRATION: "The variable i runs from zero to four, and each pass through the loop prints one value to the console."
        self.wait(AUTO)
        end_text = Text(
            "i = 5 -> condition is false -> the loop stops",
            font_size=26, color=CODE_RED
        ).to_edge(DOWN, buff=0.5)
        self.play(FadeIn(end_text))
        # NARRATION: "Once i reaches five the condition is false, and the loop stops."
        self.wait(AUTO)
        self.play(*[FadeOut(m) for m in self.mobjects])`;

export const SCRIPT_TEMPLATES: Record<"vi" | "en", string> = {
  vi: VIETNAMESE_TEMPLATE,
  en: ENGLISH_TEMPLATE,
};

/**
 * Hook and end-screen snippets (CR-006 FR17).
 *
 * These are snippets the Creator inserts, not scenes the system injects. The
 * renderer requires the number of `# NARRATION:` markers to match the number of
 * `self.wait(AUTO)` calls exactly (CR-002 FR10.5), and injecting scenes around
 * a script the Creator is still editing is the easiest way to break that
 * invariant — CR-006 §C2 flagged it. A snippet carries its own matched pair, so
 * pasting it keeps the count correct by construction.
 *
 * The hook exists because roughly 70% of viewers leave in the first 15 seconds;
 * the end screen leaves the ~20 seconds YouTube's end-screen elements need.
 */
const HOOK_VI = `        # ---------- HOOK (5 giây đầu — giữ chân người xem) ----------
        hook = Text("Câu hỏi khiến 90% người mới sai", font_size=44, weight=BOLD)
        self.play(Write(hook), run_time=1.5)
        # NARRATION: "Chỉ một chi tiết nhỏ ở đây thôi, mà rất nhiều người vẫn làm sai."
        self.wait(AUTO)
        self.play(FadeOut(hook), run_time=0.8)
`;

const HOOK_EN = `        # ---------- HOOK (first 5 seconds — earn the watch) ----------
        hook = Text("The mistake almost everyone makes", font_size=44, weight=BOLD)
        self.play(Write(hook), run_time=1.5)
        # NARRATION: "There is one small detail here that a surprising number of people get wrong."
        self.wait(AUTO)
        self.play(FadeOut(hook), run_time=0.8)
`;

const END_SCREEN_VI = `        # ---------- END SCREEN (~20s cuối, chừa chỗ cho element của YouTube) ----------
        outro = Text("Cảm ơn bạn đã xem!", font_size=44)
        cta = Text("Đăng ký kênh để xem tiếp phần sau", font_size=28, color=GREY_B)
        cta.next_to(outro, DOWN, buff=0.4)
        self.play(Write(outro), FadeIn(cta, shift=UP * 0.2), run_time=2.0)
        # NARRATION: "Nếu video hữu ích, hãy đăng ký kênh để không bỏ lỡ phần tiếp theo."
        self.wait(AUTO)
        # Khoảng lặng cuối để YouTube hiển thị end-screen element.
        self.wait(8)
`;

const END_SCREEN_EN = `        # ---------- END SCREEN (last ~20s, leaving room for YouTube elements) ----------
        outro = Text("Thanks for watching!", font_size=44)
        cta = Text("Subscribe for the next one", font_size=28, color=GREY_B)
        cta.next_to(outro, DOWN, buff=0.4)
        self.play(Write(outro), FadeIn(cta, shift=UP * 0.2), run_time=2.0)
        # NARRATION: "If this was useful, subscribe so you don't miss the next one."
        self.wait(AUTO)
        # Trailing hold so YouTube's end-screen elements have somewhere to sit.
        self.wait(8)
`;

export const HOOK_SNIPPETS: Record<"vi" | "en", string> = { vi: HOOK_VI, en: HOOK_EN };
export const END_SCREEN_SNIPPETS: Record<"vi" | "en", string> = {
  vi: END_SCREEN_VI,
  en: END_SCREEN_EN,
};
