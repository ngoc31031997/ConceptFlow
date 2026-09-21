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

  const VALID_REMOTION_CODE = [
    "import {registerRoot, Composition} from 'remotion';",
    "import {calculateMetadataFromSegments, Segments} from './conceptflow-mini/segments';",
    "import {TitleText} from './conceptflow-mini/primitives';",
    "",
    'export const narrations: string[] = ["xin chào"];',
    "",
    "function CreatorComposition({segments = []}) {",
    "  return <Segments segments={segments}>{(index) => <TitleText>{narrations[index]}</TitleText>}</Segments>;",
    "}",
    "",
    'registerRoot(() => (',
    '  <Composition id="creator" component={CreatorComposition} width={1920} height={1080} fps={30} durationInFrames={150} calculateMetadata={calculateMetadataFromSegments} />',
    "));",
  ].join("\n");

  it("skips the Manim-only lint entirely for Remotion code, instead of misreporting it as invalid Manim", () => {
    render(
      <ScriptEditor
        value={VALID_REMOTION_CODE}
        onChange={vi.fn()}
        contentLanguage="vi"
        renderEngine="remotion"
      />,
    );
    // Manim's validateScript would call this "Chưa tìm thấy class Scene" —
    // that check has nothing to do with Remotion and must not appear here.
    expect(screen.getByTestId("script-editor-validation")).not.toHaveTextContent("class Scene");
    expect(screen.getByTestId("script-editor-validation")).toHaveTextContent("Hợp lệ");
    // Manim-only snippets (self.hook()/self.call_to_action()) don't apply.
    expect(screen.queryByTestId("script-editor-insert-hook")).not.toBeInTheDocument();
    expect(screen.getByText("Script Remotion (.tsx)")).toBeInTheDocument();
  });

  it("runs Remotion's own structural lint instead of accepting any non-empty text", () => {
    render(
      <ScriptEditor
        value={'export const narrations: string[] = ["xin chào"];'}
        onChange={vi.fn()}
        contentLanguage="vi"
        renderEngine="remotion"
      />,
    );
    expect(screen.getByTestId("script-editor-validation")).toHaveTextContent('id="creator"');
  });
});
