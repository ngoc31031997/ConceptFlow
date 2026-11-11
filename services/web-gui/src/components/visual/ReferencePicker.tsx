import { useState } from "react";
import { useNavigate } from "react-router-dom";
import type { VisualConceptState } from "../../api/client";
import type { VisualConceptControls } from "../../hooks/useVisualConcept";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dialog } from "../ui";
import styles from "./visual.module.css";

/** Where the shared reference library is kept. */
export const REFERENCE_LIBRARY_ROUTE = "/settings/reference-styles";

/**
 * The reference descriptions of the shared library, each with a tick: the
 * ticked ones are what the AI reads when it writes the parts. Changing the
 * ticks once a part that reads them is written asks first, since those parts
 * then need redoing.
 */
export function ReferencePicker({
  state,
  concept,
  readOnly,
}: {
  state: VisualConceptState;
  concept: VisualConceptControls;
  readOnly: boolean;
}) {
  const navigate = useNavigate();
  const [pending, setPending] = useState<string[] | null>(null);
  const picked = state.refs.filter((r) => r.picked).map((r) => r.ref_id);
  const affected = state.units.filter((u) => u.written && u.reads.includes("Mô tả tham chiếu"));

  function toggle(id: string) {
    const next = picked.includes(id) ? picked.filter((x) => x !== id) : [...picked, id];
    if (affected.length > 0) setPending(next);
    else void concept.setRefs(next);
  }

  return (
    <Card data-testid="concept-refs">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>Tham chiếu phong cách</h3>
          <span className={researchStyles.hint}>
            Chọn mô tả trong thư viện để AI đọc khi viết quy tắc.
          </span>
          <span className={researchStyles.grow} />
          <Button
            variant="view"
            onClick={() => navigate(REFERENCE_LIBRARY_ROUTE)}
            data-testid="concept-refs-library"
          >
            Mở thư viện tham chiếu
          </Button>
        </div>
        {state.refs.length === 0 ? (
          <p className={researchStyles.hint} data-testid="concept-refs-empty">
            Thư viện tham chiếu chưa có mô tả nào. AI viết quy tắc chỉ từ kịch bản.
          </p>
        ) : (
          <div className={styles.refs}>
            {state.refs.map((r) => (
              <div key={r.ref_id} className={`${styles.ref} ${r.picked ? styles.refOn : ""}`}>
                <label className={styles.refLabel}>
                  <input
                    type="checkbox"
                    checked={r.picked}
                    onChange={() => toggle(r.ref_id)}
                    disabled={readOnly || concept.busy || concept.running !== null}
                    data-testid={`concept-ref-${r.ref_id}`}
                  />
                  {r.ref_id} · {r.name}
                </label>
                <span className={researchStyles.hint}>{r.summary}</span>
                <span>
                  {r.tags.map((t) => (
                    <span key={t} className={styles.tag}>
                      {t}
                    </span>
                  ))}
                </span>
              </div>
            ))}
          </div>
        )}
        <span className={researchStyles.hint}>
          {picked.length} mô tả đang được chọn. Đổi lựa chọn sau khi AI đã viết thì các phần đọc
          tham chiếu hiện nhãn “cần làm lại”.
        </span>
      </div>
      <Dialog
        open={pending !== null}
        onClose={() => setPending(null)}
        title="Đổi mô tả tham chiếu?"
        size="sm"
        testId="concept-refs-confirm"
        footer={
          <>
            <Button variant="ghost" onClick={() => setPending(null)}>
              Huỷ
            </Button>
            <Button
              variant="impact"
              onClick={() => {
                const next = pending ?? [];
                setPending(null);
                void concept.setRefs(next);
              }}
              data-testid="concept-refs-confirm-save"
            >
              Đổi và đánh dấu cần làm lại
            </Button>
          </>
        }
      >
        <p>Các phần đã viết đọc tham chiếu sẽ hiện nhãn “cần làm lại”:</p>
        <ul>
          {affected.map((u) => (
            <li key={u.key}>
              Phần {u.number} · {u.title}
            </li>
          ))}
        </ul>
        <p className={researchStyles.hint}>
          Các phần khác giữ nguyên. Bạn duyệt lại hoặc cho AI làm lại từng phần.
        </p>
      </Dialog>
    </Card>
  );
}
