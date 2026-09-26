import type { AuthoringModelOption, AuthoringStepModels } from "../api/client";
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
export function AuthoringModelPicker({ models, onChange, options, defaultModel = "", disabled }: AuthoringModelPickerProps) {
  if (options.length === 0) return null;

  // "" nghĩa là "mặc định máy chủ" — luôn hiện tên model thật thay vì nhãn mơ hồ.
  const shown = options.some((o) => o.id === defaultModel) || !defaultModel
    ? options
    : [...options, { id: defaultModel, label: defaultModel }];

  return (
    <div className={`${glass.card} ${styles.card}`} data-testid="authoring-model-picker">
      <div className={styles.text}>
        <div className={glass.cardTitle}>Model AI</div>
        <p className={styles.hint}>
          Chọn model AI cho từng bước.
        </p>
      </div>
      <div className={styles.selects}>
        {STEP_ORDER.map(({ key, label }) => (
          <label key={key} className={styles.field}>
            <span className={styles.fieldLabel}>{label}</span>
            <select
              className={styles.select}
              value={models[key] || defaultModel || shown[0].id}
              onChange={(event) => onChange({ ...models, [key]: event.target.value })}
              disabled={disabled}
              data-testid={`authoring-model-${key}`}
            >
              {shown.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
        ))}
      </div>
    </div>
  );
}
