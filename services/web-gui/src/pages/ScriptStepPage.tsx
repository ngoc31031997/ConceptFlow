import { useContext, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { Card } from "../components/ui";
import { SelectableOption } from "../components/SelectableOption";
import type { ScriptSource } from "../context/ProjectDraftContext";
import { ContentLanguagePicker } from "../components/ContentLanguagePicker";
import { RenderEnginePicker } from "../components/RenderEnginePicker";
import { AuthoringModeBar } from "../components/AuthoringModeBar";
import { AuthoringModelPicker } from "../components/AuthoringModelPicker";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import { useAuthoringModels } from "../hooks/useAuthoringModels";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { createProjectDraft } from "../api/client";
import selectable from "../styles/selectable.module.css";
import styles from "./WizardSteps.module.css";

/**
 * Tình huống nào vào tab nào của chuỗi 1a → 1b → 1c.
 *
 * "idea" và "outline" cùng vào 1a vì dàn ý là thứ tab đó sở hữu: ô bên phải
 * của nó VỪA là chỗ AI trả kết quả về, VỪA là chỗ dán một dàn ý có sẵn. Khác
 * nhau ở chỗ Creator dùng nửa nào của màn hình, nên hai lựa chọn riêng vẫn
 * đáng có — chúng đặt đúng kỳ vọng trước khi trang mở ra.
 */
const ENTRY_ROUTE: Record<ScriptSource, string> = {
  idea: "/create/script/outline",
  outline: "/create/script/outline",
  storyboard: "/create/script/storyboard",
  code: "/create/script/code",
};

const ENTRY_LABEL: Record<ScriptSource, string> = {
  idea: "Bắt đầu từ 1a. Dàn ý",
  outline: "Sang 1a. Dàn ý để dán dàn ý",
  storyboard: "Sang 1b. Storyboard",
  code: "Sang 1c. Code",
};

/**
 * Bốn tình huống — chỉ chữ khác nhau giữa hai engine, còn đường đi thì không.
 */
function situationOptions(
  renderEngine: "manim" | "remotion",
): { value: ScriptSource; label: string; hint: string }[] {
  const codeName = renderEngine === "remotion" ? "code Remotion" : "script Manim";
  return [
    {
      value: "idea",
      label: "Chưa có gì, chỉ có ý tưởng",
      hint: "Nhập chủ đề rồi dựng dàn ý → storyboard → code cùng AI, từng bước một.",
    },
    {
      value: "outline",
      label: "Đã có dàn ý",
      hint: "Dán dàn ý sẵn có vào bước 1a, bỏ qua phần sinh dàn ý, đi thẳng sang storyboard.",
    },
    {
      value: "storyboard",
      label: "Đã có storyboard",
      hint: "Dán storyboard sẵn có vào bước 1b, chỉ còn sinh code là xong.",
    },
    {
      value: "code",
      label: `Đã có ${codeName}`,
      hint: `Dán thẳng vào bước 1c. Hệ thống kiểm tra ngay; chưa đúng chuẩn thì có sẵn prompt chuẩn hoá ${codeName} tại đó.`,
    },
  ];
}

/**
 * Bước 1 của 7 (xem AppShell's STEP_LABELS) — chọn điểm vào của chuỗi dựng
 * script, và chốt ba thứ chi phối cả chuỗi đó.
 *
 * CR-031: trang này chỉ còn là màn chọn. Trước đây nó vừa chọn tình huống vừa
 * ôm luôn cả trình soạn thảo cho hai tình huống "đã có code", nên cùng một
 * việc — dán code, lint nó, chuẩn hoá nó — tồn tại ở hai nơi (ở đây và ở tab
 * 1c) và hai nơi đó đã trôi khỏi nhau. Giờ mọi tình huống đều đi vào cùng một
 * chuỗi tab, mỗi artefact có đúng một chỗ ở.
 *
 * Ba lựa chọn dưới đây nằm ở đây chứ không rải trong các tab vì cả ba đều chi
 * phối những bước sau: ngôn ngữ chọn prompt và giọng đọc, engine quyết định
 * role prompt cho 1b/1c (RoleFor ở server), còn chế độ manual/API quyết định
 * cả ba tab chạy kiểu gì. Chế độ vẫn đổi được ở từng tab bất cứ lúc nào — nó
 * lưu trên project chứ không phải trên màn hình.
 */
export function ScriptStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const { mode: authoringMode, setMode: setAuthoringMode } = useAuthoringMode(draft.projectId);
  const { models: authoringModels, setModels: setAuthoringModels } = useAuthoringModels(draft.projectId);

  // A draft that already started a saga belongs to an existing project;
  // reusing it would overwrite that project's video.
  useEffect(() => {
    if (draft.hasSubmitted) dispatch({ type: "RESET" });
  }, [draft.hasSubmitted, dispatch]);

  const isRemotion = draft.renderEngine === "remotion";

  function handleEngineChange(engine: "manim" | "remotion") {
    dispatch({ type: "SET_RENDER_ENGINE", payload: engine });
    // Server phải biết engine trước khi render prompt cho storyboard/code
    // (RoleFor đọc project.RenderEngine), không chỉ lúc nộp render.
    // Best-effort: lỗi mạng ở đây không được chặn Creator đổi lựa chọn.
    if (draft.projectId) {
      void createProjectDraft(draft.projectId, "", draft.voiceLanguage, engine).catch(() => {});
    }
  }

  function handleContinue() {
    navigate(ENTRY_ROUTE[draft.scriptSource]);
  }

  return (
    <div data-testid="script-step-page">
      <AppShell
        currentStep={1}
        wide
        title={isRemotion ? "Bước 1 — Script Remotion" : "Bước 1 — Script Manim"}
        subtitle="Chốt ngôn ngữ, công cụ render và cách làm, rồi chọn bạn đang có sẵn tới đâu."
      >
        <div className={styles.settingsRow}>
          <ContentLanguagePicker
            value={draft.voiceLanguage}
            onChange={(lang) => dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang })}
          />
        </div>

        <div className={styles.settingsRow}>
          <RenderEnginePicker value={draft.renderEngine} onChange={handleEngineChange} />
        </div>

        {/* Không truyền `steps`: ở đây chưa có chủ đề và chưa có artefact nào
            để sinh, nên chỉ có công tắc chế độ. Nút chạy nằm ở các tab 1a–1c,
            nơi thật sự có dữ liệu đầu vào. */}
        <div className={styles.settingsRow}>
          <AuthoringModeBar
            llm={llm}
            mode={authoringMode}
            onModeChange={setAuthoringMode}
            projectId={draft.projectId}
          />
        </div>

        {/* Model-per-step picker: chỉ có ý nghĩa ở chế độ AI — chưa chọn "Gọi
            API" thì chưa có lượt gọi nào để áp dụng model. Đặt ngay dưới công
            tắc chế độ, cùng lý do AuthoringModeBar nằm ở đây: một lựa chọn cho
            cả ba tab, chọn một lần ở bước 1. */}
        {authoringMode === "ai" && llm?.enabled && (
          <div className={styles.settingsRow}>
            <AuthoringModelPicker
              models={authoringModels}
              onChange={setAuthoringModels}
              options={llm.models ?? []}
            />
          </div>
        )}

        <Card
          title="Bạn đang ở tình huống nào?"
          hint="Mỗi lựa chọn đưa bạn vào đúng tab đang cần làm — các tab còn lại vẫn bấm sang được bất cứ lúc nào."
        >
          <div className={selectable.stack} role="radiogroup" aria-label="Tình huống script">
            {situationOptions(draft.renderEngine).map((option) => (
              <SelectableOption
                key={option.value}
                selected={draft.scriptSource === option.value}
                onSelect={() => dispatch({ type: "SET_SCRIPT_SOURCE", payload: option.value })}
                label={option.label}
                hint={option.hint}
                testId={`script-source-${option.value}`}
              />
            ))}
          </div>
        </Card>
      </AppShell>

      <WizardNav
        hint={
          authoringMode === "ai" && llm?.enabled
            ? "Chế độ gọi API đang bật — các tab 1a–1c sẽ có nút chạy bằng AI."
            : "Chế độ copy prompt ra ngoài — đổi sang gọi API ở đây hoặc ở bất kỳ tab nào."
        }
        onNext={handleContinue}
        nextLabel={ENTRY_LABEL[draft.scriptSource]}
        nextTestId="script-step-next"
      />
    </div>
  );
}
