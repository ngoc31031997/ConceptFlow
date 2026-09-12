import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ShortScriptAssistant } from "../../src/components/ShortScriptAssistant";

function renderAssistant(onCreated = vi.fn()) {
  render(
    <ShortScriptAssistant
      sourceProjectId="proj-long"
      sourceScriptContent="from conceptflow import *\n..."
      contentLanguage="vi"
      onCreated={onCreated}
    />,
  );
  return { onCreated };
}

describe("ShortScriptAssistant", () => {
  afterEach(() => vi.restoreAllMocks());

  it('copy prompt bọc sẵn chủ đề và yêu cầu with self.clip("short")', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderAssistant();

    fireEvent.change(screen.getByTestId("short-script-topic"), {
      target: { value: "Vì sao vòng lặp for chạy đúng 5 lần" },
    });
    fireEvent.click(screen.getByTestId("short-script-copy"));

    await waitFor(() => expect(writeText).toHaveBeenCalled());
    const copiedPrompt = writeText.mock.calls[0][0] as string;
    expect(copiedPrompt).toContain("Vì sao vòng lặp for chạy đúng 5 lần");
    expect(copiedPrompt).toContain('with self.clip("short"):');
  });

  it("soạn bằng AI nội bộ đổ kết quả vào ô nháp, không tự nộp", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ script_content: "from conceptflow import *\n# nháp AI" }),
    }) as never;
    const { onCreated } = renderAssistant();

    fireEvent.change(screen.getByTestId("short-script-topic"), { target: { value: "chủ đề" } });
    fireEvent.click(screen.getByTestId("short-script-suggest-ai"));

    await waitFor(() =>
      expect(screen.getByTestId("short-script-draft")).toHaveValue("from conceptflow import *\n# nháp AI"),
    );
    expect(onCreated).not.toHaveBeenCalled();
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/short-script-suggestions"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("hiện lỗi rõ ràng khi AI nội bộ thất bại", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ error: "ollama unreachable" }),
    }) as never;
    renderAssistant();

    fireEvent.change(screen.getByTestId("short-script-topic"), { target: { value: "chủ đề" } });
    fireEvent.click(screen.getByTestId("short-script-suggest-ai"));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("ollama unreachable"));
  });

  it("nộp bản nháp thì tạo project mới với video_output_mode=short và companion_project_id đúng", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ saga_id: "saga-1", status: "parsing_script" }),
    }) as never;
    const { onCreated } = renderAssistant();

    fireEvent.change(screen.getByTestId("short-script-draft"), {
      target: { value: "from conceptflow import *\n..." },
    });
    fireEvent.click(screen.getByTestId("short-script-submit"));

    await waitFor(() => expect(onCreated).toHaveBeenCalled());
    const [, init] = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    const body = JSON.parse(init.body as string);
    expect(body.video_output_mode).toBe("short");
    expect(body.companion_project_id).toBe("proj-long");
    expect(body.script_content).toBe("from conceptflow import *\n...");
  });

  it("không cho nộp khi ô nháp còn trống", () => {
    renderAssistant();
    expect(screen.getByTestId("short-script-submit")).toBeDisabled();
  });
});
