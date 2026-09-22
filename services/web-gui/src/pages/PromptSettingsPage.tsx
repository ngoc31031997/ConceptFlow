import { useCallback, useEffect, useState } from "react";
import { AppShell } from "../components/AppShell";
import { Card, Button, FormField, Select, TextArea, CtaRow } from "../components/ui";
import {
  listPromptTemplates,
  listPromptOverrides,
  savePromptOverride,
  setPromptOverrideActive,
  deletePromptOverride,
  type PromptTemplate,
  type PromptOverride,
} from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./PromptSettingsPage.module.css";

const ROLES: { value: PromptTemplate["role"]; label: string }[] = [
  { value: "story_architect", label: "1. Story Architect — dựng dàn ý" },
  { value: "visual_director", label: "2. Visual Director — dựng storyboard" },
  { value: "manim_engineer", label: "3. Manim Engineer — viết code" },
  // feature/remotion-engine — không thuộc chuỗi 3 bước Manim ở trên (Remotion
  // chưa có pipeline nhiều bước), nhưng vẫn đánh số tiếp theo cho nhất quán
  // với các lựa chọn khác trong danh sách.
  { value: "remotion_visual_director", label: "4. Remotion Visual Director — storyboard cho engine Remotion" },
  { value: "remotion_engineer", label: "5. Remotion Engineer — viết code Remotion" },
];

/** Dữ liệu mẫu chỉ để xem trước định dạng — không gửi lên server. */
const PREVIEW_SAMPLE: Record<string, string> = {
  topic: "Vì sao bầu trời có màu xanh",
  channel_identity: "(khối bản sắc kênh — CHANNEL_IDENTITY trong scriptPrompts.ts — sẽ hiện ở đây)",
  format_beats: "(danh sách beat của format đã chọn sẽ hiện ở đây)",
  narration_language_rule: "Toàn bộ lời thoại phải viết bằng TIẾNG VIỆT.",
  previous_output: "(nội dung bước trước — dàn ý/storyboard/code — sẽ hiện ở đây)",
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

  const [templates, setTemplates] = useState<PromptTemplate[]>([]);
  const [overrides, setOverrides] = useState<PromptOverride[]>([]);
  const [draft, setDraft] = useState("");
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [showPreview, setShowPreview] = useState(false);
  const [status, setStatus] = useState<string | null>(null);

  /**
   * Cả hai tầng tải cùng một lượt rồi ghép ở client. Hai danh sách đúng 12
   * hàng mỗi bên, nên gọi thêm một endpoint chuyên dụng chỉ để tránh một
   * phép ghép là không đáng.
   */
  const reload = useCallback(async () => {
    setLoading(true);
    try {
      const [t, o] = await Promise.all([listPromptTemplates(), listPromptOverrides()]);
      setTemplates(t);
      setOverrides(o);
    } catch {
      setStatus("Không tải được prompt — kiểm tra Orchestrator.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const seed = templates.find((t) => t.role === role && t.language === language) ?? null;
  const override = overrides.find((o) => o.role === role && o.language === language) ?? null;

  // Đổi vai trò/ngôn ngữ thì ô soạn thảo theo bản tuỳ chỉnh của đúng ô đó.
  useEffect(() => {
    setDraft(override?.template_text ?? "");
    setStatus(null);
  }, [role, language, override?.template_text]);

  /**
   * FR84.7 — bản gốc đã đi tiếp kể từ lúc bản tuỳ chỉnh này được viết.
   * Chỉ báo, không tự làm gì: tự trộn vào nội dung Creator đã viết là đúng
   * kiểu bất ngờ mà cả CR này sinh ra để dẹp. `based_on_version === 0` nghĩa
   * là không rõ (hàng do di trú tạo ra), nên không kết luận gì.
   */
  const seedMovedOn =
    override !== null &&
    override.is_active &&
    override.based_on_version > 0 &&
    seed !== null &&
    seed.version > override.based_on_version;

  const inUse = override?.is_active ? "override" : "seed";
  const effectiveText = inUse === "override" ? (override?.template_text ?? "") : (seed?.template_text ?? "");

  async function run(action: () => Promise<void>, ok: string) {
    setBusy(true);
    setStatus(null);
    try {
      await action();
      await reload();
      setStatus(ok);
    } catch {
      setStatus("Thao tác thất bại, thử lại.");
    } finally {
      setBusy(false);
    }
  }

  const handleSave = () =>
    run(async () => {
      await savePromptOverride(role, language, draft);
    }, "Đã lưu bản tuỳ chỉnh.");

  const handleStartFromSeed = () => setDraft(seed?.template_text ?? "");

  const handleToggle = () =>
    run(async () => {
      await setPromptOverrideActive(role, language, !(override?.is_active ?? false));
    }, override?.is_active ? "Đã tắt — đang chạy bản gốc." : "Đã bật bản tuỳ chỉnh.");

  /**
   * Xoá là thao tác phá huỷ duy nhất còn lại ở màn này, nên vẫn hỏi xác nhận.
   * Muốn tạm quay về bản gốc thì dùng công tắc bật/tắt — không mất gì.
   */
  const handleDelete = () => {
    const okToDelete = window.confirm(
      "Xoá hẳn bản tuỳ chỉnh này? Không khôi phục lại được. Nếu chỉ muốn tạm dùng bản gốc, hãy TẮT nó thay vì xoá.",
    );
    if (!okToDelete) return;
    return run(async () => {
      await deletePromptOverride(role, language);
      setDraft("");
    }, "Đã xoá bản tuỳ chỉnh — đang chạy bản gốc.");
  };

  return (
    <div data-testid="prompt-settings-page">
      <AppShell
        title="Cài đặt prompt soạn kịch bản"
        subtitle="Prompt gốc của hệ thống là chỉ-đọc và tự cập nhật theo mỗi bản build. Bản tuỳ chỉnh của bạn nằm riêng, bật/tắt được bất cứ lúc nào."
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

              <p className={`${styles.versionRow} ${glass.mtSm}`} data-testid="prompt-in-use">
                {inUse === "override"
                  ? "Đang chạy: BẢN TUỲ CHỈNH của bạn"
                  : "Đang chạy: bản gốc của hệ thống"}
                {seed !== null && ` (bản gốc phiên bản ${seed.version})`}
              </p>

              {seedMovedOn && (
                <p className={`${glass.cardHint} ${glass.mtXs}`} data-testid="prompt-seed-moved-on">
                  ⚠ Bản gốc đã cập nhật lên phiên bản {seed?.version} kể từ khi bạn viết bản tuỳ chỉnh
                  này (dựa trên phiên bản {override?.based_on_version}). Hệ thống không tự trộn —
                  bạn tự đối chiếu rồi quyết định.
                </p>
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
                onClick={handleStartFromSeed}
                disabled={busy || loading || seed === null}
                data-testid="prompt-copy-seed-button"
              >
                Chép từ bản gốc
              </Button>
              <Button
                onClick={handleSave}
                disabled={busy || loading || draft.trim().length === 0}
                data-testid="prompt-save-button"
              >
                {busy ? "Đang lưu..." : "Lưu bản tuỳ chỉnh"}
              </Button>
            </CtaRow>

            {override !== null && (
              <CtaRow>
                <Button
                  variant="ghost"
                  onClick={handleToggle}
                  disabled={busy || loading}
                  data-testid="prompt-toggle-button"
                >
                  {override.is_active ? "Tắt (quay về bản gốc)" : "Bật bản tuỳ chỉnh"}
                </Button>
                <Button
                  variant="ghost"
                  onClick={handleDelete}
                  disabled={busy || loading}
                  data-testid="prompt-delete-button"
                >
                  Xoá bản tuỳ chỉnh
                </Button>
              </CtaRow>
            )}
          </div>

          <div>
            <Card
              title="Bản tuỳ chỉnh của bạn"
              hint={
                override === null
                  ? "Chưa có. Bấm 'Chép từ bản gốc' để lấy điểm khởi đầu."
                  : override.is_active
                    ? "Đang được dùng."
                    : "Đang tắt — hệ thống chạy bản gốc. Nội dung vẫn được giữ."
              }
            >
              <TextArea
                data-testid="prompt-template-textarea"
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                disabled={loading}
                rows={20}
              />
            </Card>

            <div className={styles.previewBlock}>
              <Card
                title="Prompt gốc của hệ thống (chỉ đọc)"
                hint="Tự cập nhật theo mỗi bản build. Không sửa và không xoá được — bản sửa của bạn nằm ở khung trên."
              >
                <TextArea
                  data-testid="prompt-seed-textarea"
                  value={seed?.template_text ?? ""}
                  readOnly
                  rows={14}
                />
              </Card>
            </div>

            {showPreview && (
              <div className={styles.previewBlock}>
                <Card title="Xem trước nội dung ĐANG CHẠY" hint="Dữ liệu mẫu, không gửi server">
                  <TextArea
                    data-testid="prompt-preview-textarea"
                    value={renderPreview(effectiveText)}
                    readOnly
                    rows={18}
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
