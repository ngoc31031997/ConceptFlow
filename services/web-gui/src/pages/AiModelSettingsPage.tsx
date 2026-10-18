import { useEffect, useState } from "react";
import { AppShell } from "../components/AppShell";
import { Button, Card, Dropdown, FormField } from "../components/ui";
import { ApiError, getAiModelSetting, setAiModelDefault, type AiModelSetting } from "../api/client";
import { useLlmStatus } from "../hooks/useLlmStatus";
import styles from "./AiModelSettingsPage.module.css";

/**
 * Cài đặt · Model AI mặc định: the model every AI call runs on until the
 * Creator picks another at the button or for the project. Only a model that can
 * serve every step (code and drawings included) is offered; a model that
 * cannot, such as the local one, is still chosen per button.
 */
export function AiModelSettingsPage() {
  const llm = useLlmStatus();
  const [setting, setSetting] = useState<AiModelSetting | null>(null);
  const [choice, setChoice] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getAiModelSetting()
      .then((s) => {
        setSetting(s);
        setChoice(s.defaultModel);
      })
      .catch((err) => setLoadError(err instanceof ApiError && err.message ? err.message : "Không đọc được cài đặt."));
  }, []);

  const serverModel = setting?.serverModel ?? "";
  const options = [
    { value: "", label: serverModel ? `Mặc định của máy chủ (${serverModel})` : "Mặc định của máy chủ" },
    ...(llm?.models ?? []).filter((m) => m.code_ok !== false).map((m) => ({ value: m.id, label: m.label })),
  ];
  const changed = setting !== null && choice !== setting.defaultModel;

  async function save() {
    setSaving(true);
    setError(null);
    setMessage(null);
    try {
      await setAiModelDefault(choice);
      setSetting((s) => (s ? { ...s, defaultModel: choice } : s));
      setMessage("Đã lưu. Project mới và các chỗ chưa chọn model riêng sẽ chạy bằng model này.");
    } catch (err) {
      setError(err instanceof ApiError && err.message ? err.message : "Không lưu được. Thử lại.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <AppShell
      title="Model AI mặc định"
      subtitle="Model mà mọi lượt gọi AI chạy khi bạn chưa chọn gì khác. Mỗi project và mỗi nút AI vẫn đổi được model riêng."
    >
      <Card>
        {loadError ? (
          <p className={styles.error} role="alert">
            {loadError}
          </p>
        ) : (
          <div className={styles.form}>
            <FormField label="Model mặc định">
              <Dropdown
                value={choice}
                options={options}
                onChange={setChoice}
                disabled={setting === null}
                aria-label="Model AI mặc định"
                data-testid="ai-default-model"
              />
            </FormField>
            <p className={styles.hint}>
              Chỉ liệt kê model dùng được cho mọi bước, kể cả viết code và vẽ hình. Model chạy trên máy (Ollama) chọn riêng
              tại từng nút AI.
            </p>
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
              <Button onClick={() => void save()} disabled={!changed || saving} data-testid="ai-default-model-save">
                {saving ? "Đang lưu…" : "Lưu"}
              </Button>
            </div>
          </div>
        )}
      </Card>
    </AppShell>
  );
}
