import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { ScriptEditor } from "../../src/components/ScriptEditor";
import { HOOK_SNIPPETS } from "../../src/components/scriptTemplates";

const VALID =
  'from conceptflow import *\n\nclass DemoScene(ConceptFlowScene):\n    def construct(self):\n        self.narrate("xin chào các bạn")';

describe("ScriptEditor", () => {
  it("calls onChange when typing", () => {
    const onChange = vi.fn();
    render(<ScriptEditor value="" onChange={onChange} contentLanguage="vi" />);
    fireEvent.change(screen.getByTestId("new-project-script-textarea"), {
      target: { value: "# Scene 1" },
    });
    expect(onChange).toHaveBeenCalledWith("# Scene 1");
  });

  it("reports the narration count instead of a marker/wait tally", () => {
    // Sau CR-018 không còn gì để đếm khớp: self.narrate() gộp marker và điểm
    // chờ làm một, nên lớp lỗi "lệch số lượng" biến mất theo cấu trúc.
    vi.useFakeTimers();
    try {
      const { rerender } = render(<ScriptEditor value={VALID} onChange={vi.fn()} contentLanguage="vi" />);
      act(() => {
        vi.advanceTimersByTime(500);
      });
      expect(screen.getByTestId("script-editor-validation")).toHaveTextContent("Hợp lệ");

      rerender(
        <ScriptEditor value={"x = 1"} onChange={vi.fn()} contentLanguage="vi" />,
      );
      act(() => {
        vi.advanceTimersByTime(500);
      });
      expect(screen.getByTestId("script-editor-validation")).toHaveTextContent("class Scene");
    } finally {
      vi.useRealTimers();
    }
  });

  it("warns when the script still uses the pre-CR-018 markers", () => {
    render(
      <ScriptEditor
        value={'class A(Scene):\n    def construct(self):\n        # NARRATION: "x"\n        self.wait(AUTO)'}
        onChange={vi.fn()}
        contentLanguage="vi"
      />,
    );
    expect(screen.getByTestId("script-editor-validation")).toHaveTextContent("chuẩn cũ");
  });

  it("shows the estimated narration length before anything is rendered", () => {
    // Con số này vốn chỉ lộ ra ở bước 3 của Saga, sau khi TTS đã chạy (CR-016).
    render(<ScriptEditor value={VALID} onChange={vi.fn()} contentLanguage="vi" />);
    const estimate = screen.getByTestId("script-duration-estimate");
    expect(estimate).toHaveTextContent("giây");
    // Phải nói rõ chưa tính animation, thay vì im lặng báo thiếu (FR42.3).
    expect(estimate).toHaveTextContent("Chưa tính thời gian animation");
  });

  it("offers the snippets only once there is a script to append them to", () => {
    const onChange = vi.fn();
    const { rerender } = render(<ScriptEditor value="" onChange={onChange} contentLanguage="vi" />);
    expect(screen.queryByTestId("script-editor-insert-hook")).not.toBeInTheDocument();

    rerender(<ScriptEditor value={VALID} onChange={onChange} contentLanguage="vi" />);
    fireEvent.click(screen.getByTestId("script-editor-insert-hook"));
    expect(onChange).toHaveBeenCalledWith(`${VALID}\n${HOOK_SNIPPETS.vi}`);
  });
});
