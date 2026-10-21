import type { AuthoringModelOption, AuthoringStepModels, ModelUsageStats } from "../api/client";
import { Dropdown } from "./ui";
import { LOCAL_CODE_MODEL_WARNING } from "./localModelWarning";
import glass from "../styles/glass.module.css";
import styles from "./AuthoringModelPicker.module.css";

const STEP_ORDER: { key: keyof AuthoringStepModels; label: string }[] = [
  { key: "story", label: "Kịch bản" },
  { key: "storyboard", label: "Visual" },
  { key: "code", label: "Code" },
];

interface AuthoringModelPickerProps {
  models: AuthoringStepModels;
  onChange: (models: AuthoringStepModels) => void;
  /** Danh mục model server cho phép chọn (từ GET /v1/llm/status). */
  options: AuthoringModelOption[];
  /** Model mặc định của từng bước ở Cài đặt; bước thiếu dùng model đầu danh mục. */
  stepDefaults?: Record<string, string>;
  disabled?: boolean;
  /** Chi phí đã đo của từng model ở bước Code (GET /v1/llm/status). */
  codeStats?: Record<string, ModelUsageStats>;
  codeStatsError?: string;
}

const FAILURE_LABEL: Record<string, string> = {
  empty: "trả về rỗng",
  server: "lỗi máy chủ",
  timeout: "hết giờ",
  budget: "nghĩ hết token",
  truncated: "bị cắt",
  balance: "hết số dư",
  rate_limit: "quá tốc độ",
  auth: "key bị từ chối",
  malformed: "sai định dạng",
};
// Những lỗi nói rằng model không hợp với bước Code, không phải lỗi thoáng qua.
const UNFIT = ["timeout", "budget", "truncated"];

/** Một dòng số liệu cho model đang chọn ở bước Code. */
export function codeStatsLine(stats: ModelUsageStats | undefined): { text: string; warn: boolean } {
  if (!stats || stats.calls === 0) return { text: "Chưa có số liệu đo cho model này ở bước Code.", warn: false };
  const parts = [`Đo 30 ngày: ${stats.ok}/${stats.calls} đoạn code đạt`];
  if (stats.ok > 0) {
    const tokens = Math.round(stats.avg_completion_tokens / 1000);
    const minutes = (stats.avg_duration_ms / 60000).toFixed(1).replace(".", ",");
    parts.push(`TB ${tokens}k token, ${minutes} phút mỗi đoạn`);
  }
  const failures = Object.entries(stats.failures ?? {}).filter(([, n]) => n > 0);
  if (failures.length > 0) {
    parts.push("lỗi: " + failures.map(([k, n]) => `${FAILURE_LABEL[k] ?? k} ${n}`).join(", "));
  }
  return { text: parts.join(" · "), warn: failures.some(([k]) => UNFIT.includes(k)) };
}

/**
 * Model-per-step picker — Creator chọn model Hive cho từng tab (1a/1b/1c)
 * ngay ở bước 1, một lần, thay vì hệ thống lúc nào cũng gọi đúng một model
 * cố định (HIVE_MODEL). Đặt cạnh AuthoringModeBar vì cùng là quyết định "cách
 * làm cả bước 3", chỉ hiện khi Creator đã chọn "Gọi API trực tiếp" — chưa
 * chọn API thì chưa có model nào để chọn.
 *
 * Lựa chọn áp dụng ngay khi gọi generateAuthoringStep thật (server đọc lại
 * từ project_authoring, không phải giá trị FE gửi kèm — xem
 * useAuthoringModels), nên đổi ở đây rồi quay lại tab nào cũng thấy đúng model
 * vừa chọn, và lượt chạy AI tiếp theo dùng đúng nó.
 */
export function AuthoringModelPicker({
  models, onChange, options, stepDefaults = {}, disabled, codeStats, codeStatsError,
}: AuthoringModelPickerProps) {
  if (options.length === 0) return null;

  // Bước chưa chọn hiện mặc định của bước (Cài đặt), trước khi có thì model đầu danh mục.
  const shown = options;
  const defaultFor = (step: string) => stepDefaults[step] || options[0].id;

  const codeModel = models.code || defaultFor("code");
  const stats = codeStatsError ? { text: codeStatsError, warn: true } : codeStatsLine(codeStats?.[codeModel]);

  return (
    <div className={`${glass.card} ${styles.card}`} data-testid="authoring-model-picker">
      <div className={styles.text}>
        <div className={glass.cardTitle}>Model AI</div>
        <p className={styles.hint}>
          Chọn model AI cho từng bước.
        </p>
      </div>
      <div className={styles.selects}>
        {STEP_ORDER.map(({ key, label }) => {
          const stepDefault = defaultFor(key);
          const value = models[key] || stepDefault || shown[0].id;
          const localCode = key === "code" && shown.find((o) => o.id === value)?.local === true;
          return (
            <div key={key} className={styles.field}>
              <span className={styles.fieldLabel}>{label}</span>
              <Dropdown
                aria-label={`Model AI cho bước ${label}`}
                value={value}
                onChange={(v) => onChange({ ...models, [key]: v })}
                disabled={disabled}
                data-testid={`authoring-model-${key}`}
                options={shown.map((option) => ({
                  value: option.id,
                  label: option.label,
                  badge: option.id === stepDefault ? "Mặc định" : undefined,
                }))}
              />
              {localCode && (
                <span className={styles.statsWarn} data-testid="authoring-model-code-local-warning">
                  {LOCAL_CODE_MODEL_WARNING}
                </span>
              )}
              {key === "code" && (
                <span className={stats.warn ? styles.statsWarn : styles.stats} data-testid="authoring-model-code-stats">
                  {stats.text}
                </span>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
