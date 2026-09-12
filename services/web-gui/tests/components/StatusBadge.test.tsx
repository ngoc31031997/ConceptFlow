import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "../../src/components/StatusBadge";

describe("StatusBadge", () => {
  it("hiện đúng nhãn tiếng Việt cho từng trạng thái", () => {
    render(<StatusBadge status="ready_to_publish" />);
    expect(screen.getByTestId("status-badge")).toHaveTextContent("Sẵn sàng đăng");
  });

  it("nhận diện trạng thái lỗi failed_at_* dù không có nhãn cụ thể", () => {
    render(<StatusBadge status="failed_at_render_scenes" />);
    // statusLabel đã tự thêm tiền tố "Thất bại — " cho mọi failed_at_*.
    expect(screen.getByTestId("status-badge")).toHaveTextContent("Thất bại");
  });
});
