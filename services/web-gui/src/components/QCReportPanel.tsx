import { useEffect, useState } from "react";
import { ApiError, getQCReport } from "../api/client";
import type { QCFinding, QCReport } from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./QCReportPanel.module.css";

interface QCReportPanelProps {
  projectId: string;
  /**
   * Tua video tới giây của một phát hiện (FR61.2). Không truyền thì mốc thời
   * gian hiện ra dạng chữ thường — báo cáo vẫn đọc được khi chưa có player.
   */
  onSeek?: (seconds: number) => void;
}

/** Nhãn tiếng Việt cho từng luật. Tên luật là mã máy, không đưa thẳng ra UI. */
const RULE_LABELS: Record<string, string> = {
  frame_overflow: "Tràn khung",
  text_overlap: "Chữ chồng lấn",
  text_too_small: "Chữ quá nhỏ",
  low_contrast: "Tương phản kém",
  static_frame: "Hình đứng yên",
  narration_overlap: "Lời thoại chồng lấn",
  subtitle_cue_overlap: "Phụ đề chồng thời gian",
  loudness_off_target: "Âm lượng lệch chuẩn",
  clipping: "Âm thanh chạm trần",
  publish_attributes: "Thuộc tính phát hành",
};

function formatTimestamp(seconds: number): string {
  const safe = Number.isFinite(seconds) && seconds > 0 ? Math.floor(seconds) : 0;
  const minutes = Math.floor(safe / 60);
  return `${minutes}:${String(safe % 60).padStart(2, "0")}`;
}

function FindingRow({ finding, onSeek }: { finding: QCFinding; onSeek?: (s: number) => void }) {
  const label = RULE_LABELS[finding.rule] ?? finding.rule;
  const timestamp = formatTimestamp(finding.timestamp_seconds);

  return (
    <li className={styles.finding} data-testid={`qc-finding-${finding.rule}`}>
      {onSeek ? (
        <button
          type="button"
          className={styles.timestamp}
          data-testid={`qc-seek-${finding.rule}`}
          onClick={() => onSeek(finding.timestamp_seconds)}
          aria-label={`Tua tới ${timestamp}`}
        >
          {timestamp}
        </button>
      ) : (
        <span className={styles.timestamp}>{timestamp}</span>
      )}
      <span>
        <strong>{label}</strong> — {finding.message}
      </span>
    </li>
  );
}

/**
 * Báo cáo QC, đặt trước nút đăng (CR-021 FR61.2).
 *
 * Ba trạng thái, và khác biệt giữa chúng là điều quan trọng nhất ở đây:
 * - `has_findings`: có phát hiện, nhóm theo mức độ, lỗi chặn lên trước.
 * - `passed`: đã chấm, không có gì. Một dòng, không cần hơn.
 * - `not_scored`: KHÔNG chấm được (thiếu dữ liệu, ffmpeg lỗi). Hiện như thông
 *   tin, không phải như lỗi — FR61.4: một cổng hỏng không được biến thành cổng
 *   khoá, nên giao diện cũng không được làm nó trông như đang khoá.
 *
 * Lỗi khi tải báo cáo cũng im lặng theo đúng tinh thần đó: không có báo cáo thì
 * không hiện gì, chứ không chặn đường đăng video.
 */
export function QCReportPanel({ projectId, onSeek }: QCReportPanelProps) {
  const [report, setReport] = useState<QCReport | null>(null);

  useEffect(() => {
    let active = true;
    getQCReport(projectId)
      .then((r) => {
        if (active) setReport(r);
      })
      .catch((err) => {
        if (!(err instanceof ApiError)) throw err;
      });
    return () => {
      active = false;
    };
  }, [projectId]);

  if (!report) return null;

  /*
    Chịu được báo cáo khuyết trường: cùng lý do với FR61.4 — QC là phần phụ của
    trang này, một phản hồi lạ không được làm trắng cả màn đăng video.
  */
  const findings = Array.isArray(report.findings) ? report.findings : [];
  const blocking = findings.filter((f) => f.severity === "blocking");
  const warnings = findings.filter((f) => f.severity === "warning");

  return (
    <div className={glass.card} data-testid="qc-report">
      <div className={glass.cardHeader}>
        <span className={glass.cardTitle}>Kiểm tra chất lượng</span>
        {report.overridden_at && (
          <span className={styles.overridden} data-testid="qc-overridden">
            Đã bỏ qua
          </span>
        )}
      </div>

      {report.status === "not_scored" && (
        <p className={glass.cardHint} data-testid="qc-not-scored">
          Không chấm được lần này{report.reason ? `: ${report.reason}` : "."} Video vẫn đăng được
          bình thường.
        </p>
      )}

      {report.status === "passed" && (
        <p className={glass.cardHint} data-testid="qc-passed">
          Đã kiểm tra, không phát hiện vấn đề nào.
        </p>
      )}

      {blocking.length > 0 && (
        <section data-testid="qc-blocking-group">
          <p className={styles.groupTitle}>Lỗi nghiêm trọng ({blocking.length})</p>
          <ul className={styles.findingList}>
            {blocking.map((f, i) => (
              <FindingRow key={`${f.rule}-${i}`} finding={f} onSeek={onSeek} />
            ))}
          </ul>
        </section>
      )}

      {warnings.length > 0 && (
        <section data-testid="qc-warning-group">
          <p className={styles.groupTitle}>Cảnh báo ({warnings.length})</p>
          <ul className={styles.findingList}>
            {warnings.map((f, i) => (
              <FindingRow key={`${f.rule}-${i}`} finding={f} onSeek={onSeek} />
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}
