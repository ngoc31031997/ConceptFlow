import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { OutlineReview } from "../../src/components/OutlineReview";
import type { Project } from "../../src/types";

const PROJECT: Project = {
  project_id: "proj-1",
  status: "awaiting_review",
  voice_language: "vi",
  scenes: [
    { scene_index: 0, narration_text: "Vì sao vòng lặp này chạy mãi", visual: "Text×2, Arrow" },
    { scene_index: 1, narration_text: "Thiếu bước nhảy nên điều kiện không bao giờ sai", visual: "khung trống" },
  ],
  beats: [
    { scene_index: 0, id: "hook" },
    { scene_index: 1, id: "concrete" },
  ],
  validation_warnings: ["script chưa khai báo beat nào"],
};

beforeEach(() => {
  global.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) }) as never;
});

describe("OutlineReview", () => {
  it("hiện lời thoại, beat và khung hình — không hiện code", () => {
    // FR68.4/68.5: với kênh đặt trọng tâm vào ví dụ trực quan, duyệt mà chỉ đọc
    // được lời thoại là duyệt đúng nửa ít quan trọng hơn.
    render(<OutlineReview project={PROJECT} onDecided={vi.fn()} onRejected={vi.fn()} />);

    expect(screen.getByText(/Vì sao vòng lặp này chạy mãi/)).toBeInTheDocument();
    expect(screen.getByText("hook")).toBeInTheDocument();
    expect(screen.getByText(/trên màn hình: Text×2, Arrow/)).toBeInTheDocument();
    expect(screen.queryByText(/ConceptFlowScene/)).not.toBeInTheDocument();
  });

  it("nêu cảnh báo cùng chỗ với dàn ý", () => {
    // FR68.3: duyệt nội dung và duyệt cảnh báo tách làm hai lần nhìn thì lần
    // thứ hai sẽ bị bỏ qua.
    render(<OutlineReview project={PROJECT} onDecided={vi.fn()} onRejected={vi.fn()} />);
    expect(screen.getByTestId("outline-warnings")).toHaveTextContent("chưa khai báo beat");
  });

  it("ước lượng thời lượng để Creator biết video dài bao nhiêu", () => {
    render(<OutlineReview project={PROJECT} onDecided={vi.fn()} onRejected={vi.fn()} />);
    expect(screen.getByTestId("outline-review")).toHaveTextContent("giây");
  });

  it("duyệt thì gọi endpoint approve", async () => {
    const onDecided = vi.fn();
    render(<OutlineReview project={PROJECT} onDecided={onDecided} onRejected={vi.fn()} />);

    fireEvent.click(screen.getByTestId("outline-approve"));

    await waitFor(() => expect(onDecided).toHaveBeenCalled());
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/proj-1/approve"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("từ chối thì gọi endpoint reject và onRejected, không phải onDecided", async () => {
    // Bug đã sửa: "Quay lại sửa script" trước đây gọi cùng callback với nút
    // Duyệt, nên trang gọi nó (RenderPage) không có cách nào biết phải điều
    // hướng Creator về đâu — họ kẹt lại ở một trang đã hết việc để hiện.
    const onDecided = vi.fn();
    const onRejected = vi.fn();
    render(<OutlineReview project={PROJECT} onDecided={onDecided} onRejected={onRejected} />);

    fireEvent.click(screen.getByTestId("outline-reject"));

    await waitFor(() => expect(onRejected).toHaveBeenCalled());
    expect(onDecided).not.toHaveBeenCalled();
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/proj-1/reject"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("bấm vào một câu là sửa được tại chỗ", async () => {
    render(<OutlineReview project={PROJECT} onDecided={vi.fn()} onRejected={vi.fn()} />);

    fireEvent.click(screen.getByTestId("outline-line-0"));
    fireEvent.change(screen.getByTestId("outline-edit-0"), {
      target: { value: "Câu đã sửa" },
    });
    fireEvent.click(screen.getByTestId("outline-save-0"));

    await waitFor(() =>
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/v1/projects/proj-1/narration"),
        expect.objectContaining({ method: "POST" }),
      ),
    );
  });

  it("hiện nguyên văn lý do khi server từ chối sửa", async () => {
    // Server trả 422 khi câu này không truy ngược được về một chỗ trong script
    // (vòng lặp, f-string). Nuốt thông báo đi thì Creator không hiểu vì sao.
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ error: "câu này xuất hiện 3 lần trong script" }),
    }) as never;

    render(<OutlineReview project={PROJECT} onDecided={vi.fn()} onRejected={vi.fn()} />);
    fireEvent.click(screen.getByTestId("outline-line-0"));
    fireEvent.change(screen.getByTestId("outline-edit-0"), { target: { value: "x" } });
    fireEvent.click(screen.getByTestId("outline-save-0"));

    await waitFor(() =>
      expect(screen.getByTestId("outline-error")).toHaveTextContent("xuất hiện 3 lần"),
    );
  });
});
