import { useContext, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import type { ResearchFact, ResearchState } from "../api/client";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import type { StepStripProps } from "../components/StepStrip";
import { Button, Card } from "../components/ui";
import { CoverageCard } from "../components/research/CoverageCard";
import { FactAddModal } from "../components/research/FactAddModal";
import { FactEditModal } from "../components/research/FactEditModal";
import { FactsCard } from "../components/research/FactsCard";
import { ImpactConfirm } from "../components/research/ImpactConfirm";
import { PasteCard } from "../components/research/PasteCard";
import { PromptCard } from "../components/research/PromptCard";
import { QuestionsCard } from "../components/research/QuestionsCard";
import styles from "../components/research/research.module.css";
import { UnitStatusChip } from "../components/units/UnitStatusChip";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useAiBridge, type AiBridgeModes } from "../hooks/useAiBridge";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useResearch } from "../hooks/useResearch";
import { VIEW_RESEARCH, readOnlyReason, storyRoute } from "../utils/flow";

const SUBTITLE =
  "Sinh câu hỏi cần tra, chép prompt ra công cụ nghiên cứu bên ngoài, rồi dán kết quả lại. Hệ thống tải từng link và kiểm đoạn trích có thật trong trang không.";

/** Research needs sources, so DeepSeek reasons and searches the web. */
const RESEARCH_MODES: AiBridgeModes = { deepThink: true, search: true };

const ROLE =
  "Thu thập dữ kiện có nguồn; phân biệt sự thật, giả thuyết và điểm chưa biết. Bước tuỳ chọn: bỏ qua thì dữ kiện mang nhãn “chưa có nguồn” và không bước nào bị chặn.";

/** Where the step stands and what comes next, for the line under the heading. */
function stripOf(state: ResearchState): StepStripProps {
  const kept = state.facts.filter((f) => f.keep);
  const count = (s: string) => kept.filter((f) => f.link.state === s).length;
  if (state.gate.skipped) {
    return {
      tone: "todo",
      label: "Đã bỏ qua",
      message: "Không bước nào bị chặn. Dữ kiện trong chuyện do AI tự biết và mang nhãn “chưa có nguồn”.",
    };
  }
  if (state.gate.status === "approved") {
    return { tone: "ok", label: "Bảng dữ kiện đã duyệt", message: "Tiếp theo: xây dựng câu chuyện từ các dữ kiện này." };
  }
  if (state.facts.length === 0) {
    return {
      tone: "todo",
      label: "Chưa có dữ kiện",
      message: "Sinh hoặc tự thêm câu hỏi, chép prompt sang công cụ nghiên cứu, rồi dán kết quả vào ③.",
    };
  }
  const running = count("run");
  return {
    tone: running > 0 ? "run" : "wait",
    label: `Chờ bạn duyệt · ${kept.length} dữ kiện`,
    message:
      `${count("ok")} khớp, ${count("bad")} không khớp, ${count("na")} không tải được, ${count("none")} không có nguồn` +
      (running > 0 ? `, ${running} đang kiểm` : "") +
      ".",
  };
}

/**
 * Step 3 of the 21-step flow (optional): research and fact-check. The Creator
 * gets the questions, takes a prompt to an outside research tool, pastes the
 * answer back, and the server checks every quote against its page. The step is
 * done when the fact table is approved or the step is skipped; nothing after it
 * is held back either way.
 */
export function ResearchStepPage() {
  const params = useParams<{ id?: string }>();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const projectId = params.id ?? draft.projectId;
  const research = useResearch(projectId, !!projectId);
  const [skipping, setSkipping] = useState(false);
  const [editing, setEditing] = useState<ResearchFact | null>(null);
  const [adding, setAdding] = useState(false);
  const [pasteText, setPasteText] = useState("");
  const [bridgeError, setBridgeError] = useState<string | null>(null);
  const bridge = useAiBridge();

  if (!projectId) return <Navigate to="/" replace />;

  const state = research.state;
  const locked = !flow.editable;
  const aiAvailable = llm?.enabled === true;

  async function readPaste(raw: string): Promise<boolean> {
    const read = await research.pasteResults(raw);
    if (read) setPasteText("");
    return read;
  }

  /** Runs the prompt on DeepSeek, puts the answer in section 3 and reads it, as a paste by hand would be. */
  async function runOnDeepSeek(prompt: string) {
    setBridgeError(null);
    const outcome = await bridge.run("deepseek", prompt, RESEARCH_MODES);
    if (!outcome.ok) {
      setBridgeError(outcome.cancelled ? null : outcome.message);
      return;
    }
    setPasteText(outcome.text);
    await readPaste(outcome.text);
  }

  function content(s: ResearchState) {
    const checking = new Set(s.facts.filter((f) => f.link.state === "run" || research.holding.has(f.id)).map((f) => f.id));
    const checkingQuestions = new Set(s.facts.filter((f) => checking.has(f.id)).map((f) => f.question));
    if (s.gate.skipped) {
      return (
        <>
          <Card data-testid="research-skipped-card">
            <div className={styles.body}>
              <h3>Khi bỏ qua bước này</h3>
              <ul className={styles.confirmList}>
                <li>Ở bước Câu chuyện, AI tự liệt kê dữ kiện cần dùng từ hiểu biết của nó. Mỗi dữ kiện vẫn có mã (F01, F02…) và mang nhãn “chưa có nguồn”.</li>
                <li>Beat và lời thoại tham chiếu mã dữ kiện như bình thường.</li>
                <li>Bước Rà soát và chốt hiện cảnh báo “dữ kiện chưa có nguồn”; bạn chấp nhận là chốt được.</li>
                <li>Ghi chú bàn giao mang nhãn này để các vai AI phía sau không nói hay vẽ quá lên.</li>
              </ul>
              <div className={styles.actions}>
                <Button variant="ghost" onClick={() => void research.skip(false)} disabled={locked || research.busy} data-testid="research-unskip">
                  Làm nghiên cứu
                </Button>
              </div>
            </div>
          </Card>
        </>
      );
    }
    return (
      <>
        <div className={styles.optional} data-testid="research-optional">
          <span className={styles.tag}>Tuỳ chọn</span>
          <span>Bỏ qua thì dữ kiện mang nhãn “chưa có nguồn”; bước Rà soát hiện cảnh báo nhưng không chặn.</span>
          <Button variant="impact" className={styles.sm} onClick={() => setSkipping(true)} disabled={locked} data-testid="research-skip">
            Bỏ qua bước này
          </Button>
        </div>
        <QuestionsCard
          projectId={projectId}
          questions={s.questions}
          coverage={s.coverage}
          stale={s.questionsStale}
          aiAvailable={aiAvailable}
          running={research.running}
          run={research.run}
          locked={locked}
          idea={s.idea}
          brief={s.brief}
          onReloadInput={research.reload}
          onGenerate={research.generateQuestions}
          onAdd={research.addQuestion}
          onSave={research.saveQuestion}
          onDelete={research.deleteQuestion}
        />
        <PromptCard
          projectId={projectId}
          questions={s.questions}
          bridge={{
            available: bridge.providers.includes("deepseek"),
            stage: bridge.stage,
            modes: RESEARCH_MODES,
            error: bridgeError,
            enabled: !locked && !research.busy,
            onRun: (prompt) => void runOnDeepSeek(prompt),
            onCancel: bridge.cancel,
          }}
        />
        <PasteCard
          pastes={s.pastes}
          message={research.pasteMessage}
          busy={research.busy}
          locked={locked}
          text={pasteText}
          onTextChange={setPasteText}
          onRead={() => readPaste(pasteText)}
        />
        <FactsCard
          projectId={projectId}
          facts={s.facts}
          checking={checking}
          locked={locked}
          onAdd={() => setAdding(true)}
          onEdit={setEditing}
          onFlags={research.setFlags}
          onRecheck={(id) => research.recheck([id])}
        />
        <CoverageCard
          projectId={projectId}
          questions={s.questions}
          coverage={s.coverage}
          checking={checkingQuestions}
        />
        <Card data-testid="research-approve-card">
          <div className={styles.body}>
            <div className={styles.head}>
              <h3>Duyệt bảng dữ kiện</h3>
              <UnitStatusChip state={s.gate.status} testId="research-status" />
            </div>
            <p className={styles.hint}>
              Duyệt không cần mọi link khớp. Dữ kiện chưa khớp mang nhãn tương ứng ở các bước sau, và bước Rà soát nhắc lại những
              câu thoại dùng chúng.
            </p>
            <div className={`${styles.actions} ${styles.actionsEnd}`}>
              {s.gate.status === "approved" ? (
                <Button variant="ghost" onClick={() => void research.unapprove()} disabled={locked || research.busy} data-testid="research-unapprove">
                  Bỏ duyệt
                </Button>
              ) : (
                <Button variant="primary" onClick={() => void research.approve()} disabled={locked || research.busy} data-testid="research-approve">
                  Duyệt bảng dữ kiện
                </Button>
              )}
            </div>
          </div>
        </Card>
      </>
    );
  }

  return (
    <div data-testid="research-step-page">
      <AppShell
        viewStep={VIEW_RESEARCH}
        wide
        title="Nghiên cứu và kiểm chứng"
        subtitle={SUBTITLE}
        role={ROLE}
        strip={state ? stripOf(state) : undefined}
      >
        {locked && flow.status && (
          <p className={styles.banner} role="status" data-testid="research-read-only">
            {readOnlyReason(flow.status)}
          </p>
        )}
        {research.actionError && (
          <p className={styles.error} role="alert" data-testid="research-error">
            {research.actionError}
          </p>
        )}
        {!state && research.loadError && (
          <>
            <p className={styles.error} role="alert" data-testid="research-load-error">
              Không tải được bước Nghiên cứu: {research.loadError}
            </p>
            <div className={styles.actions}>
              <Button variant="ghost" onClick={research.reload} data-testid="research-reload">
                Tải lại
              </Button>
            </div>
          </>
        )}
        {!state && !research.loadError && (
          <p className={styles.hint} role="status">
            Đang tải…
          </p>
        )}
        {state && <div className={styles.body}>{content(state)}</div>}
      </AppShell>

      <WizardNav
        allowWhenLocked
        isBlocked={false}
        hint={
          state?.gate.resolved
            ? "Nghiên cứu đã xong hoặc đã bỏ qua. Sang bước Câu chuyện."
            : "Duyệt bảng dữ kiện hoặc bỏ qua nghiên cứu để sang bước Câu chuyện."
        }
        onBack={() => navigate("/create/script/brief")}
        onNext={() => navigate(storyRoute(projectId))}
        nextDisabled={!state?.gate.resolved}
        nextLabel="Lưu và sang câu chuyện"
        nextTestId="research-next"
      />

      <ImpactConfirm
        open={skipping}
        onClose={() => setSkipping(false)}
        title="Bỏ qua bước Nghiên cứu?"
        lead="Các bước sau dùng dữ kiện AI tự biết, chưa có nguồn."
        list={[
          "Câu chuyện, beat sheet và lời thoại gắn nhãn “chưa có nguồn” cho mọi dữ kiện",
          "Bước Rà soát và chốt hiện một cảnh báo; chấp nhận là chốt được",
          "Ghi chú bàn giao mang nhãn này để các vai AI phía sau không nói hay vẽ quá lên",
          "Bảng dữ kiện hiện có giữ nguyên",
        ]}
        undo="Bấm “Làm nghiên cứu” để quay lại bước này bất cứ lúc nào."
        choices={[{ label: "Bỏ qua nghiên cứu", variant: "impact", onChoose: () => void research.skip(true), testId: "research-skip-confirm" }]}
        testId="research-skip-dialog"
      />
      <FactEditModal projectId={projectId} fact={editing} onClose={() => setEditing(null)} onSave={research.saveFact} />
      <FactAddModal open={adding} onClose={() => setAdding(false)} questions={state?.questions ?? []} onAdd={research.addFact} />
    </div>
  );
}
