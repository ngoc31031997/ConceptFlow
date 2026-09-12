import { useState } from "react";
import type { Project } from "../types";
import { Disclosure } from "./Disclosure";
import glass from "../styles/glass.module.css";
import styles from "./ProjectInputPanel.module.css";

interface ProjectInputPanelProps {
  project: Project;
}

const RENDER_QUALITY_LABELS: Record<string, string> = {
  "480p15": "Test (480p15)",
  "720p30": "Nháp (720p30)",
  "1080p60": "Chuẩn (1080p60)",
  "4k60": "Cao (4K60)",
};

const SUBTITLE_MODE_LABELS: Record<string, string> = {
  off: "Tắt",
  track: "Track (CC riêng trên YouTube)",
  burn_in: "Burn-in (khắc thẳng vào hình)",
  both: "Cả hai",
};

/**
 * Toàn bộ input đã dùng để tạo project này — không chỉ script mà cả mọi lựa
 * chọn cấu hình đi kèm (giọng, phụ đề, chất lượng, nhạc nền, định dạng).
 * Backend luôn trả các trường này về (dùng để dựng lại saga khi "Render lại
 * ở chất lượng khác"), nhưng trước đây ResultPage chỉ dùng chúng âm thầm chứ
 * chưa từng hiện ra — Creator muốn xem lại toàn bộ input để tối ưu hoặc tái
 * sử dụng cho video sau, kể cả sau khi đã đăng.
 */
export function ProjectInputPanel({ project }: ProjectInputPanelProps) {
  const [copied, setCopied] = useState(false);

  async function handleCopyScript() {
    if (!project.script_content) return;
    try {
      await navigator.clipboard.writeText(project.script_content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API có thể bị chặn (context không an toàn, quyền trình
      // duyệt) — im lặng bỏ qua, script vẫn chọn/copy tay được từ textarea.
    }
  }

  const qualityLabel = project.render_quality
    ? RENDER_QUALITY_LABELS[project.render_quality] ?? project.render_quality
    : "Mặc định";
  const subtitleLabel = project.subtitle_mode
    ? SUBTITLE_MODE_LABELS[project.subtitle_mode] ?? project.subtitle_mode
    : "Không rõ";

  return (
    <Disclosure
      title="Toàn bộ input đã dùng"
      hint="Xem lại kịch bản và cấu hình gốc để tối ưu hoặc dùng lại cho video sau."
      testId="project-input"
    >
      <dl className={styles.grid}>
        <dt>Ngôn ngữ nội dung</dt>
        <dd>{project.voice_language === "en" ? "Tiếng Anh" : "Tiếng Việt"}</dd>

        <dt>Giọng đọc</dt>
        <dd>
          {project.tts_enabled === false
            ? "Không dùng giọng đọc"
            : project.voice_id ?? "Mặc định theo ngôn ngữ"}
        </dd>

        <dt>Phụ đề</dt>
        <dd>{subtitleLabel}</dd>

        <dt>Chất lượng render</dt>
        <dd>{qualityLabel}</dd>

        <dt>Định dạng video</dt>
        <dd>{project.video_format_id ?? "Mặc định"}</dd>

        <dt>Nhạc nền</dt>
        <dd>
          {project.background_music_path
            ? `${project.background_music_path}${
                project.background_music_volume !== undefined
                  ? ` (âm lượng ${Math.round(project.background_music_volume * 100)}%)`
                  : ""
              }`
            : "Không có"}
        </dd>
      </dl>

      <div className={styles.scriptHeader}>
        <span className={styles.scriptLabel}>Script gốc</span>
        {project.script_content && (
          <button type="button" className={glass.ghostBtn} onClick={handleCopyScript}>
            {copied ? "Đã copy!" : "Copy script"}
          </button>
        )}
      </div>
      {project.script_content ? (
        <textarea
          className={glass.textArea}
          readOnly
          value={project.script_content}
          rows={16}
          data-testid="project-input-script"
          onFocus={(e) => e.currentTarget.select()}
        />
      ) : (
        <p className={glass.helperText} role="alert">
          Project này không còn lưu script gốc — có thể được tạo trước khi trường này được ghi lại,
          hoặc dữ liệu đã bị dọn.
        </p>
      )}
    </Disclosure>
  );
}
