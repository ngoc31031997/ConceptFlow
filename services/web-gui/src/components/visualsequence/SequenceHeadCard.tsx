import type { VisualSequenceState } from "../../api/client";
import researchStyles from "../research/research.module.css";
import { Card } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";

interface SequenceHeadCardProps {
  state: VisualSequenceState;
}

/**
 * Section 1 of step 10: the locked inputs the step reads and why the plan is
 * out of step with them.
 */
export function SequenceHeadCard({ state }: SequenceHeadCardProps) {
  const locked = state.locked_version > 0;

  return (
    <Card data-testid="seq-head-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>1. Đầu vào đã chốt</h3>
          {locked && state.draft.length === 0 ? (
            <span className={`${researchStyles.chip} ${researchStyles.chipOk}`}>
              ✓ Đã chốt v{state.locked_version}
            </span>
          ) : (
            <UnitStatusChip state="pending" />
          )}
          <span className={researchStyles.hint} data-testid="seq-head-count">
            {state.items_approved}/{state.coverage.units_total} unit đã duyệt ·{" "}
            {state.review.open_count} vấn đề cần xử lý
          </span>
        </div>
        <div data-testid="seq-input">
          <b>Đầu vào:</b> Kế hoạch hình v{state.input.plan_version} · {state.input.units_count} unit
          · Ngôn ngữ hình ảnh v{state.input.concept_version} · Kịch bản v
          {state.input.script_version}
        </div>
        {state.input.outdated.length > 0 && (
          <div className={researchStyles.banner} role="status" data-testid="seq-outdated">
            ⚠ Kế hoạch hình chưa theo bản chốt mới nhất: {state.input.outdated.join("; ")}. Vẫn sửa
            và duyệt được, nhưng chỉ chốt được sau khi bước 9 chốt lại.
          </div>
        )}
      </div>
    </Card>
  );
}
