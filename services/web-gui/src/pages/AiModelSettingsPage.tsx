import { useEffect, useState } from "react";
import { AppShell } from "../components/AppShell";
import { Button, Card, Dropdown, FormField } from "../components/ui";
import { LOCAL_CODE_MODEL_WARNING } from "../components/localModelWarning";
import { ApiError, getAiModelSetting, setAiModelStepDefault, type AiModelSetting } from "../api/client";
import { useLlmStatus } from "../hooks/useLlmStatus";
import styles from "./AiModelSettingsPage.module.css";

/** What each configurable step does, as the Creator knows it. */
const STEP_LABELS: Record<string, string> = {
  brief: "Viết brief (Ý tưởng)",
  research: "Sinh câu hỏi (Nghiên cứu)",
  story_frame: "Câu chuyện (hướng tiếp cận, khung)",
  story: "Kịch bản",
  storyboard: "Visual (storyboard)",
  code: "Viết code",
  publish: "Gợi ý tiêu đề, mô tả, thẻ khi đăng",
  illustrations: "Vẽ hình minh hoạ (Thư viện hình)",
};

/** Steps whose output is code, where the local model can time out. */
const CODE_STEPS = new Set(["code", "illustrations"]);

function errorText(err: unknown, fallback: string): string {
  return err instanceof ApiError && err.message ? err.message : fallback;
}

/**
 * Cài đặt · Model AI mặc định: the model each AI step is pre-selected on when
 * a project is made, and runs on until the Creator picks another at the step
 * or the button. The settings belong to no project. A step nobody configured
 * is on the first model of the list.
 */
export function AiModelSettingsPage() {
  const llm = useLlmStatus();
  const [setting, setSetting] = useState<AiModelSetting | null>(null);
  const [stepChoices, setStepChoices] = useState<Record<string, string>>({});
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getAiModelSetting()
      .then((s) => {
        setSetting(s);
        setStepChoices(s.stepModels);
      })
      .catch((err) => setLoadError(errorText(err, "Không đọc được cài đặt.")));
  }, []);

  const models = llm?.models ?? [];
  const modelOptions = models.map((m) => ({ value: m.id, label: m.label }));
  const changedSteps = setting ? setting.steps.filter((step) => stepChoices[step] !== setting.stepModels[step]) : [];

  async function save() {
    if (!setting) return;
    setSaving(true);
    setError(null);
    setMessage(null);
    const stepModels = { ...setting.stepModels };
    try {
      for (const step of changedSteps) {
        await setAiModelStepDefault(step, stepChoices[step]);
        stepModels[step] = stepChoices[step];
      }
      setMessage("Đã lưu. Project mới được chọn sẵn model theo từng bước; chỗ nào bạn đã chọn model riêng thì giữ nguyên.");
    } catch (err) {
      setError(errorText(err, "Không lưu được. Thử lại."));
    } finally {
      setSetting({ ...setting, stepModels });
      setSaving(false);
    }
  }

  return (
    <AppShell
      wide
      title="Model AI mặc định"
      subtitle="Model được chọn sẵn cho từng bước khi làm project. Mỗi project và mỗi nút AI vẫn đổi được model riêng."
    >
      <Card>
        {loadError ? (
          <p className={styles.error} role="alert">
            {loadError}
          </p>
        ) : (
          <div className={styles.form}>
            <div className={styles.stepGrid}>
              {setting?.steps.map((step) => {
                const value = stepChoices[step] ?? "";
                const local = CODE_STEPS.has(step) && models.find((m) => m.id === value)?.local === true;
                return (
                  <FormField key={step} label={STEP_LABELS[step] ?? step}>
                    <Dropdown
                      value={value}
                      options={modelOptions}
                      onChange={(model) => setStepChoices((c) => ({ ...c, [step]: model }))}
                      aria-label={`Model AI mặc định cho ${STEP_LABELS[step] ?? step}`}
                      data-testid={`ai-step-model-${step}`}
                    />
                    {local && (
                      <p className={styles.warn} data-testid={`ai-step-model-${step}-local-warning`}>
                        {LOCAL_CODE_MODEL_WARNING}
                      </p>
                    )}
                  </FormField>
                );
              })}
            </div>

            {error && (
              <p className={styles.error} role="alert">
                {error}
              </p>
            )}
            {message && (
              <p className={styles.hint} role="status" data-testid="ai-default-model-saved">
                {message}
              </p>
            )}
            <div className={styles.actions}>
              <Button
                onClick={() => void save()}
                disabled={changedSteps.length === 0 || saving}
                data-testid="ai-default-model-save"
              >
                {saving ? "Đang lưu…" : "Lưu"}
              </Button>
            </div>
          </div>
        )}
      </Card>
    </AppShell>
  );
}
