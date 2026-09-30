import type { AuthoringModelOption, AuthoringStepModels, ModelUsageStats } from "../api/client";
import { Dropdown } from "./ui";
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
  /** Model cụ thể mà giá trị rỗng ("") được máy chủ quy về. */
  defaultModel?: string;
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
  models, onChange, options, defaultModel = "", disabled, codeStats, codeStatsError,
}: AuthoringModelPickerProps) {
  if (options.length === 0) return null;

  // "" nghĩa là "mặc định máy chủ" — luôn hiện tên model thật thay vì nhãn mơ hồ.
  const shown = options.some((o) => o.id === defaultModel) || !defaultModel
    ? options
    : [...options, { id: defaultModel, label: defaultModel }];

  const codeModel = models.code || defaultModel;
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
          const value = models[key] || defaultModel || shown[0].id;
          // Bước Code chỉ liệt kê model viết được code. Lựa chọn
          // cũ không hợp lệ vẫn hiện (kèm nhãn) để Creator thấy mà đổi.
          const listed = key === "code"
            ? shown.filter((o) => o.code_ok !== false || o.id === value)
            : shown;
          return (
            <div key={key} className={styles.field}>
              <span className={styles.fieldLabel}>{label}</span>
              <Dropdown
                aria-label={`Model AI cho bước ${label}`}
                value={value}
                onChange={(v) => onChange({ ...models, [key]: v })}
                disabled={disabled}
                data-testid={`authoring-model-${key}`}
                options={listed.map((option) => ({
                  value: option.id,
                  label: option.label,
                  badge: key === "code" && option.code_ok === false
                    ? "Không dùng cho Code"
                    : option.id === defaultModel ? "Mặc định" : undefined,
                }))}
              />
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
