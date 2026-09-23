import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { renderHook, waitFor, cleanup } from "@testing-library/react";
import { useLlmStatus } from "../../src/hooks/useLlmStatus";
import * as client from "../../src/api/client";

// Bug thật: bước 1 có 4 tab, mỗi tab tự mount useLlmStatus(). Không cache,
// mỗi lần đổi tab (remount) status rơi về null trong lúc chờ API trả lời —
// modeLabel/aiMode ở mọi nơi đọc `llm?.enabled` coi null là "chưa bật", nên
// một project đã chọn "Gọi API trực tiếp" chớp qua "Copy prompt ra ngoài"
// mỗi khi chuyển tab. Test này mô phỏng đúng chuỗi mount → unmount → mount
// (chuyển tab) và khẳng định lần mount thứ hai không còn thấy null.
describe("useLlmStatus", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("giữ status đã biết qua lần remount tiếp theo (chuyển tab), không rơi về null", async () => {
    vi.spyOn(client, "getLlmStatus").mockResolvedValue({
      enabled: true,
      provider: "hive",
      models: [],
    });

    const first = renderHook(() => useLlmStatus());
    expect(first.result.current).toBeNull();
    await waitFor(() => expect(first.result.current?.enabled).toBe(true));
    first.unmount();

    // Remount mô phỏng chuyển sang tab khác trong bước 1.
    const second = renderHook(() => useLlmStatus());
    expect(second.result.current?.enabled).toBe(true);
  });

  it("một request lỗi thoáng qua không xoá status đã biết từ trước", async () => {
    vi.spyOn(client, "getLlmStatus").mockResolvedValueOnce({
      enabled: true,
      provider: "hive",
      models: [],
    });
    const first = renderHook(() => useLlmStatus());
    await waitFor(() => expect(first.result.current?.enabled).toBe(true));
    first.unmount();

    vi.spyOn(client, "getLlmStatus").mockRejectedValueOnce(new Error("down"));
    const second = renderHook(() => useLlmStatus());
    // Vẫn thấy giá trị cache trong lúc request lỗi này đang treo/chạy xong.
    expect(second.result.current?.enabled).toBe(true);
    await waitFor(() => expect(second.result.current?.enabled).toBe(true));
  });
});
