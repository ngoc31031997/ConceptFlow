import { useEffect, useState } from "react";
import { AppShell } from "../components/AppShell";
import {
  getPromptTemplate,
  updatePromptTemplate,
  type PromptTemplate,
} from "../api/client";
import styles from "./WizardSteps.module.css";

const ROLES: { value: PromptTemplate["role"]; label: string }[] = [
  { value: "story_architect", label: "1. Story Architect — dựng dàn ý" },
  { value: "visual_director", label: "2. Visual Director — dựng storyboard" },
  { value: "manim_engineer", label: "3. Manim Engineer — viết code" },
  { value: "script_reviewer", label: "4. Script Reviewer — duyệt kết quả" },
];

/** Dữ liệu mẫu chỉ để xem trước định dạng — không gửi lên server. */
const PREVIEW_SAMPLE: Record<string, string> = {
  topic: "Vì sao bầu trời có màu xanh",
  format_beats: "(danh sách beat của format đã chọn sẽ hiện ở đây)",
  narration_language_rule: "Toàn bộ lời thoại phải viết bằng TIẾNG VIỆT.",
  previous_output: "(nội dung bước trước — dàn ý/storyboard/code — sẽ hiện ở đây)",
  lint_results: "(kết quả kiểm tra tĩnh sẽ hiện ở đây)",
};

function renderPreview(templateText: string): string {
  let out = templateText;
  for (const [key, value] of Object.entries(PREVIEW_SAMPLE)) {
    out = out.split(`{{${key}}}`).join(value);
  }
  return out;
}

/**
 * CR-025 — màn admin sửa nội dung prompt của 4 vai trò trong pipeline soạn
 * kịch bản, không cần build lại web-gui. Cố tình để ngoài luồng wizard của
 * Creator (mục "Cài đặt" riêng) — đây là công cụ cho người vận hành kênh,
 * không phải bước Creator đi qua mỗi lần tạo video.
 */
export function PromptSettingsPage() {
  const [role, setRole] = useState<PromptTemplate["role"]>("story_architect");
  const [language, setLanguage] = useState<"vi" | "en">("vi");
  const [text, setText] = useState("");
  const [version, setVersion] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [showPreview, setShowPreview] = useState(false);
  const [status, setStatus] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setStatus(null);
    getPromptTemplate(role, language)
      .then((template) => {
        if (cancelled) return;
        setText(template.template_text);
        setVersion(template.version);
      })
      .catch(() => {
        if (!cancelled) {
          setText("");
          setVersion(null);
          setStatus("Không tải được template — có thể vai trò/ngôn ngữ này chưa có sẵn.");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [role, language]);

  async function handleSave() {
    setSaving(true);
    setStatus(null);
    try {
      const saved = await updatePromptTemplate(role, language, text);
      setVersion(saved.version);
      setStatus(`Đã lưu — phiên bản ${saved.version}.`);
    } catch {
      setStatus("Lưu thất bại, thử lại.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div data-testid="prompt-settings-page">
      <AppShell
        title="Cài đặt prompt soạn kịch bản"
        subtitle="Sửa nội dung prompt của từng bước trong quy trình 4 vai trò (CR-025) — không cần build lại giao diện."
        wide
      >
        <div className={styles.scriptLayout}>
          <div className={styles.assistantColumn}>
            <label style={{ display: "block", marginBottom: 8 }}>
              Vai trò
              <select
                data-testid="prompt-role-select"
                value={role}
                onChange={(e) => setRole(e.target.value as PromptTemplate["role"])}
                style={{ display: "block", width: "100%", marginTop: 4 }}
              >
                {ROLES.map((r) => (
                  <option key={r.value} value={r.value}>
                    {r.label}
                  </option>
                ))}
              </select>
            </label>

            <label style={{ display: "block", marginBottom: 8 }}>
              Ngôn ngữ nội dung video
              <select
                data-testid="prompt-language-select"
                value={language}
                onChange={(e) => setLanguage(e.target.value as "vi" | "en")}
                style={{ display: "block", width: "100%", marginTop: 4 }}
              >
                <option value="vi">Tiếng Việt</option>
                <option value="en">Tiếng Anh</option>
              </select>
            </label>

            {version !== null && <p>Phiên bản hiện tại: {version}</p>}
            {status && <p data-testid="prompt-settings-status">{status}</p>}

            <button type="button" onClick={handleSave} disabled={saving || loading} data-testid="prompt-save-button">
              {saving ? "Đang lưu..." : "Lưu"}
            </button>{" "}
            <button type="button" onClick={() => setShowPreview((v) => !v)} data-testid="prompt-preview-toggle">
              {showPreview ? "Ẩn xem trước" : "Xem trước"}
            </button>
          </div>

          <div>
            <textarea
              data-testid="prompt-template-textarea"
              value={text}
              onChange={(e) => setText(e.target.value)}
              disabled={loading}
              rows={24}
              style={{ width: "100%", fontFamily: "monospace" }}
            />
            {showPreview && (
              <>
                <h3>Xem trước (dữ liệu mẫu, không gửi server)</h3>
                <textarea
                  data-testid="prompt-preview-textarea"
                  value={renderPreview(text)}
                  readOnly
                  rows={20}
                  style={{ width: "100%", fontFamily: "monospace" }}
                />
              </>
            )}
          </div>
        </div>
      </AppShell>
    </div>
  );
}
