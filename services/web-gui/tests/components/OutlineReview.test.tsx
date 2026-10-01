import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { OutlineReview } from "../../src/components/OutlineReview";
import { useOutlineReview } from "../../src/hooks/useOutlineReview";
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

/**
 * OutlineReview (danh sách) và hai nút Duyệt/Từ chối chia nhau một instance
 * `useOutlineReview` duy nhất, như ValidatePage lắp chúng. Harness dựng lại
 * cặp đó, với hai nút trơn đứng thay thanh dưới đáy màn.
 */
function Harness({ onDecided, onRejected }: { onDecided: () => void; onRejected: () => void }) {
  const outline = useOutlineReview(PROJECT, onDecided, onRejected);
  return (
    <>
      <OutlineReview project={PROJECT} outline={outline} />
      <button type="button" onClick={() => void outline.approve()} data-testid="outline-approve">
        Duyệt
      </button>
      <button type="button" onClick={() => void outline.reject()} data-testid="outline-reject">
        Từ chối
      </button>
    </>
  );
}

beforeEach(() => {
  global.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) }) as never;
});

describe("OutlineReview", () => {
  it("hiện lời thoại, beat và khung hình — không hiện code", () => {
    // Với kênh đặt trọng tâm vào ví dụ trực quan, duyệt mà chỉ đọc
    // được lời thoại là duyệt đúng nửa ít quan trọng hơn.
    render(<Harness onDecided={vi.fn()} onRejected={vi.fn()} />);

    expect(screen.getByText(/Vì sao vòng lặp này chạy mãi/)).toBeInTheDocument();
    expect(screen.getByText("hook")).toBeInTheDocument();
    expect(screen.getByText(/trên màn hình: Text×2, Arrow/)).toBeInTheDocument();
    expect(screen.queryByText(/ConceptFlowScene/)).not.toBeInTheDocument();
  });

  it("nêu cảnh báo cùng chỗ với dàn ý", () => {
    // Duyệt nội dung và duyệt cảnh báo tách làm hai lần nhìn thì lần
    // thứ hai sẽ bị bỏ qua.
    render(<Harness onDecided={vi.fn()} onRejected={vi.fn()} />);
    expect(screen.getByTestId("outline-warnings")).toHaveTextContent("chưa khai báo beat");
  });

  it("ước lượng thời lượng để Creator biết video dài bao nhiêu", () => {
    render(<Harness onDecided={vi.fn()} onRejected={vi.fn()} />);
    expect(screen.getByTestId("outline-review")).toHaveTextContent("giây");
  });

  it("bấm vào một câu là sửa được tại chỗ", async () => {
    render(<Harness onDecided={vi.fn()} onRejected={vi.fn()} />);

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

    render(<Harness onDecided={vi.fn()} onRejected={vi.fn()} />);
    fireEvent.click(screen.getByTestId("outline-line-0"));
    fireEvent.change(screen.getByTestId("outline-edit-0"), { target: { value: "x" } });
    fireEvent.click(screen.getByTestId("outline-save-0"));

    await waitFor(() =>
      expect(screen.getByTestId("outline-error")).toHaveTextContent("xuất hiện 3 lần"),
    );
  });
});

describe("useOutlineReview: duyệt / từ chối", () => {
  it("duyệt thì gọi endpoint approve", async () => {
    const onDecided = vi.fn();
    render(<Harness onDecided={onDecided} onRejected={vi.fn()} />);

    fireEvent.click(screen.getByTestId("outline-approve"));

    await waitFor(() => expect(onDecided).toHaveBeenCalled());
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/proj-1/approve"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("từ chối thì gọi endpoint reject và onRejected, không phải onDecided", async () => {
    // "Quay lại sửa script" phải gọi callback riêng, khác nút Duyệt, để trang
    // gọi nó (ValidatePage) biết phải điều hướng Creator về đâu thay vì để họ
    // kẹt lại ở một trang đã hết việc để hiện.
    const onDecided = vi.fn();
    const onRejected = vi.fn();
    render(<Harness onDecided={onDecided} onRejected={onRejected} />);

    fireEvent.click(screen.getByTestId("outline-reject"));

    await waitFor(() => expect(onRejected).toHaveBeenCalled());
    expect(onDecided).not.toHaveBeenCalled();
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/proj-1/reject"),
      expect.objectContaining({ method: "POST" }),
    );
  });
});
