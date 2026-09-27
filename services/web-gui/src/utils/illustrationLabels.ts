import type { Illustration } from "../api/client";

/** CR-044 — nhãn trạng thái của một hình trong thư viện. */
export function illustrationStatusLabel(ill: Pick<Illustration, "builtin" | "status"> & { exemplar?: boolean }): string {
  if (ill.exemplar) return "Hình mẫu";
  if (ill.builtin) return "Có sẵn";
  return ill.status === "approved" ? "Đã duyệt" : "Chờ duyệt";
}
