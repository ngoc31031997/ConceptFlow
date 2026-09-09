"""
Video minh hoạ vòng lặp `for` trong Java, viết bằng design system ConceptFlow.

Đây là bản gốc của template khởi đầu trong
`web-gui/src/components/scriptTemplates.ts`. Giữ ở đây dưới dạng .py để lint và
test chạy được trên nó — một template mà chính lint từ chối là cách chắc chắn
nhất để Creator mất niềm tin vào cả hệ thống.
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
            "for (int i = 0; i < 5; i++) {\n"
            "    System.out.println(i);\n"
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
