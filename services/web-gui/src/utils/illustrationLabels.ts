import type { DrawProgress, Illustration, ProjectIllustration } from "../api/client";
import { formatChars, formatClock } from "../lib/formatProgress";

/** Nhãn trạng thái của một hình trong thư viện. */
export function illustrationStatusLabel(ill: Pick<Illustration, "builtin" | "status"> & { exemplar?: boolean }): string {
  if (ill.exemplar) return "Hình mẫu";
  if (ill.builtin) return "Có sẵn";
  return ill.status === "approved" ? "Đã duyệt" : "Chờ duyệt";
}

/** Mirrors domain.ProjectIllustration.Ready: may the code step go past this drawing? */
export function isProjectIllustrationReady(r: ProjectIllustration): boolean {
  if (r.state === "skipped") return true;
  if (r.state !== "drawn" && r.state !== "reused") return false;
  return !!r.illustration && (r.illustration.builtin || r.illustration.status === "approved");
}

/** "Lần 1/3 · AI đang viết code · 3,2k ký tự · 42s". */
export function drawProgressText(p: DrawProgress): string {
  const phase =
    p.phase === "checking"
      ? "Đang kiểm tra và dựng ảnh"
      : p.phase === "writing"
        ? `AI đang viết code · ${formatChars(p.content_chars)} ký tự`
        : p.phase === "reasoning"
          ? `AI đang phân tích · ${formatChars(p.reasoning_chars)} ký tự`
          : "Đang chờ AI";
  const attempt = p.attempt > 0 ? `Lần ${p.attempt}/${p.max_attempts} · ` : "";
  return `${attempt}${phase} · ${formatClock(p.elapsed_seconds)}`;
}
