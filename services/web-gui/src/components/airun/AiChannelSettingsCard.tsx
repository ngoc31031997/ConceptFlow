import { useEffect, useState } from "react";
import {
  ApiError,
  getAiChannelSetting,
  setAiChannelDefault,
  setAiChannelStep,
  type AiChannel,
  type AiChannelSetting,
} from "../../api/client";
import { Card, Dropdown, FormField } from "../ui";
import styles from "../../pages/AiModelSettingsPage.module.css";

const CHANNEL_OPTIONS: { value: AiChannel; label: string }[] = [
  { value: "api", label: "API · model của hệ thống" },
  { value: "extension", label: "Extension · DeepSeek, Gemini hoặc Grok qua AI Bridge" },
  { value: "manual", label: "Thủ công · chép prompt, dán câu trả lời" },
];

/** The steps whose AI buttons follow the channel setting, with their names (llm-service CHANNEL_STEPS). */
const CHANNEL_STEPS: { step: string; label: string }[] = [
  { step: "idea", label: "Bước 1 · Ý tưởng" },
  { step: "brief", label: "Bước 2 · Creative brief" },
  { step: "research", label: "Bước 3 · Nghiên cứu" },
  { step: "story", label: "Bước 4 · Câu chuyện" },
  { step: "beats", label: "Bước 5 · Beat sheet" },
  { step: "script", label: "Bước 6 · Viết kịch bản" },
  { step: "lock", label: "Bước 7 · Rà soát và chốt" },
  { step: "visual_concept", label: "Bước 8 · Ngôn ngữ hình ảnh" },
  { step: "visual_plan", label: "Bước 9 · Kế hoạch hình" },
  { step: "visual_sequence", label: "Bước 10 · Chuỗi hình" },
  { step: "visual_storyboard", label: "Bước 11 · Storyboard" },
];

function errorText(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : "Không lưu được. Thử lại.";
}

/**
 * Cài đặt · Kênh chạy AI: which channel the "▶ Chạy AI" button of a step runs on by default, for
 * all steps and per step. Each run can still pick another channel from the button's ▾ menu. A choice
 * is saved as soon as it is picked; nothing already run changes.
 */
export function AiChannelSettingsCard() {
  const [setting, setSetting] = useState<AiChannelSetting | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);

  async function load() {
    setSetting(await getAiChannelSetting());
  }

  useEffect(() => {
    load().catch((err) => setError(err instanceof ApiError && err.message ? err.message : "Không đọc được cài đặt kênh."));
  }, []);

  async function change(save: () => Promise<void>, what: string) {
    setError(null);
    setSaved(null);
    try {
      await save();
      await load();
      setSaved(`Đã lưu kênh của ${what}.`);
    } catch (err) {
      setError(errorText(err));
    }
  }

  return (
    <Card data-testid="ai-channel-card">
      <div className={styles.form}>
        <div>
          <h3>Kênh chạy AI</h3>
          <p className={styles.hint}>
            Nút “▶ Chạy AI” của mỗi bước chạy bằng kênh này; menu ▾ cạnh nút đổi kênh cho một lần chạy.
          </p>
        </div>
        {setting && (
          <div className={styles.stepGrid}>
            <FormField label="Mặc định cho mọi bước">
              <Dropdown
                value={setting.defaultChannel}
                options={CHANNEL_OPTIONS}
                onChange={(c) => void change(() => setAiChannelDefault(c as AiChannel), "mọi bước")}
                aria-label="Kênh chạy AI mặc định"
                data-testid="ai-channel-default"
              />
            </FormField>
            {CHANNEL_STEPS.map(({ step, label }) => (
              <FormField key={step} label={label}>
                <Dropdown
                  value={setting.stepChannels[step] ?? ""}
                  options={[
                    { value: "", label: `Theo mặc định (${CHANNEL_OPTIONS.find((o) => o.value === setting.defaultChannel)?.label.split(" · ")[0]})` },
                    ...CHANNEL_OPTIONS,
                  ]}
                  onChange={(c) => void change(() => setAiChannelStep(step, c as AiChannel | ""), label)}
                  aria-label={`Kênh chạy AI của ${label}`}
                  data-testid={`ai-channel-step-${step}`}
                />
              </FormField>
            ))}
          </div>
        )}
        {error && (
          <p className={styles.error} role="alert">
            {error}
          </p>
        )}
        {saved && (
          <p className={styles.hint} role="status" data-testid="ai-channel-saved">
            {saved}
          </p>
        )}
      </div>
    </Card>
  );
}
