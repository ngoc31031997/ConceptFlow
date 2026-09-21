import { useEffect, useState } from "react";
import { AppShell } from "../components/AppShell";
import { Card, Button, FormField, Select, TextArea, CtaRow } from "../components/ui";
import {
  getPromptTemplate,
  updatePromptTemplate,
  resetPromptTemplate,
  type PromptTemplate,
} from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./PromptSettingsPage.module.css";

const ROLES: { value: PromptTemplate["role"]; label: string }[] = [
  { value: "story_architect", label: "1. Story Architect — dựng dàn ý" },
  { value: "visual_director", label: "2. Visual Director — dựng storyboard" },
  { value: "manim_engineer", label: "3. Manim Engineer — viết code" },
  { value: "script_reviewer", label: "4. Script Reviewer — duyệt kết quả" },
  // feature/remotion-engine — không thuộc chuỗi 4 bước Manim ở trên (Remotion
  // chưa có pipeline nhiều bước), nhưng vẫn đánh số tiếp theo cho nhất quán
  // với các lựa chọn khác trong danh sách.
  { value: "remotion_visual_director", label: "5. Remotion Visual Director — storyboard cho engine Remotion" },
  { value: "remotion_engineer", label: "6. Remotion Engineer — viết code Remotion" },
];

/** Dữ liệu mẫu chỉ để xem trước định dạng — không gửi lên server. */
const PREVIEW_SAMPLE: Record<string, string> = {
  topic: "Vì sao bầu trời có màu xanh",
  channel_identity: "(khối bản sắc kênh — CHANNEL_IDENTITY trong scriptPrompts.ts — sẽ hiện ở đây)",
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
 *
 * Dùng components/ui (Card/Button/FormField/Select/TextArea/CtaRow) thay vì
 * tự viết class/style — đây là màn tham chiếu cho DESIGN_SYSTEM.md.
 */
export function PromptSettingsPage() {
  const [role, setRole] = useState<PromptTemplate["role"]>("story_architect");
  const [language, setLanguage] = useState<"vi" | "en">("vi");
  const [text, setText] = useState("");
  const [version, setVersion] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);
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

  /**
   * Khôi phục prompt mặc định đang ship trong binary.
   *
   * Hỏi xác nhận vì thao tác này xoá hẳn bản người vận hành đã sửa, và phía
   * server không giữ lịch sử prompt (khác video_formats — một prompt cũ không
   * cần tái lập lại được với các bản render trước).
   */
  async function handleReset() {
    const ok = window.confirm(
      "Khôi phục prompt mặc định? Nội dung bạn đã sửa cho vai trò/ngôn ngữ này sẽ mất và không khôi phục lại được.",
    );
    if (!ok) return;

    setResetting(true);
    setStatus(null);
    try {
      const restored = await resetPromptTemplate(role, language);
      setText(restored.template_text);
      setVersion(restored.version);
      setStatus(`Đã khôi phục prompt mặc định — phiên bản ${restored.version}.`);
    } catch {
      setStatus("Khôi phục thất bại, thử lại.");
    } finally {
      setResetting(false);
    }
  }

  return (
    <div data-testid="prompt-settings-page">
      <AppShell
        title="Cài đặt prompt soạn kịch bản"
        subtitle="Sửa nội dung prompt của từng bước trong quy trình 4 vai trò (CR-025) — không cần build lại giao diện."
        wide
      >
        <div className={styles.layout}>
          <div className={styles.controls}>
            <Card title="Chọn prompt">
              <FormField label="Vai trò">
                <Select
                  data-testid="prompt-role-select"
                  value={role}
                  onChange={(e) => setRole(e.target.value as PromptTemplate["role"])}
                >
                  {ROLES.map((r) => (
                    <option key={r.value} value={r.value}>
                      {r.label}
                    </option>
                  ))}
                </Select>
              </FormField>

              <FormField label="Ngôn ngữ nội dung video" className={glass.mtSm}>
                <Select
                  data-testid="prompt-language-select"
                  value={language}
                  onChange={(e) => setLanguage(e.target.value as "vi" | "en")}
                >
                  <option value="vi">Tiếng Việt</option>
                  <option value="en">Tiếng Anh</option>
                </Select>
              </FormField>

              {version !== null && (
                <p className={`${styles.versionRow} ${glass.mtSm}`}>Phiên bản hiện tại: {version}</p>
              )}
              {status && (
                <p className={`${glass.cardHint} ${glass.mtXs}`} data-testid="prompt-settings-status">
                  {status}
                </p>
              )}
            </Card>

            <CtaRow>
              <Button
                variant="ghost"
                onClick={() => setShowPreview((v) => !v)}
                data-testid="prompt-preview-toggle"
              >
                {showPreview ? "Ẩn xem trước" : "Xem trước"}
              </Button>
              <Button
                variant="ghost"
                onClick={handleReset}
                disabled={resetting || saving || loading}
                data-testid="prompt-reset-button"
              >
                {resetting ? "Đang khôi phục..." : "Khôi phục mặc định"}
              </Button>
              <Button
                onClick={handleSave}
                disabled={saving || resetting || loading}
                data-testid="prompt-save-button"
              >
                {saving ? "Đang lưu..." : "Lưu"}
              </Button>
            </CtaRow>
          </div>

          <div>
            <Card title="Nội dung prompt">
              <TextArea
                data-testid="prompt-template-textarea"
                value={text}
                onChange={(e) => setText(e.target.value)}
                disabled={loading}
                rows={24}
              />
            </Card>

            {showPreview && (
              <div className={styles.previewBlock}>
                <Card title="Xem trước" hint="Dữ liệu mẫu, không gửi server">
                  <TextArea
                    data-testid="prompt-preview-textarea"
                    value={renderPreview(text)}
                    readOnly
                    rows={20}
                  />
                </Card>
              </div>
            )}
          </div>
        </div>
      </AppShell>
    </div>
  );
}
