import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
  deleteLlmCalls,
  getLlmCall,
  listLlmCalls,
  type LlmCall,
  type LlmCallDetail,
  type LlmCallsDeletion,
} from "../api/client";
import { ConfirmModal } from "./ConfirmModal";
import { Button, Card, CopyBox, Dialog, FormField, Select, TextInput } from "./ui";
import glass from "../styles/glass.module.css";
import styles from "./LlmCallsPanel.module.css";

/** Cuộc gọi cũ hơn mức này là thứ "Xoá các dòng cũ" dọn đi. */
const RETENTION_DAYS = 30;

type StatusFilter = "all" | "ok" | "failed";

interface Applied {
  project: string;
  role: string;
  status: StatusFilter;
}

type Pending = { kind: "selected" } | { kind: "older" } | null;

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("vi-VN", { hour12: false });
}

/** The provider label of the reply a Creator pasted from an AI outside the system. */
const EXTERNAL_PROVIDER = "external";

/** The model column: calls made through an outside AI have no model of the system's own. */
function modelLabel(call: { provider: string; model: string }): string {
  return call.provider === EXTERNAL_PROVIDER ? "AI ngoài" : call.model;
}

function formatMs(ms: number): string {
  if (ms <= 0) return "—";
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`;
}

function errorText(err: unknown): string {
  return err instanceof ApiError ? err.message : String(err);
}

/**
 * The raw table of llm-service's call log: what each model call answered, what it
 * cost, and (on "Xem") the system and user prompt that went out. Rows can be picked and deleted, or every row
 * older than 30 days; a deletion also removes those calls from the statistics
 * that are measured over the same rows.
 */
export function LlmCallsPanel() {
  const [draft, setDraft] = useState<Applied>({ project: "", role: "", status: "all" });
  const [applied, setApplied] = useState<Applied>(draft);
  const [rows, setRows] = useState<LlmCall[] | null>(null);
  const [nextBefore, setNextBefore] = useState<string | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [shown, setShown] = useState<LlmCall | null>(null);
  const [detail, setDetail] = useState<LlmCallDetail | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [pending, setPending] = useState<Pending>(null);

  const query = useCallback(
    (before?: string) => ({
      project_id: applied.project.trim() || undefined,
      role: applied.role.trim() || undefined,
      ok: applied.status === "all" ? undefined : applied.status === "ok",
      before,
    }),
    [applied],
  );

  const reload = useCallback(async () => {
    setError(null);
    try {
      const page = await listLlmCalls(query());
      setRows(page.calls);
      setNextBefore(page.next_before);
      setSelected(new Set());
    } catch (err) {
      setError(errorText(err));
      setRows([]);
    }
  }, [query]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const loadMore = async () => {
    if (!nextBefore) return;
    setLoadingMore(true);
    try {
      const page = await listLlmCalls(query(nextBefore));
      setRows((cur) => [...(cur ?? []), ...page.calls]);
      setNextBefore(page.next_before);
    } catch (err) {
      setError(errorText(err));
    } finally {
      setLoadingMore(false);
    }
  };

  const open = (r: LlmCall) => {
    setShown(r);
    setDetail(null);
    setDetailError(null);
    getLlmCall(r.call_id)
      .then(setDetail)
      .catch((err) => setDetailError(errorText(err)));
  };

  const all = rows ?? [];
  const allSelected = all.length > 0 && all.every((r) => selected.has(r.call_id));
  const toggleAll = () => setSelected(allSelected ? new Set() : new Set(all.map((r) => r.call_id)));
  const toggle = (id: string) =>
    setSelected((cur) => {
      const next = new Set(cur);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const confirmDeletion = async () => {
    const choice: LlmCallsDeletion =
      pending?.kind === "selected" ? { ids: [...selected] } : { older_than_days: RETENTION_DAYS };
    setNotice(null);
    setError(null);
    try {
      const { deleted } = await deleteLlmCalls(choice);
      setNotice(`Đã xoá ${deleted} dòng.`);
      await reload();
    } catch (err) {
      setError(errorText(err));
    }
  };

  return (
    <Card
      title="Cuộc gọi AI"
      hint="Prompt đã gửi, phản hồi thô và chi phí của từng lượt gọi model, ghi ở llm-service."
      data-testid="llm-calls-panel"
    >
      <form
        className={styles.filters}
        onSubmit={(e) => {
          e.preventDefault();
          setApplied(draft);
        }}
      >
        <FormField label="Dự án">
          <TextInput
            id="llm-filter-project"
            value={draft.project}
            onChange={(e) => setDraft({ ...draft, project: e.target.value })}
            placeholder="Mã dự án"
            data-testid="llm-filter-project"
          />
        </FormField>
        <FormField label="Vai AI">
          <TextInput
            id="llm-filter-role"
            value={draft.role}
            onChange={(e) => setDraft({ ...draft, role: e.target.value })}
            placeholder="vd. story_frame"
            data-testid="llm-filter-role"
          />
        </FormField>
        <FormField label="Kết quả">
          <Select
            id="llm-filter-status"
            value={draft.status}
            onChange={(e) => setDraft({ ...draft, status: e.target.value as StatusFilter })}
            data-testid="llm-filter-status"
          >
            <option value="all">Tất cả</option>
            <option value="ok">Thành công</option>
            <option value="failed">Lỗi</option>
          </Select>
        </FormField>
        <Button type="submit" variant="ghost" data-testid="llm-filter-apply">
          Lọc
        </Button>
      </form>

      {error && (
        <p role="alert" className={glass.helperText} data-testid="llm-calls-error">
          {error}
        </p>
      )}
      {notice && (
        <p role="status" className={styles.notice} data-testid="llm-calls-notice">
          {notice}
        </p>
      )}
      {rows === null && !error && <p className={styles.empty}>Đang tải...</p>}
      {rows !== null && rows.length === 0 && !error && <p className={styles.empty}>Chưa có cuộc gọi nào khớp.</p>}

      {all.length > 0 && (
        <div className={styles.tableWrap}>
          <table className={styles.table} data-testid="llm-calls-table">
            <thead>
              <tr>
                <th className={styles.check}>
                  <input
                    type="checkbox"
                    checked={allSelected}
                    onChange={toggleAll}
                    aria-label="Chọn tất cả các dòng đang hiện"
                    data-testid="llm-select-all"
                  />
                </th>
                <th>Thời điểm</th>
                <th>Dự án</th>
                <th>Vai AI</th>
                <th>Bước</th>
                <th>Model</th>
                <th>Kết quả</th>
                <th>Token vào / ra</th>
                <th>Prompt (ký tự)</th>
                <th>Thời gian</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {all.map((r) => (
                <tr key={r.call_id} data-testid={`llm-call-${r.call_id}`}>
                  <td className={styles.check}>
                    <input
                      type="checkbox"
                      checked={selected.has(r.call_id)}
                      onChange={() => toggle(r.call_id)}
                      aria-label={`Chọn cuộc gọi ${r.call_id.slice(0, 8)}`}
                      data-testid={`llm-select-${r.call_id}`}
                    />
                  </td>
                  <td>{formatTime(r.created_at)}</td>
                  <td className={styles.mono}>{r.project_id ? r.project_id.slice(0, 8) : "—"}</td>
                  <td>{r.role || "—"}</td>
                  <td>{[r.step, r.phase].filter(Boolean).join(" · ") || "—"}</td>
                  <td>{modelLabel(r)}</td>
                  <td className={r.ok ? styles.ok : styles.fail}>{r.ok ? "OK" : r.error_kind || "lỗi"}</td>
                  <td>
                    {r.usage_reported ? `${r.prompt_tokens} / ${r.completion_tokens}` : "không rõ"}
                  </td>
                  <td>{r.prompt_chars > 0 ? r.prompt_chars.toLocaleString("vi-VN") : "—"}</td>
                  <td>{formatMs(r.duration_ms)}</td>
                  <td>
                    <Button variant="view" onClick={() => open(r)} data-testid={`llm-view-${r.call_id}`}>
                      Xem
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {nextBefore && (
        <div className={styles.actions}>
          <Button variant="ghost" onClick={() => void loadMore()} disabled={loadingMore} data-testid="llm-load-more">
            {loadingMore ? "Đang tải..." : "Tải thêm"}
          </Button>
        </div>
      )}

      <div className={styles.actions}>
        <Button
          variant="danger"
          disabled={selected.size === 0}
          onClick={() => setPending({ kind: "selected" })}
          data-testid="llm-delete-selected"
        >
          Xoá {selected.size} dòng đã chọn
        </Button>
        <Button variant="danger" onClick={() => setPending({ kind: "older" })} data-testid="llm-delete-older">
          Xoá các dòng cũ hơn {RETENTION_DAYS} ngày
        </Button>
      </div>

      <ConfirmModal
        isOpen={pending !== null}
        onClose={() => setPending(null)}
        onConfirm={() => void confirmDeletion()}
        isDangerous
        title={pending?.kind === "selected" ? `Xoá ${selected.size} dòng nhật ký?` : `Xoá các dòng cũ hơn ${RETENTION_DAYS} ngày?`}
        message={
          <>
            {pending?.kind === "selected"
              ? `${selected.size} cuộc gọi đã chọn sẽ bị xoá cùng phản hồi đã lưu.`
              : `Mọi cuộc gọi cũ hơn ${RETENTION_DAYS} ngày sẽ bị xoá cùng phản hồi đã lưu, kể cả các dòng chưa hiện trên trang.`}{" "}
            Số liệu thống kê tính trên các dòng này (ví dụ chi phí các đoạn code của từng model) cũng mất. Không
            khôi phục được. Các dòng khác giữ nguyên.
          </>
        }
        confirmLabel={pending?.kind === "selected" ? "Xoá các dòng đã chọn" : `Xoá dòng cũ hơn ${RETENTION_DAYS} ngày`}
        cancelLabel="Huỷ"
      />

      <Dialog
        open={shown !== null}
        onClose={() => setShown(null)}
        size="xl"
        title={shown ? `Cuộc gọi · ${shown.role || "không rõ vai"} · ${formatTime(shown.created_at)}` : ""}
        testId="llm-response-dialog"
        footer={
          <Button variant="ghost" onClick={() => setShown(null)} data-testid="llm-response-close">
            Đóng
          </Button>
        }
      >
        {shown && (
          <>
            <p className={styles.detailMeta}>
              Mã cuộc gọi <span className={styles.mono}>{shown.call_id}</span> · {shown.provider} · {modelLabel(shown)}
              {shown.num_ctx > 0 ? ` · num_ctx ${shown.num_ctx}` : ""}
              {shown.prompt_chars > 0 ? ` · prompt ${shown.prompt_chars.toLocaleString("vi-VN")} ký tự` : ""}
              {shown.ok ? "" : ` · lỗi ${shown.error_kind}: ${shown.error_message}`}
            </p>
            {detailError && (
              <p role="alert" className={glass.helperText} data-testid="llm-detail-error">
                Không đọc được prompt: {detailError}
              </p>
            )}
            {!detail && !detailError && <p className={styles.empty}>Đang tải prompt...</p>}
            {detail && (
              <>
                <h4 className={styles.blockTitle}>System prompt</h4>
                <CopyBox text={detail.system_prompt} what="system prompt" copyTestId="llm-system-copy">
                  <pre className={styles.response} data-testid="llm-system-text">
                    {detail.system_prompt || "(không có)"}
                  </pre>
                </CopyBox>
                <h4 className={styles.blockTitle}>User prompt</h4>
                <CopyBox text={detail.user_prompt} what="user prompt" copyTestId="llm-user-copy">
                  <pre className={styles.response} data-testid="llm-user-text">
                    {detail.user_prompt || "(không có)"}
                  </pre>
                </CopyBox>
              </>
            )}
            <h4 className={styles.blockTitle}>Phản hồi</h4>
            <CopyBox text={shown.response} what="phản hồi" copyTestId="llm-response-copy">
              <pre className={styles.response} data-testid="llm-response-text">
                {shown.response || "(không có phản hồi)"}
              </pre>
            </CopyBox>
          </>
        )}
      </Dialog>
    </Card>
  );
}
