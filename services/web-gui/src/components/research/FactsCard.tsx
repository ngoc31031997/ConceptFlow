import { useState } from "react";
import { FACT_TYPES, getFactImpact, type FactType, type LinkState, type ResearchFact, type UnitStatus } from "../../api/client";
import { Button, Card, Select } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { FACT_TYPE_NAMES } from "./factTypes";
import { ImpactConfirm } from "./ImpactConfirm";
import styles from "./research.module.css";

type Filter = "all" | "ok" | "bad" | "na" | "none" | "drop";

const FILTERS: { id: Filter; label: string }[] = [
  { id: "all", label: "Tất cả" },
  { id: "ok", label: "Khớp" },
  { id: "bad", label: "Không khớp" },
  { id: "na", label: "Không tải được" },
  { id: "none", label: "Không có nguồn" },
  { id: "drop", label: "Đã bỏ" },
];

/** How many facts each filter shows. */
function countOf(facts: ResearchFact[], filter: Filter): number {
  return facts.filter((f) => matches(f, filter)).length;
}

function matches(fact: ResearchFact, filter: Filter): boolean {
  if (filter === "drop") return !fact.keep;
  if (!fact.keep) return false;
  return filter === "all" || fact.link.state === filter;
}

const LINK_CHIPS: Record<LinkState, { label: string; className: string }> = {
  ok: { label: "✓ khớp", className: styles.chipOk },
  bad: { label: "✗ không khớp", className: styles.chipWait },
  na: { label: "? không tải được", className: styles.chipTodo },
  none: { label: "– không có nguồn", className: styles.chipTodo },
  run: { label: "đang kiểm", className: styles.chipRun },
};

function LinkCell({ fact, checking }: { fact: ResearchFact; checking: boolean }) {
  const chip = checking ? LINK_CHIPS.run : LINK_CHIPS[fact.link.state];
  return (
    <>
      <span className={`${styles.chip} ${chip.className}`} data-testid={`fact-link-${fact.id}`} data-state={fact.link.state}>
        {checking && <span className={styles.spinner} aria-hidden="true" />}
        {chip.label}
      </span>
      {!checking && fact.link.why && <span className={styles.sub}>{fact.link.why}</span>}
      {!checking && fact.link.state === "bad" && fact.link.nearest && (
        <span className={styles.sub} data-testid={`fact-nearest-${fact.id}`}>
          Câu gần giống trong trang: “{fact.link.nearest}”
        </span>
      )}
    </>
  );
}

function SourceCell({ fact }: { fact: ResearchFact }) {
  if (!fact.url) return <span className={styles.hint}>không có nguồn</span>;
  return (
    <div className={styles.source}>
      {fact.sourceTitle && <b>{fact.sourceTitle}</b>}
      {(fact.publisher || fact.year) && (
        <span className={styles.sub}>
          {fact.publisher}
          {fact.publisher && fact.year ? " · " : ""}
          {fact.year}
        </span>
      )}
      <a href={fact.url} target="_blank" rel="noopener noreferrer">
        {fact.url.replace(/^https?:\/\//, "")}
      </a>
      {fact.quote && <p className={styles.quote}>“{fact.quote}”</p>}
    </div>
  );
}

interface PendingChange {
  title: string;
  lead: string;
  items: string[];
  label: string;
  run: () => void;
}

interface FactsCardProps {
  projectId: string;
  facts: ResearchFact[];
  /** Codes of the facts whose link is being checked, including ones just sent. */
  checking: ReadonlySet<string>;
  locked: boolean;
  onAdd: () => void;
  onEdit: (fact: ResearchFact) => void;
  onFlags: (id: string, flags: { type?: FactType; must?: boolean; keep?: boolean }) => Promise<boolean>;
  onRecheck: (id: string) => Promise<boolean>;
  /** Whether the table is approved, for the status chip and the button at its foot. */
  status: UnitStatus;
  /** An approval or other request of the step is running. */
  busy: boolean;
  onApprove: () => void;
  onUnapprove: () => void;
}

/**
 * Section 3 of the research step: the fact table, approved from the button at
 * its foot. Type and "bắt buộc nói" change straight in the table; wording, link
 * and quote open "Sửa". A change that later work already uses asks first and
 * lists what it sends back (docs/ux-ui-design-rules.md, rule 5), taking the
 * list from the server.
 */
export function FactsCard({ projectId, facts, checking, locked, onAdd, onEdit, onFlags, onRecheck, status, busy, onApprove, onUnapprove }: FactsCardProps) {
  const [filter, setFilter] = useState<Filter>("all");
  const [pending, setPending] = useState<PendingChange | null>(null);
  const shown = facts.filter((f) => matches(f, filter));

  // Runs the change at once when nothing uses the fact; otherwise asks first.
  async function guarded(fact: ResearchFact, title: string, label: string, run: () => void) {
    let items: string[];
    try {
      items = await getFactImpact(projectId, fact.id);
    } catch {
      items = ["Không tải được danh sách phần phụ thuộc; những bước đã làm sau dữ kiện này có thể phải làm lại."];
    }
    if (items.length === 0) {
      run();
      return;
    }
    setPending({ title, lead: `${fact.id}: ${fact.claimVi}`, items, label, run });
  }

  return (
    <Card data-testid="research-facts-card">
      <div className={styles.body}>
        <div className={styles.head}>
          <h3>③ Bảng dữ kiện</h3>
          <UnitStatusChip state={status} testId="research-status" />
          <span className={styles.grow} />
          <Button variant="ghost" className={styles.sm} onClick={onAdd} disabled={locked} data-testid="fact-add-open">
            + Thêm dữ kiện tay
          </Button>
        </div>
        <span className={styles.hint}>Loại và “bắt buộc nói” sửa ngay trong bảng; chữ, link và đoạn trích sửa ở nút Sửa.</span>
        {checking.size > 0 && (
          <p className={styles.checking} role="status" data-testid="facts-checking">
            <span className={styles.spinner} aria-hidden="true" />
            Đang tải và kiểm {checking.size} link…
          </p>
        )}
        <div className={styles.filters}>
          {FILTERS.map((f) => (
            <button
              key={f.id}
              type="button"
              className={filter === f.id ? `${styles.filter} ${styles.filterOn}` : styles.filter}
              aria-pressed={filter === f.id}
              onClick={() => setFilter(f.id)}
              data-testid={`fact-filter-${f.id}`}
            >
              {f.label} · {countOf(facts, f.id)}
            </button>
          ))}
        </div>
        <div className={styles.scroll}>
          <table className={`${styles.table} ${styles.facts}`} data-testid="research-facts-table">
            <thead>
              <tr>
                <th>Mã</th>
                <th>Dữ kiện</th>
                <th>Loại</th>
                <th>Nguồn và đoạn trích</th>
                <th>Kiểm link</th>
                <th>Bắt buộc nói</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {shown.length === 0 && (
                <tr>
                  <td colSpan={7} className={styles.hint} data-testid="research-facts-empty">
                    {facts.length === 0 ? "Chưa có dữ kiện. Lấy dữ kiện ở ② hoặc thêm tay." : "Không có dữ kiện ở bộ lọc này."}
                  </td>
                </tr>
              )}
              {shown.map((fact) => (
                <tr key={fact.id} className={fact.keep ? undefined : styles.dropped} data-testid={`fact-row-${fact.id}`}>
                  <td>
                    <span className={styles.code}>{fact.id}</span>
                    <span className={styles.sub}>{fact.question}</span>
                  </td>
                  <td>
                    {fact.claimVi}
                    {fact.note && <span className={styles.sub}>{fact.note}</span>}
                  </td>
                  <td>
                    <Select
                      value={fact.type}
                      aria-label={`Loại của ${fact.id}`}
                      disabled={locked}
                      onChange={(e) => {
                        const next = e.target.value as FactType;
                        void guarded(fact, `Đổi loại của ${fact.id}?`, "Đổi loại", () => void onFlags(fact.id, { type: next }));
                      }}
                      data-testid={`fact-type-${fact.id}`}
                    >
                      {FACT_TYPES.map((t) => (
                        <option key={t} value={t}>
                          {FACT_TYPE_NAMES[t]}
                        </option>
                      ))}
                    </Select>
                  </td>
                  <td>
                    <SourceCell fact={fact} />
                  </td>
                  <td>
                    <LinkCell fact={fact} checking={checking.has(fact.id)} />
                  </td>
                  <td className={styles.center}>
                    <input
                      type="checkbox"
                      checked={fact.must}
                      disabled={locked}
                      aria-label={`Bắt buộc nói ${fact.id}`}
                      onChange={(e) => {
                        const next = e.target.checked;
                        void guarded(fact, `Đổi “bắt buộc nói” của ${fact.id}?`, "Đổi", () => void onFlags(fact.id, { must: next }));
                      }}
                      data-testid={`fact-must-${fact.id}`}
                    />
                  </td>
                  <td>
                    <div className={styles.rowActions}>
                      <Button variant="ghost" className={styles.sm} onClick={() => onEdit(fact)} disabled={locked} data-testid={`fact-edit-${fact.id}`}>
                        Sửa
                      </Button>
                      {fact.url && (
                        <Button
                          variant="primary"
                          className={styles.sm}
                          onClick={() => void onRecheck(fact.id)}
                          disabled={locked || checking.has(fact.id)}
                          data-testid={`fact-recheck-${fact.id}`}
                        >
                          Kiểm lại
                        </Button>
                      )}
                      {fact.keep ? (
                        <Button
                          variant="ghost"
                          className={styles.sm}
                          onClick={() => void guarded(fact, `Bỏ dữ kiện ${fact.id}?`, "Bỏ dữ kiện", () => void onFlags(fact.id, { keep: false }))}
                          disabled={locked}
                          data-testid={`fact-keep-${fact.id}`}
                        >
                          Bỏ
                        </Button>
                      ) : (
                        <Button
                          variant="primary"
                          className={styles.sm}
                          onClick={() => void onFlags(fact.id, { keep: true })}
                          disabled={locked}
                          data-testid={`fact-keep-${fact.id}`}
                        >
                          Giữ lại
                        </Button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className={`${styles.actions} ${styles.actionsEnd}`}>
          <span className={styles.hint}>Duyệt được dù còn link chưa khớp; bước Rà soát nhắc lại các câu thoại dùng dữ kiện đó.</span>
          {status === "approved" ? (
            <Button variant="ghost" onClick={onUnapprove} disabled={locked || busy} data-testid="research-unapprove">
              Bỏ duyệt
            </Button>
          ) : (
            <Button variant="primary" onClick={onApprove} disabled={locked || busy} data-testid="research-approve">
              Duyệt bảng dữ kiện
            </Button>
          )}
        </div>
      </div>

      <ImpactConfirm
        open={pending !== null}
        onClose={() => setPending(null)}
        title={pending?.title ?? ""}
        lead={pending?.lead}
        list={pending?.items ?? []}
        undo="Đổi lại được bằng thao tác ngược; phần đã làm lại ở các bước sau thì không tự khôi phục."
        choices={pending ? [{ label: pending.label, variant: "impact", onChoose: pending.run, testId: "fact-change-confirm" }] : []}
        testId="fact-change-dialog"
      />
    </Card>
  );
}
