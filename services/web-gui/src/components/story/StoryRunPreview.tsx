import type { ArcBeat } from "../../api/client";
import styles from "../airun/AiRun.module.css";
import { FactTags } from "./FactTags";

interface StoryPreview {
  central_question?: string;
  core_answer?: string;
  narrative_arc?: Array<Partial<ArcBeat>>;
}

/**
 * What applying a story run would store: the central question, the core answer and the beats of the
 * arc with the facts each rests on. A field the Creator wrote is already merged in, as it will be saved.
 */
export function StoryRunPreview({ preview }: { preview: unknown }) {
  const p = (preview ?? {}) as StoryPreview;
  const arc = Array.isArray(p.narrative_arc) ? p.narrative_arc : [];
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <p className={styles.previewHead}>Câu chuyện sẽ được lưu và về “Chờ duyệt”. Ô bạn tự viết được giữ nguyên.</p>
      <dl>
        <dt>Câu hỏi trung tâm</dt>
        <dd>{p.central_question || "(trống)"}</dd>
        <dt>Câu trả lời cốt lõi</dt>
        <dd>{p.core_answer || "(trống)"}</dd>
        <dt>Mạch kể</dt>
        <dd>
          <ol className={styles.previewList}>
            {arc.map((b, i) => (
              <li key={i}>
                <b>{b.beat}</b>: {b.text}
                <FactTags ids={b.facts ?? []} />
              </li>
            ))}
          </ol>
        </dd>
      </dl>
    </div>
  );
}
