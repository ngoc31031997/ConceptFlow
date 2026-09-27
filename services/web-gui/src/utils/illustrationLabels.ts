import type { Illustration, ProjectIllustration } from "../api/client";

/** CR-044 — nhãn trạng thái của một hình trong thư viện. */
export function illustrationStatusLabel(ill: Pick<Illustration, "builtin" | "status"> & { exemplar?: boolean }): string {
  if (ill.exemplar) return "Hình mẫu";
  if (ill.builtin) return "Có sẵn";
  return ill.status === "approved" ? "Đã duyệt" : "Chờ duyệt";
}

/** CR-044 — mirrors domain.ProjectIllustration.Ready: may the code step go past this drawing? */
export function isProjectIllustrationReady(r: ProjectIllustration): boolean {
  if (r.state === "skipped") return true;
  if (r.state !== "drawn" && r.state !== "reused") return false;
  return !!r.illustration && (r.illustration.builtin || r.illustration.status === "approved");
}
