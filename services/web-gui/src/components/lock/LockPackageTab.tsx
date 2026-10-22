import { useState } from "react";
import type { ScriptLockState } from "../../api/client";
import type { ScriptLockControls } from "../../hooks/useScriptLock";
import { ConfirmModal } from "../ConfirmModal";
import researchStyles from "../research/research.module.css";
import { formatSeconds } from "../script/ScriptOverviewCard";
import scriptStyles from "../script/script.module.css";
import { Button, Card, FormField, TextInput } from "../ui";
import styles from "./lock.module.css";
import { STEPS_READ } from "./roles";

interface LockPackageTabProps {
  lock: ScriptLockControls;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
  /** Opens the handoff tab. */
  onOpenHandoff: () => void;
  /** Opens the script screen. */
  onOpenScript: () => void;
}

function factLabel(f: ScriptLockState["package"]["facts"][number], noSource: boolean): string {
  if (noSource) return `${f.id} · chưa có nguồn`;
  if (f.type === "hypothesis") return `${f.id} · giả thuyết`;
  if (f.type === "misconception") return `${f.id} · ngộ nhận`;
  if (f.type === "unknown") return `${f.id} · chưa biết`;
  if (f.link === "ok") return `${f.id} ✓`;
  if (f.link === "bad") return `${f.id} ✗`;
  return `${f.id} ?`;
}

/**
 * The third tab: the conditions to lock, the two sets the later steps read
 * (A the script, B the handoff notes), what each later step reads, and the
 * history of locked versions. Locking the first version is immediate; locking
 * a later one is confirmed with the changes it records.
 */
export function LockPackageTab({ lock, locked, onOpenHandoff, onOpenScript }: LockPackageTabProps) {
  const state = lock.state;
  const [confirming, setConfirming] = useState(false);
  const [note, setNote] = useState("");
  if (!state) return null;
  const busy = locked || lock.busy || lock.running !== null;
  const pkg = state.package;
  const next = state.nextVersion;
  const hasLocked = state.lockedVersion > 0;
  const showLock = !hasLocked || state.draft.length > 0;
  const sectionCount = (key: string) => state.handoff.sections[key] ?? 0;

  const doLock = () => void lock.lock(note).then((ok) => ok && setNote(""));

  return (
    <>
      <Card data-testid="lock-conditions-card">
        <div className={researchStyles.body}>
          <h3>Điều kiện chốt</h3>
          <div className={styles.cond}>
            {state.conditions.map((c) => (
              <span className={c.ok ? styles.condOk : styles.condWarn} key={c.text}>
                {c.ok ? "✓ " : "! "}
                {c.text}
              </span>
            ))}
          </div>
        </div>
      </Card>

      <div className={styles.grid2}>
        <Card data-testid="lock-package-a">
          <div className={researchStyles.body}>
            <div className={researchStyles.head}>
              <h3>A · Kịch bản</h3>
              <span className={researchStyles.hint}>nội dung chính thức</span>
            </div>
            <dl className={scriptStyles.kv}>
              <dt>Chủ đề</dt>
              <dd>{pkg.topic}</dd>
              <dt>Đối tượng</dt>
              <dd>{pkg.audience}</dd>
              <dt>Thời lượng</dt>
              <dd>
                mục tiêu {pkg.targetLabel} · ước tính {formatSeconds(pkg.totalSeconds)}
              </dd>
              <dt>Lời thoại</dt>
              <dd>
                {pkg.lines} câu, chia {pkg.beats} beat
              </dd>
            </dl>
            <span className={styles.eyebrow}>Trình tự và chuyển tiếp</span>
            <ol className={styles.chain}>
              {pkg.chain.map((c, i) => (
                <li key={i}>
                  <b>{c.fn}:</b> {c.title}
                  {c.next && <span className={styles.chainNext}>→ {c.next}</span>}
                </li>
              ))}
            </ol>
            <span className={styles.eyebrow}>Nguồn và lưu ý độ chính xác</span>
            <span className={scriptStyles.tags}>
              {pkg.facts.map((f) => (
                <span className={scriptStyles.flag} key={f.id}>
                  {factLabel(f, pkg.noSource)}
                </span>
              ))}
            </span>
            <span className={researchStyles.hint}>
              {pkg.facts.length} dữ kiện · {pkg.hypotheses} giả thuyết nói thận trọng · {pkg.unmatched} chưa khớp nguồn
            </span>
            <div>
              <Button variant="view" className={researchStyles.sm} onClick={onOpenScript} data-testid="lock-open-script">
                Xem lời thoại
              </Button>
            </div>
          </div>
        </Card>

        <Card data-testid="lock-package-b">
          <div className={researchStyles.body}>
            <div className={researchStyles.head}>
              <h3>B · Ghi chú bàn giao</h3>
              <span className={researchStyles.hint}>hướng dẫn cho các vai AI</span>
            </div>
            <dl className={scriptStyles.kv}>
              <dt>Mục tiêu beat</dt>
              <dd>{state.handoff.beats} beat</dd>
              <dt>Bắt buộc truyền đạt</dt>
              <dd>{sectionCount("must")} mục</dd>
              <dt>Thuật ngữ</dt>
              <dd>
                {state.handoff.glossary} thuật ngữ chung · {sectionCount("terms")} theo beat
              </dd>
              <dt>Phải giữ chính xác</dt>
              <dd>{sectionCount("exact")} mục</dd>
              <dt>Được sáng tạo</dt>
              <dd>{sectionCount("free")} mục</dd>
              <dt>Chưa chắc</dt>
              <dd>{sectionCount("unsure")} mục</dd>
            </dl>
            <div>
              <Button variant="view" className={researchStyles.sm} onClick={onOpenHandoff} data-testid="lock-open-handoff">
                Xem ghi chú bàn giao
              </Button>
            </div>
          </div>
        </Card>
      </div>

      <Card data-testid="lock-reads-card">
        <div className={researchStyles.body}>
          <h3>Các bước sau đọc gì</h3>
          <span className={researchStyles.hint}>Bộ bàn giao lưu trong cơ sở dữ liệu cùng phiên bản đã chốt. Các bước sau đọc trực tiếp từ đó, không xuất file.</span>
          <div className={styles.scroll}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Bước</th>
                  <th>Đọc</th>
                </tr>
              </thead>
              <tbody>
                {STEPS_READ.map((r) => (
                  <tr key={r.step}>
                    <td>
                      <b>{r.step}</b>
                    </td>
                    <td>{r.reads}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </Card>

      <Card data-testid="lock-versions-card">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Phiên bản</h3>
            <span className={hasLocked ? styles.chipOk : styles.chipOpen} data-testid="lock-version-chip">
              {hasLocked ? `✓ v${state.lockedVersion} đã chốt` : "Bản nháp, chưa chốt"}
            </span>
          </div>
          {state.versions.length > 0 ? (
            <div className={styles.scroll}>
              <table className={styles.table} data-testid="lock-history">
                <thead>
                  <tr>
                    <th>Bản</th>
                    <th>Lúc</th>
                    <th>Ghi chú</th>
                  </tr>
                </thead>
                <tbody>
                  {[...state.versions].reverse().map((v) => (
                    <tr key={v.version}>
                      <td>v{v.version}</td>
                      <td>{new Date(v.lockedAt).toLocaleString("vi-VN")}</td>
                      <td>{v.note}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <span className={researchStyles.hint}>Chưa có phiên bản nào được chốt.</span>
          )}
          {hasLocked && state.draft.length > 0 && (
            <div className={styles.draft} data-testid="lock-draft">
              <b>
                Bản nháp v{next}: {state.draft.length} thay đổi so với v{state.lockedVersion}
              </b>
              <ul>
                {state.draft.map((d) => (
                  <li key={d}>{d}</li>
                ))}
              </ul>
              <span>
                Các bước sau vẫn dùng v{state.lockedVersion} tới khi bạn chốt v{next}.
              </span>
            </div>
          )}
          <p className={researchStyles.hint}>
            Chốt là đóng băng một phiên bản để các bước sau dùng. Sửa sau khi chốt tạo bản nháp mới; không có gì bị khoá cứng.
            {hasLocked && state.draft.length === 0 && ` Sửa ở bất kỳ bước nào từ 1 tới 7 sẽ mở bản nháp v${next}.`}
          </p>
          {showLock && (
            <>
              <FormField label="Ghi chú phiên bản">
                <TextInput value={note} onChange={(e) => setNote(e.target.value)} placeholder={hasLocked ? "" : "Chốt lần đầu"} data-testid="lock-note" />
              </FormField>
              <div className={researchStyles.actions}>
                <span className={researchStyles.hint}>{state.canLock ? "Đủ điều kiện chốt." : "Chưa đủ điều kiện chốt: xem mục Điều kiện chốt."}</span>
                <Button
                  variant={hasLocked ? "impact" : "primary"}
                  onClick={() => (hasLocked ? setConfirming(true) : doLock())}
                  disabled={busy || !state.canLock}
                  data-testid="lock-run"
                >
                  Chốt phiên bản v{next}
                </Button>
              </div>
            </>
          )}
        </div>
      </Card>

      <ConfirmModal
        isOpen={confirming}
        onClose={() => setConfirming(false)}
        onConfirm={() => {
          setConfirming(false);
          doLock();
        }}
        title={`Chốt phiên bản v${next}?`}
        message={
          <>
            <p>
              Bản nháp có {state.draft.length} thay đổi so với v{state.lockedVersion}:
            </p>
            <ul>
              {state.draft.map((d) => (
                <li key={d}>{d}</li>
              ))}
            </ul>
            <p>
              v{state.lockedVersion} vẫn lưu trong lịch sử phiên bản. Các bước sau chưa đọc phiên bản kịch bản nên chưa cảnh nào phải làm lại; từng bước sẽ đọc bản đã chốt khi nó được nối.
            </p>
          </>
        }
        confirmLabel={`Chốt v${next}`}
      />
    </>
  );
}
