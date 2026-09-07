"""Fixture tham chiếu cho Pha 0 (CR-002 / CR-003) — video dài ~5 phút.

Mục đích: đây là kịch bản chuẩn để đo và nghiệm thu mọi thay đổi của
CR-002 (đồng bộ timeline) và CR-003 (năng lực render dài). Nó cố ý có:

- 20 marker `# NARRATION:` — đủ nhiều để sai số cộng dồn của lỗi
  đồng bộ hiện tại lộ ra rõ ràng (video 2 scene KHÔNG lộ ra lỗi này,
  đó là lý do CR-001 để lọt).
- Nhiều `self.play(...)` dài xen giữa các narration — chính phần thời
  gian animation này là thứ track audio hiện tại không hề tính đến.
- Tổng thời lượng ~5 phút, đủ dài để chạm các giới hạn timeout/RAM
  mà CR-003 phải nới.

Chạy qua pipeline thật: dán nội dung file này vào Web GUI.
Chạy để benchmark trực tiếp: dùng tests/benchmark_render.py.
"""

from manim import *


class LongFormReferenceScene(Scene):
    def construct(self):
        self.camera.background_color = "#0f1117"

        title = Text("Vòng lặp for trong Java", font_size=54)
        subtitle = Text("Từ cú pháp đến độ phức tạp", font_size=32, color=GREY_B)
        subtitle.next_to(title, DOWN, buff=0.4)

        # NARRATION: "Chào mừng bạn đến với bài học về vòng lặp for trong Java."
        self.wait(AUTO)
        self.play(Write(title), run_time=2.5)
        self.play(FadeIn(subtitle, shift=UP * 0.3), run_time=1.5)

        # NARRATION: "Hôm nay chúng ta sẽ đi từ cú pháp cơ bản nhất cho tới cách đánh giá độ phức tạp thuật toán."
        self.wait(AUTO)
        self.play(FadeOut(title), FadeOut(subtitle), run_time=1.5)

        # --- Phần 1: cấu trúc ---
        heading = Text("1. Cấu trúc của vòng lặp", font_size=40).to_edge(UP)
        # NARRATION: "Trước hết, hãy nhìn vào cấu trúc của một vòng lặp for."
        self.wait(AUTO)
        self.play(Write(heading), run_time=2.0)

        boxes = VGroup(
            *[
                VGroup(
                    Rectangle(width=3.4, height=1.1, color=color, fill_opacity=0.15),
                    Text(label, font_size=26),
                )
                for label, color in [
                    ("khởi tạo", BLUE),
                    ("điều kiện", YELLOW),
                    ("cập nhật", GREEN),
                ]
            ]
        )
        for box in boxes:
            box[1].move_to(box[0].get_center())
        boxes.arrange(RIGHT, buff=0.6).shift(DOWN * 0.5)

        # NARRATION: "Một vòng lặp for gồm đúng ba thành phần, ngăn cách nhau bởi dấu chấm phẩy."
        self.wait(AUTO)
        self.play(LaggedStart(*[FadeIn(b, shift=UP * 0.4) for b in boxes], lag_ratio=0.4), run_time=3.0)

        # NARRATION: "Thành phần thứ nhất là khởi tạo. Nó chỉ chạy đúng một lần, ngay trước khi vòng lặp bắt đầu."
        self.wait(AUTO)
        self.play(boxes[0].animate.scale(1.15).set_color(BLUE_A), run_time=1.5)
        self.play(boxes[0].animate.scale(1 / 1.15).set_color(BLUE), run_time=1.5)

        # NARRATION: "Thành phần thứ hai là điều kiện. Nó được kiểm tra trước mỗi lần lặp, và khi nó sai thì vòng lặp dừng."
        self.wait(AUTO)
        self.play(boxes[1].animate.scale(1.15).set_color(YELLOW_A), run_time=1.5)
        self.play(boxes[1].animate.scale(1 / 1.15).set_color(YELLOW), run_time=1.5)

        # NARRATION: "Thành phần thứ ba là cập nhật, chạy sau mỗi lần thân vòng lặp kết thúc."
        self.wait(AUTO)
        self.play(boxes[2].animate.scale(1.15).set_color(GREEN_A), run_time=1.5)
        self.play(boxes[2].animate.scale(1 / 1.15).set_color(GREEN), run_time=1.5)

        # NARRATION: "Ba thành phần này phối hợp với nhau tạo thành một chu trình khép kín."
        self.wait(AUTO)
        arrows = VGroup(
            Arrow(boxes[0].get_right(), boxes[1].get_left(), buff=0.1),
            Arrow(boxes[1].get_right(), boxes[2].get_left(), buff=0.1),
        )
        self.play(Create(arrows), run_time=2.5)
        self.play(FadeOut(boxes), FadeOut(arrows), FadeOut(heading), run_time=1.5)

        # --- Phần 2: chạy thử ---
        heading2 = Text("2. Vòng lặp chạy như thế nào", font_size=40).to_edge(UP)
        # NARRATION: "Bây giờ hãy theo dõi từng bước một vòng lặp đếm từ không đến bốn."
        self.wait(AUTO)
        self.play(Write(heading2), run_time=2.0)

        counter_label = Text("i =", font_size=40).shift(LEFT * 1.2)
        counter = Integer(0, font_size=40).next_to(counter_label, RIGHT, buff=0.3)
        self.play(FadeIn(counter_label), FadeIn(counter), run_time=1.5)

        dots = VGroup(*[Dot(radius=0.18, color=GREY_D) for _ in range(5)])
        dots.arrange(RIGHT, buff=0.7).shift(DOWN * 1.8)
        self.play(FadeIn(dots), run_time=1.5)

        # NARRATION: "Mỗi lần điều kiện còn đúng, thân vòng lặp chạy một lần và biến đếm tăng thêm một."
        self.wait(AUTO)
        for k in range(5):
            self.play(
                counter.animate.set_value(k),
                dots[k].animate.set_color(BLUE).scale(1.4),
                run_time=1.2,
            )

        # NARRATION: "Đến khi biến đếm bằng năm, điều kiện trở thành sai và vòng lặp kết thúc."
        self.wait(AUTO)
        cross = Cross(scale_factor=0.6).next_to(dots, RIGHT, buff=0.6)
        self.play(Create(cross), run_time=2.0)
        self.play(FadeOut(counter_label), FadeOut(counter), FadeOut(dots), FadeOut(cross), FadeOut(heading2), run_time=1.5)

        # --- Phần 3: độ phức tạp ---
        heading3 = Text("3. Độ phức tạp", font_size=40).to_edge(UP)
        # NARRATION: "Phần cuối cùng, và cũng là phần quan trọng nhất khi đi phỏng vấn: độ phức tạp."
        self.wait(AUTO)
        self.play(Write(heading3), run_time=2.0)

        formula_single = MathTex(r"O(n)", font_size=72, color=GREEN)
        # NARRATION: "Một vòng lặp chạy n lần có độ phức tạp tuyến tính, ký hiệu là O của n."
        self.wait(AUTO)
        self.play(Write(formula_single), run_time=2.5)

        # NARRATION: "Nghĩa là khi dữ liệu tăng gấp đôi, thời gian chạy cũng tăng gấp đôi."
        self.wait(AUTO)
        self.play(formula_single.animate.shift(UP * 1.5).scale(0.7), run_time=2.0)

        formula_nested = MathTex(r"O(n^2)", font_size=72, color=RED)
        formula_nested.shift(DOWN * 0.8)
        # NARRATION: "Nhưng nếu bạn lồng một vòng lặp vào trong một vòng lặp khác, mọi thứ thay đổi hoàn toàn."
        self.wait(AUTO)
        self.play(Write(formula_nested), run_time=2.5)

        # NARRATION: "Lúc này dữ liệu tăng gấp đôi thì thời gian chạy tăng gấp bốn. Đây là lý do vòng lặp lồng nhau thường là thủ phạm khi chương trình chạy chậm."
        self.wait(AUTO)
        axes = Axes(x_range=[0, 5, 1], y_range=[0, 25, 5], x_length=5, y_length=3).shift(DOWN * 1.2)
        self.play(FadeOut(formula_nested), Create(axes), run_time=3.0)

        graph_linear = axes.plot(lambda x: 5 * x, color=GREEN)
        graph_quad = axes.plot(lambda x: x**2, color=RED)

        # NARRATION: "Hãy so sánh trực tiếp hai đường cong này với nhau."
        self.wait(AUTO)
        self.play(Create(graph_linear), run_time=2.5)
        self.play(Create(graph_quad), run_time=2.5)

        # NARRATION: "Đường màu đỏ vượt lên rất nhanh, và khoảng cách chỉ ngày càng nới rộng."
        self.wait(AUTO)
        self.play(Indicate(graph_quad, scale_factor=1.1), run_time=2.5)

        # NARRATION: "Nắm chắc ba thành phần và biết đánh giá độ phức tạp, bạn đã hiểu vòng lặp for tốt hơn phần lớn người mới học."
        self.wait(AUTO)
        outro = Text("Cảm ơn bạn đã theo dõi!", font_size=44)
        self.play(
            FadeOut(axes), FadeOut(graph_linear), FadeOut(graph_quad),
            FadeOut(formula_single), FadeOut(heading3),
            run_time=2.0,
        )
        self.play(Write(outro), run_time=2.5)

        # NARRATION: "Nếu thấy hữu ích, hãy đăng ký kênh để không bỏ lỡ những bài tiếp theo."
        self.wait(AUTO)
        self.play(FadeOut(outro), run_time=2.0)
