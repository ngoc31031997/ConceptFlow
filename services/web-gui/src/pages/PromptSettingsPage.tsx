import { useCallback, useEffect, useState } from "react";
import { AppShell } from "../components/AppShell";
import { Card, Button, FormField, Select, TextArea, TextInput, CtaRow } from "../components/ui";
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
  { value: "story_architect", label: "1. Story Architect — dựng dàn ý" },
  { value: "visual_director", label: "2. Visual Director — dựng storyboard" },
  { value: "manim_engineer", label: "3. Manim Engineer — viết code" },
  // Bước 3 rẽ theo engine render: dự án chọn Remotion dùng prompt này thay
  // cho Manim Engineer. Hai bước đầu dùng chung cho mọi engine.
  { value: "remotion_engineer", label: "3. Remotion Engineer — viết code Remotion" },
];

const NEW_ROW = "new";

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
 * CR-031 — thư viện prompt. Mỗi vai trò trong pipeline soạn kịch bản có một
 * DANH SÁCH prompt; tại một thời điểm chỉ một dòng được bật và đó là dòng
 * pipeline chạy.
 *
 * Dòng "Hệ thống" đi kèm bản build: chỉ xem và copy, không sửa/xoá. Copy ra
 * thì được một dòng "Của bạn" — sửa, xoá, bật tuỳ ý. Không có version, không
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
      setStatus("Không tải được prompt — kiểm tra Orchestrator.");
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
      setStatus("Thao tác thất bại, thử lại.");
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
    }, "Đã copy thành một prompt của bạn.");

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
   * Xoá là thao tác phá huỷ duy nhất ở màn này, nên hỏi xác nhận. Xoá dòng
   * đang chạy thì vai trò tự quay về prompt hệ thống.
   */
  const handleDelete = () => {
    if (!selected) return;
    const okToDelete = window.confirm(
      selected.is_active
        ? "Xoá prompt này? Nó đang chạy, nên vai trò sẽ quay về prompt mặc định của hệ thống. Không khôi phục lại được."
        : "Xoá prompt này? Không khôi phục lại được.",
    );
    if (!okToDelete) return;
    return run(async () => {
      await deletePrompt(selected.id);
      setSelectedId(null);
    }, "Đã xoá.");
  };

  return (
    <div data-testid="prompt-settings-page">
      <AppShell
        title="Cài đặt prompt soạn kịch bản"
        subtitle="Mỗi vai trò có một danh sách prompt, chỉ một prompt được bật. Prompt hệ thống chỉ xem và copy được; copy ra để có bản của bạn."
        wide
      >
        <div className={styles.layout}>
          <div className={styles.controls}>
            <Card title="Chọn vai trò">
              <FormField label="Vai trò">
                <Select
                  data-testid="prompt-role-select"
                  value={role}
                  onChange={(e) => chooseRole(e.target.value as PromptRole)}
                >
                  {ROLES.map((r) => (
                    <option key={r.value} value={r.value}>
                      {r.label}
                    </option>
                  ))}
                </Select>
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
                <TextArea
                  data-testid="prompt-template-textarea"
                  value={draftText}
                  onChange={(e) => setDraftText(e.target.value)}
                  readOnly={readOnly}
                  disabled={loading || (!creating && selected === null)}
                  rows={22}
                />
              </FormField>
              <CtaRow>
                <Button
                  variant="ghost"
                  onClick={() => setShowPreview((v) => !v)}
                  data-testid="prompt-preview-toggle"
                >
                  {showPreview ? "Ẩn xem trước" : "Xem trước"}
                </Button>
                {selected !== null && (
                  <Button
                    variant="ghost"
                    onClick={() => handleCopy(selected.id)}
                    disabled={busy || loading}
                    data-testid="prompt-copy-button"
                  >
                    Copy
                  </Button>
                )}
                {selected !== null && !selected.is_active && (
                  <Button
                    variant="ghost"
                    onClick={() => handleActivate(selected.id)}
                    disabled={busy || loading}
                    data-testid="prompt-activate-button"
                  >
                    Bật prompt này
                  </Button>
                )}
                {!readOnly && selected !== null && (
                  <Button
                    variant="ghost"
                    onClick={handleDelete}
                    disabled={busy || loading}
                    data-testid="prompt-delete-button"
                  >
                    Xoá
                  </Button>
                )}
                {!readOnly && (creating || selected !== null) && (
                  <Button
                    onClick={handleSave}
                    disabled={busy || loading || draftName.trim() === "" || draftText.trim() === ""}
                    data-testid="prompt-save-button"
                  >
                    {busy ? "Đang lưu..." : "Lưu"}
                  </Button>
                )}
              </CtaRow>
            </Card>

            {showPreview && (
              <div className={styles.previewBlock}>
                <Card title="Xem trước" hint="Dữ liệu mẫu, không gửi server">
                  <TextArea
                    data-testid="prompt-preview-textarea"
                    value={renderPreview(draftText)}
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
