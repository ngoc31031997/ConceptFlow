import { useCallback, useEffect, useState } from "react";
import { AppShell } from "../components/AppShell";
import { Card, Button, CopyBox, Dropdown, FormField, TextArea, TextInput, CtaRow } from "../components/ui";
import {
  listPrompts,
  createPrompt,
  copyPrompt,
  updatePrompt,
  activatePrompt,
  deletePrompt,
  type Prompt,
  type PromptRole,
} from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./PromptSettingsPage.module.css";

const ROLES: { value: PromptRole; label: string }[] = [
  { value: "creative_brief", label: "0. Creative brief — viết brief từ đề tài (bước Ý tưởng)" },
  { value: "research_questions", label: "0b. Câu hỏi nghiên cứu — AI đặt câu hỏi từ brief (bước Nghiên cứu)" },
  { value: "research_prompt", label: "0c. Prompt nghiên cứu — chép sang công cụ nghiên cứu bên ngoài (bước Nghiên cứu)" },
  { value: "story_approaches", label: "0d. Góc tiếp cận — AI gợi ý cách kể từ brief và dữ kiện (bước Câu chuyện)" },
  { value: "story_frame", label: "0e. Khung câu chuyện — AI viết câu hỏi mở đầu, lời hứa, chuỗi nhân quả, payoff (bước Câu chuyện)" },
  { value: "story_architect", label: "1. Story Architect — dựng dàn ý" },
  { value: "visual_director", label: "2. Visual Director — dựng storyboard" },
  { value: "remotion_engineer", label: "3. Remotion Engineer — viết code Remotion" },
  // Ba prompt của luồng "Chạy bằng AI" (không dùng cho nút Copy).
  { value: "visual_director_ai", label: "AI 2. Visual Director — storyboard dạng JSON" },
  { value: "remotion_engineer_ai", label: "AI 3. Remotion Engineer — chỉ viết các hàm shot" },
  // Các prompt "chép sang AI ngoài", do server dựng.
  { value: "remotion_adjust", label: "Phụ trợ — chuẩn hoá code Remotion có sẵn" },
  { value: "thumbnail_design", label: "Phụ trợ — prompt sinh ảnh thumbnail" },
  // Luật AI vẽ hình phải theo, và là luật trang Thư viện hình hiển thị.
  { value: "illustration_style", label: "Thư viện hình — luật style hình minh hoạ (AI vẽ)" },
];

const NEW_ROW = "new";

/** Dữ liệu mẫu chỉ để xem trước định dạng — không gửi lên server. */
const PREVIEW_SAMPLE: Record<string, string> = {
  topic: "Vì sao bầu trời có màu xanh",
  channel_identity: "(khối bản sắc kênh — do server dựng theo ngôn ngữ — sẽ hiện ở đây)",
  video_archetypes: "(danh sách kiểu video và playbook từng kiểu — lấy từ mục Kiểu video — sẽ hiện ở đây)",
  format_beats: "(danh sách beat của format đã chọn sẽ hiện ở đây)",
  narration_language_rule: "Toàn bộ lời thoại phải viết bằng TIẾNG VIỆT.",
  previous_output: "(nội dung bước trước — dàn ý/storyboard/code — sẽ hiện ở đây)",
  script: "(code có sẵn Creator dán vào sẽ hiện ở đây)",
  subtitle_zone: "(vùng phụ đề theo cấu hình của video sẽ hiện ở đây)",
  frame: "1920×1080 (ngang)",
  frame_width: "1920",
  frame_height: "1080",
  safe_area: "(96, 96) đến (1824, 984)",
  frame_rules: "(luật khung dọc — chỉ có khi video là short dọc — sẽ hiện ở đây)",
  narrate_example: "Nội dung lời thoại cho đoạn này",
  thumbnail_audience: "(khán giả theo ngôn ngữ video sẽ hiện ở đây)",
  creative_brief: "(creative brief đã duyệt ở bước Ý tưởng, kèm thời lượng mục tiêu, sẽ hiện ở đây)",
  video_kind: "Video dài 16:9",
  target_duration: "4–5 phút",
  brief_language: "tiếng Việt",
  current_brief: "(brief hiện có, khi làm lại, sẽ hiện ở đây)",
  kept_fields: "(các ô Creator đã tự viết, AI giữ nguyên, sẽ hiện ở đây)",
  redo_note: "(ghi chú của Creator khi làm lại sẽ hiện ở đây)",
  brief: "(creative brief đã duyệt, kèm thời lượng mục tiêu, sẽ hiện ở đây)",
  kept_questions: "(các câu hỏi Creator đã tự viết, AI giữ nguyên, sẽ hiện ở đây)",
  central_question: "(câu hỏi trung tâm của brief sẽ hiện ở đây)",
  audience: "(khán giả của brief sẽ hiện ở đây)",
  questions_heading: "CÂU HỎI CẦN TRA",
  questions: "(danh sách câu hỏi Q1, Q2... sẽ hiện ở đây)",
  extra: "(dữ kiện cần nguồn khác, chỉ có ở prompt bổ sung, sẽ hiện ở đây)",
  answer_note: "(lời nhắc đánh mã tiếp, chỉ có ở prompt bổ sung, sẽ hiện ở đây)",
  facts: "(bảng dữ kiện: mã, loại, nội dung, sẽ hiện ở đây)",
  facts_rule: "(luật về dữ kiện: dùng bảng có sẵn hoặc tự liệt kê, sẽ hiện ở đây)",
  archetypes: "(các kiểu video có trong bảng Kiểu video sẽ hiện ở đây)",
  approach: "(góc tiếp cận Creator đã chọn, kèm cách kể của kiểu đó, sẽ hiện ở đây)",
  current_frame: "(các ô của khung câu chuyện hiện có sẽ hiện ở đây)",
  fields_to_write: "(các ô AI cần viết lần này sẽ hiện ở đây)",
};

function renderPreview(templateText: string): string {
  let out = templateText;
  for (const [key, value] of Object.entries(PREVIEW_SAMPLE)) {
    out = out.split(`{{${key}}}`).join(value);
  }
  return out;
}

/**
 * Thư viện prompt. Mỗi vai trò trong pipeline soạn kịch bản có một
 * DANH SÁCH prompt; tại một thời điểm chỉ một dòng được bật và đó là dòng
 * pipeline chạy.
 *
 * Dòng "Hệ thống" đi kèm bản build: chỉ xem và copy, không sửa/xóa. Copy ra
 * thì được một dòng "Của bạn" — sửa, xóa, bật tuỳ ý. Không có version, không
 * có ngôn ngữ riêng: ngôn ngữ lời thoại do `{{narration_language_rule}}` quyết
 * định lúc render.
 *
 * Cố tình để ngoài luồng wizard của Creator (mục "Cài đặt" riêng) — đây là
 * công cụ cho người vận hành kênh, không phải bước đi qua mỗi lần tạo video.
 */
export function PromptSettingsPage() {
  const [role, setRole] = useState<PromptRole>("story_architect");
  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [draftName, setDraftName] = useState("");
  const [draftText, setDraftText] = useState("");
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [showPreview, setShowPreview] = useState(false);
  const [status, setStatus] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      setPrompts(await listPrompts());
    } catch {
      setStatus("Không tải được prompt. Vui lòng thử lại.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const rows = prompts.filter((p) => p.role === role);
  const selected = rows.find((p) => p.id === selectedId) ?? null;
  const creating = selectedId === NEW_ROW;

  // Đổi vai trò, hoặc danh sách vừa tải xong mà chưa chọn dòng nào: nhảy tới
  // dòng đang chạy của vai trò đó.
  useEffect(() => {
    if (creating) return;
    if (selected === null) {
      setSelectedId(rows.find((p) => p.is_active)?.id ?? rows[0]?.id ?? null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [role, prompts, selectedId]);

  // Ô soạn thảo luôn theo dòng đang chọn.
  useEffect(() => {
    if (creating) return;
    setDraftName(selected?.name ?? "");
    setDraftText(selected?.template_text ?? "");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected?.id, selected?.name, selected?.template_text, creating]);

  const readOnly = selected?.is_system === true;
  const active = rows.find((p) => p.is_active) ?? null;

  function chooseRole(next: PromptRole) {
    setRole(next);
    setSelectedId(null);
    setStatus(null);
  }

  /**
   * `action` có thể trả về id dòng cần chọn sau khi danh sách tải lại. Chọn
   * TRƯỚC khi tải lại thì dòng mới chưa có trong danh sách, và effect tự chọn
   * sẽ kéo lựa chọn về dòng cũ.
   */
  async function run(action: () => Promise<string | void>, ok: string) {
    setBusy(true);
    setStatus(null);
    try {
      const nextId = await action();
      await reload();
      if (nextId) setSelectedId(nextId);
      setStatus(ok);
    } catch {
      setStatus("Thao tác không thành công. Vui lòng thử lại.");
    } finally {
      setBusy(false);
    }
  }

  const startNew = () => {
    setSelectedId(NEW_ROW);
    setDraftName("");
    setDraftText("");
    setStatus(null);
  };

  const handleCopy = (id: string) =>
    run(async () => {
      return (await copyPrompt(id)).id;
    }, "Đã sao chép thành một prompt của bạn.");

  const handleSave = () =>
    run(async () => {
      if (creating) {
        return (await createPrompt(role, draftName, draftText)).id;
      } else if (selected) {
        await updatePrompt(selected.id, draftName, draftText);
      }
    }, creating ? "Đã tạo prompt." : "Đã lưu.");

  const handleActivate = (id: string) =>
    run(async () => {
      await activatePrompt(id);
    }, "Đã bật — đây là prompt đang chạy của vai trò này.");

  /**
   * Xóa là thao tác phá hủy duy nhất ở màn này, nên hỏi xác nhận. Xóa dòng
   * đang chạy thì vai trò tự quay về prompt hệ thống.
   */
  const handleDelete = () => {
    if (!selected) return;
    const okToDelete = window.confirm(
      selected.is_active
        ? "Xóa prompt này? Nó đang chạy, nên vai trò sẽ quay về prompt mặc định của hệ thống. Không khôi phục lại được."
        : "Xóa prompt này? Không khôi phục lại được.",
    );
    if (!okToDelete) return;
    return run(async () => {
      await deletePrompt(selected.id);
      setSelectedId(null);
    }, "Đã xóa.");
  };

  return (
    <div data-testid="prompt-settings-page">
      <AppShell
        title="Cài đặt prompt soạn kịch bản"
        subtitle="Mỗi vai trò có nhiều prompt, chỉ một prompt được bật. Prompt hệ thống chỉ có thể xem và sao chép để tạo bản riêng."
        wide
      >
        <div className={styles.layout}>
          <div className={styles.controls}>
            <Card title="Chọn vai trò">
              <FormField label="Vai trò">
                <Dropdown
                  data-testid="prompt-role-select"
                  aria-label="Vai trò"
                  value={role}
                  onChange={(v) => chooseRole(v as PromptRole)}
                  options={ROLES.map((r) => {
                    const [label, hint] = r.label.split(" — ");
                    return { value: r.value, label, hint };
                  })}
                />
              </FormField>
              <p className={`${styles.versionRow} ${glass.mtSm}`} data-testid="prompt-in-use">
                {active
                  ? `Đang chạy: ${active.name}${active.is_system ? " (hệ thống)" : ""}`
                  : "Chưa có prompt nào đang chạy"}
              </p>
              {status && (
                <p className={`${glass.cardHint} ${glass.mtXs}`} data-testid="prompt-settings-status">
                  {status}
                </p>
              )}
            </Card>

            <Card title="Danh sách prompt">
              <ul className={styles.list} data-testid="prompt-list">
                {rows.map((p) => (
                  <li key={p.id}>
                    <button
                      type="button"
                      className={`${styles.row} ${p.id === selectedId ? styles.rowSelected : ""}`}
                      onClick={() => {
                        setSelectedId(p.id);
                        setStatus(null);
                      }}
                      data-testid={`prompt-row-${p.id}`}
                    >
                      <span className={styles.rowName}>{p.name}</span>
                      <span className={styles.badges}>
                        <span className={styles.badge}>{p.is_system ? "Hệ thống" : "Của bạn"}</span>
                        {p.is_active && (
                          <span className={`${styles.badge} ${styles.badgeActive}`}>Đang dùng</span>
                        )}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
              <CtaRow>
                <Button variant="ghost" onClick={startNew} disabled={busy || loading} data-testid="prompt-new-button">
                  + Thêm prompt mới
                </Button>
              </CtaRow>
            </Card>
          </div>

          <div>
            <Card
              headerAction={
                (selected !== null || creating) ? (
                  <div className={styles.actions} role="toolbar" aria-label="Thao tác prompt">
                    <button
                      type="button"
                      className={styles.actionBtn}
                      onClick={() => setShowPreview((v) => !v)}
                      data-testid="prompt-preview-toggle"
                      aria-pressed={showPreview}
                    >
                      <span aria-hidden="true">👁</span> {showPreview ? "Ẩn xem trước" : "Xem trước"}
                    </button>
                    {selected !== null && !creating && !selected.is_active && (
                      <button
                        type="button"
                        className={`${styles.actionBtn} ${styles.actionPrimary}`}
                        onClick={() => handleActivate(selected.id)}
                        disabled={busy || loading}
                        data-testid="prompt-activate-button"
                        title="Bật prompt này cho vai trò"
                      >
                        <span aria-hidden="true">⏻</span> Bật
                      </button>
                    )}
                    {selected !== null && !creating && (
                    <button
                      type="button"
                      className={styles.actionBtn}
                      onClick={() => handleCopy(selected.id)}
                      disabled={busy || loading}
                      data-testid="prompt-copy-button"
                      title="Nhân bản thành prompt của bạn"
                    >
                      <span aria-hidden="true">⧉</span> Copy
                    </button>
                    )}
                    {!readOnly && selected !== null && !creating && (
                      <button
                        type="button"
                        className={`${styles.actionBtn} ${styles.actionDanger}`}
                        onClick={handleDelete}
                        disabled={busy || loading}
                        data-testid="prompt-delete-button"
                        title="Xóa prompt này"
                      >
                        <span aria-hidden="true">🗑</span> Xóa
                      </button>
                    )}
                    {!readOnly && (
                      <button
                        type="button"
                        className={`${styles.actionBtn} ${styles.actionSave}`}
                        onClick={handleSave}
                        disabled={busy || loading || draftName.trim() === "" || draftText.trim() === ""}
                        data-testid="prompt-save-button"
                      >
                        <span aria-hidden="true">💾</span> {busy ? "Đang lưu..." : "Lưu"}
                      </button>
                    )}
                  </div>
                ) : undefined
              }
              title={creating ? "Prompt mới" : (selected?.name ?? "Chưa chọn prompt")}
              hint={
                readOnly
                  ? "Prompt hệ thống — chỉ xem. Bấm 'Copy' để tạo bản của bạn rồi sửa."
                  : creating
                    ? "Viết nội dung rồi bấm Lưu. Prompt mới không tự bật."
                    : selected?.is_active
                      ? "Đang được dùng."
                      : "Đang tắt."
              }
            >
              <FormField label="Tên">
                <TextInput
                  data-testid="prompt-name-input"
                  value={draftName}
                  onChange={(e) => setDraftName(e.target.value)}
                  readOnly={readOnly}
                  disabled={loading || (!creating && selected === null)}
                />
              </FormField>
              <FormField label="Nội dung" className={glass.mtSm}>
                <CopyBox text={draftText} what="nội dung prompt" copyTestId="prompt-template-quick-copy">
                  <TextArea
                    data-testid="prompt-template-textarea"
                    value={draftText}
                    onChange={(e) => setDraftText(e.target.value)}
                    readOnly={readOnly}
                    disabled={loading || (!creating && selected === null)}
                    rows={22}
                  />
                </CopyBox>
              </FormField>
            </Card>

            {showPreview && (
              <div className={styles.previewBlock}>
                <Card title="Xem trước" hint="Dữ liệu mẫu">
                  <CopyBox text={renderPreview(draftText)} what="bản xem trước" copyTestId="prompt-preview-quick-copy">
                    <TextArea
                      data-testid="prompt-preview-textarea"
                      value={renderPreview(draftText)}
                      readOnly
                      rows={18}
                    />
                  </CopyBox>
                </Card>
              </div>
            )}
          </div>
        </div>
      </AppShell>
    </div>
  );
}
