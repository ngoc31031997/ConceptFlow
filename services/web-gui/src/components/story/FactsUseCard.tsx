import { useState } from "react";
import type { StoryFact } from "../../api/client";
import { Card } from "../ui";
import { FACT_TYPE_NAMES } from "../research/factTypes";
import { ImpactConfirm } from "../research/ImpactConfirm";
import researchStyles from "../research/research.module.css";
import styles from "./story.module.css";

interface FactsUseCardProps {
  facts: StoryFact[];
  /** Turns the use of one fact on or off. */
  onUse: (id: string, use: boolean) => void;
  /** An action is in flight, or the AI is writing. */
  busy: boolean;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
}

/** What a fact of this kind obliges the story to do, shown under it while the story uses it. */
function KindNote({ fact }: { fact: StoryFact }) {
  if (fact.type === "hypothesis" && fact.used) {
    return <span className={styles.factWarn}>Giả thuyết: lời thoại phải nói rõ là “người ta cho rằng”</span>;
  }
  if (fact.type === "unknown") {
    return <span className={styles.factNote}>Chưa biết: chỉ dùng để nói khoa học chưa biết, không dùng làm khẳng định</span>;
  }
  if (fact.type === "misconception" && fact.used) {
    return <span className={styles.factNote}>Ngộ nhận: chỉ nhắc để gỡ</span>;
  }
  return null;
}

/**
 * Section 3 of the story step: the facts the story uses. A fact the script must
 * state is always used. Turning off a fact the chain or the payoff rests on asks
 * first, since the story goes back to review.
 */
export function FactsUseCard({ facts, onUse, busy, locked }: FactsUseCardProps) {
  const [pending, setPending] = useState<StoryFact | null>(null);
  const usedCount = facts.filter((f) => f.used).length;

  function toggle(fact: StoryFact, use: boolean) {
    if (!use && fact.cited) setPending(fact);
    else onUse(fact.id, use);
  }

  return (
    <Card data-testid="story-facts-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>③ Dữ kiện dùng trong chuyện</h3>
          <span className={researchStyles.hint} data-testid="story-facts-count">
            {usedCount}/{facts.length} dữ kiện · dữ kiện bắt buộc luôn dùng
          </span>
        </div>
        {facts.length === 0 ? (
          <p className={researchStyles.hint} data-testid="story-facts-empty">
            Chưa có dữ kiện. Khi bạn bỏ qua bước Nghiên cứu, AI tự liệt kê dữ kiện lúc viết khung và chúng hiện ở đây với nhãn “chưa có nguồn”.
          </p>
        ) : (
          <div className={researchStyles.scroll}>
            <table className={researchStyles.table}>
              <tbody>
                {facts.map((fact) => (
                  <tr key={fact.id} data-testid={`story-fact-${fact.id}`}>
                    <td style={{ width: 30 }}>
                      <input
                        type="checkbox"
                        checked={fact.used}
                        disabled={locked || busy || fact.must}
                        onChange={(e) => toggle(fact, e.target.checked)}
                        aria-label={`Dùng ${fact.id} trong chuyện`}
                        data-testid={`story-fact-use-${fact.id}`}
                      />
                    </td>
                    <td style={{ width: 150 }}>
                      <span className={styles.tag}>{fact.id}</span>{" "}
                      {fact.link === "none" && <span className={styles.noSource}>chưa có nguồn</span>}
                    </td>
                    <td>
                      <span className={styles.factText}>
                        <span>{fact.claimVi}</span>
                        <KindNote fact={fact} />
                      </span>
                    </td>
                    <td>
                      <span className={researchStyles.sub}>{FACT_TYPE_NAMES[fact.type]}</span>
                      {fact.must && <span className={researchStyles.sub}>bắt buộc</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
      <ImpactConfirm
        open={pending !== null}
        onClose={() => setPending(null)}
        title={`Thôi dùng ${pending?.id ?? ""} trong chuyện?`}
        lead={pending?.claimVi}
        list={["Chuỗi nhân quả hoặc payoff đang trích dữ kiện này: bạn cần sửa các ô đó", "Câu chuyện về chờ duyệt"]}
        undo="Bật lại ô chọn để dùng lại dữ kiện."
        choices={[
          {
            label: "Thôi dùng",
            variant: "impact",
            onChoose: () => {
              if (pending) onUse(pending.id, false);
            },
            testId: "story-fact-confirm",
          },
        ]}
        testId="story-fact-dialog"
      />
    </Card>
  );
}
