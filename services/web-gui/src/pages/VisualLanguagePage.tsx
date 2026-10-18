import { useCallback, useEffect, useState } from "react";
import { AppShell } from "../components/AppShell";
import { ConfirmModal } from "../components/ConfirmModal";
import { Button, Card, Dialog, FormField, TextArea, TextInput } from "../components/ui";
import {
  activateVisualLanguage,
  copyVisualLanguage,
  deleteVisualLanguage,
  languageSampleUrl,
  listLanguageSamples,
  listVisualLanguages,
  rebuildLanguageSamples,
  updateVisualLanguage,
  type LanguageSample,
  type VisualLanguage,
  type VisualLanguageRules,
} from "../api/client";
import glass from "../styles/glass.module.css";
import listStyles from "./PromptSettingsPage.module.css";
import styles from "./VisualLanguagePage.module.css";

const SCENE_LABEL: Record<LanguageSample["scene"], string> = {
  fig: "Cảnh thử 1: lòng quả sung (sáng, hồng), có nhân vật",
  space: "Cảnh thử 2: vũ trụ (tối)",
};

const RULES: { key: keyof VisualLanguageRules; label: string; hint: string }[] = [
  { key: "illustration_rules", label: "Luật vẽ hình", hint: "người vẽ hình đọc" },
  { key: "backdrop_rules", label: "Luật vẽ nền", hint: "người vẽ nền đọc" },
  { key: "scene_rules", label: "Luật cảnh", hint: "Đạo diễn hình ảnh đọc: cách dùng ánh sáng; màu do bảng màu của video quyết định" },
];

type Draft = { name: string; note: string; description: string; rules: VisualLanguageRules };

function draftOf(l: VisualLanguage): Draft {
  return { name: l.name, note: l.note, description: l.description, rules: { ...l.rules } };
}

/** Sample of a scene, in the order wide then tall, so the page can lay a pair side by side. */
function samplesOf(samples: LanguageSample[], scene: LanguageSample["scene"]): LanguageSample[] {
  return (["wide", "tall"] as const)
    .map((frame) => samples.find((s) => s.scene === scene && s.frame === frame))
    .filter((s): s is LanguageSample => s !== undefined);
}

/**
 * Ngôn ngữ hình của kênh: cách dựng khối, nét, ánh sáng, cách vẽ nhân vật.
 * Chỉ một ngôn ngữ bật tại một thời điểm; video mới ghim ngôn ngữ đang bật lúc
 * tạo project, video đã tạo giữ ngôn ngữ và phiên bản đã ghim. Ngôn ngữ hệ
 * thống chỉ xem và sao chép; sửa luật của bản của bạn tạo phiên bản mới.
 */
export function VisualLanguagePage() {
  const [languages, setLanguages] = useState<VisualLanguage[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [samples, setSamples] = useState<LanguageSample[]>([]);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [rebuilding, setRebuilding] = useState(false);
  const [status, setStatus] = useState<string | null>(null);
  const [editing, setEditing] = useState<Draft | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      setLanguages(await listVisualLanguages());
    } catch (e) {
      setStatus(e instanceof Error && e.message ? e.message : "Không tải được danh sách ngôn ngữ hình. Vui lòng thử lại.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const selected = languages.find((l) => l.id === selectedId) ?? languages[0] ?? null;
  const active = languages.find((l) => l.is_active) ?? null;
  const selectedKey = selected?.id ?? null;

  useEffect(() => {
    if (!selectedKey) {
      setSamples([]);
      return;
    }
    let cancelled = false;
    listLanguageSamples(selectedKey)
      .then((s) => {
        if (!cancelled) setSamples(s);
      })
      .catch(() => {
        if (!cancelled) setSamples([]);
      });
    return () => {
      cancelled = true;
    };
  }, [selectedKey]);

  async function run(action: () => Promise<string | void>, ok: string) {
    setBusy(true);
    setStatus(null);
    try {
      const nextId = await action();
      await reload();
      if (nextId) setSelectedId(nextId);
      setStatus(ok);
    } catch (e) {
      setStatus(e instanceof Error && e.message ? e.message : "Thao tác không thành công. Vui lòng thử lại.");
    } finally {
      setBusy(false);
    }
  }

  async function rebuildSamples() {
    if (!selected) return;
    setRebuilding(true);
    setStatus(null);
    try {
      setSamples(await rebuildLanguageSamples(selected.id));
      setStatus("Đã dựng lại ảnh mẫu.");
    } catch (e) {
      setStatus(e instanceof Error && e.message ? e.message : "Không dựng được ảnh mẫu. Vui lòng thử lại.");
    } finally {
      setRebuilding(false);
    }
  }

  const readOnly = selected?.is_system === true;
  const draftIncomplete =
    editing === null ||
    editing.name.trim() === "" ||
    Object.values(editing.rules).some((text) => text.trim() === "");

  async function saveEdit() {
    if (!selected || !editing) return;
    const draft = editing;
    setEditing(null);
    await run(async () => {
      await updateVisualLanguage(selected.id, draft);
    }, `Đã lưu thành phiên bản mới. Video đã ghim giữ phiên bản cũ.`);
  }

  const setRule = (key: keyof VisualLanguageRules, text: string) =>
    setEditing((d) => (d ? { ...d, rules: { ...d.rules, [key]: text } } : d));

  return (
    <div data-testid="visual-language-page">
      <AppShell
        title="Ngôn ngữ hình"
        subtitle="Ngôn ngữ hình chung của kênh: cách dựng khối, nét, ánh sáng, cách vẽ nhân vật. Chỉ một ngôn ngữ bật tại một thời điểm. Video mới ghim ngôn ngữ đang bật lúc tạo project; video đã tạo giữ ngôn ngữ và phiên bản đã ghim."
        wide
      >
        {active && (
          <p className={styles.activeStrip} data-testid="language-active-strip">
            <strong>● Đang bật: {active.name} v{active.version}</strong> Ngôn ngữ hình quy định cách vẽ. Màu, bối cảnh, chất liệu
            của từng video đổi theo chủ đề: vũ trụ thì tối, lòng quả sung thì sáng hồng.
          </p>
        )}
        <div className={listStyles.layout}>
          <div className={listStyles.controls}>
            <Card title="Các ngôn ngữ">
              <ul className={listStyles.list} data-testid="language-list">
                {languages.map((l) => (
                  <li key={l.id}>
                    <button
                      type="button"
                      className={`${listStyles.row} ${l.id === selected?.id ? listStyles.rowSelected : ""}`}
                      onClick={() => {
                        setSelectedId(l.id);
                        setStatus(null);
                      }}
                      data-testid={`language-row-${l.id}`}
                    >
                      <span className={listStyles.rowName}>{l.name}</span>
                      <span className={listStyles.badges}>
                        <span className={listStyles.badge}>{l.is_system ? "Hệ thống" : "Của bạn"}</span>
                        {l.is_active && <span className={listStyles.badge}>Đang bật</span>}
                      </span>
                      <span className={listStyles.versionRow}>
                        {l.is_system ? l.note : "bản sao"} · v{l.version}
                      </span>
                    </button>
                  </li>
                ))}
                {languages.length === 0 && !loading && <li className={glass.cardHint}>Chưa có ngôn ngữ nào.</li>}
              </ul>
              {status && (
                <p className={`${glass.cardHint} ${glass.mtXs}`} role="status" data-testid="language-status">
                  {status}
                </p>
              )}
            </Card>
          </div>

          {selected && (
            <Card title={selected.name} hint={`kit ${selected.kit_key} · v${selected.version}`}>
              <div className={styles.samples} data-testid="language-samples">
                {(["fig", "space"] as const).map((scene) => (
                  <figure key={scene} className={styles.sampleSet}>
                    <div className={styles.samplePair}>
                      {samplesOf(samples, scene).map((s) => (
                        <img
                          key={s.frame}
                          className={s.frame === "wide" ? styles.wide : styles.tall}
                          src={languageSampleUrl(s)}
                          alt={`${SCENE_LABEL[scene]} (${s.frame === "wide" ? "16:9" : "9:16"})`}
                          loading="lazy"
                        />
                      ))}
                    </div>
                    <figcaption className={glass.cardHint}>{SCENE_LABEL[scene]}</figcaption>
                  </figure>
                ))}
              </div>
              {samples.length === 0 && (
                <p className={glass.cardHint} data-testid="language-no-samples">
                  Chưa có ảnh mẫu. Bấm “Dựng lại ảnh mẫu” (rendering phải đang chạy).
                </p>
              )}
              <div className={styles.row}>
                <Button variant="run" onClick={() => void rebuildSamples()} disabled={rebuilding || busy} data-testid="language-rebuild-samples">
                  {rebuilding ? "Đang dựng lại ảnh mẫu…" : "Dựng lại ảnh mẫu"}
                </Button>
                <span className={glass.cardHint}>Cùng một lối vẽ cho hai bảng màu khác hẳn · 16:9 và 9:16</span>
              </div>

              {readOnly ? (
                <p className={styles.readOnly}>
                  <strong>Chỉ đọc</strong> Ngôn ngữ hệ thống không sửa được. Bấm “Sao chép để sửa” để có bản của bạn.
                </p>
              ) : null}
              <p className={glass.cardHint}>{selected.description}</p>
              {selected.note && <p className={glass.cardHint}>{selected.note} · ghi chú cho bạn, không đưa vào prompt</p>}

              {RULES.map((r) => (
                <FormField key={r.key} label={`${r.label} · ${r.hint}`} className={glass.mtSm}>
                  <TextArea
                    data-testid={`language-rule-${r.key}`}
                    value={selected.rules[r.key]}
                    readOnly
                    rows={6}
                    aria-label={r.label}
                  />
                </FormField>
              ))}

              <p className={glass.cardHint} data-testid="language-activate-note">
                {selected.is_active
                  ? "Ngôn ngữ này đang bật."
                  : `Bật thì ${active?.name ?? "ngôn ngữ đang bật"} tắt. Video mới ghim ngôn ngữ này; video đã tạo giữ ngôn ngữ đã ghim.`}
              </p>
              <div className={styles.actions}>
                <Button variant="ghost" disabled={busy} onClick={() => void run(async () => (await copyVisualLanguage(selected.id)).id, "Đã sao chép thành một ngôn ngữ của bạn.")} data-testid="language-copy">
                  Sao chép để sửa
                </Button>
                {!readOnly && (
                  <Button variant="ghost" disabled={busy} onClick={() => setEditing(draftOf(selected))} data-testid="language-edit">
                    Sửa
                  </Button>
                )}
                {!readOnly && (
                  <Button
                    variant="danger"
                    disabled={busy || selected.is_active}
                    title={selected.is_active ? "Đang bật nên chưa xoá được." : undefined}
                    onClick={() => setConfirmDelete(true)}
                    data-testid="language-delete"
                  >
                    Xoá
                  </Button>
                )}
                <Button
                  variant="primary"
                  disabled={busy || selected.is_active}
                  onClick={() => void run(() => activateVisualLanguage(selected.id), `Đã bật ${selected.name}.`)}
                  data-testid="language-activate"
                >
                  {selected.is_active ? "Đang bật" : "Bật ngôn ngữ này"}
                </Button>
              </div>
            </Card>
          )}
        </div>
      </AppShell>

      <Dialog
        open={editing !== null}
        onClose={() => setEditing(null)}
        hasInput
        size="lg"
        title={selected ? `Sửa ${selected.name}` : "Sửa ngôn ngữ hình"}
        testId="language-edit-dialog"
        footer={
          <>
            <Button variant="ghost" onClick={() => setEditing(null)}>
              Huỷ
            </Button>
            <Button variant="primary" onClick={() => void saveEdit()} disabled={draftIncomplete} data-testid="language-save">
              Lưu phiên bản mới
            </Button>
          </>
        }
      >
        {editing && (
          <>
            <FormField label="Tên">
              <TextInput data-testid="language-name-input" value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} />
            </FormField>
            <FormField label="Ghi chú tham khảo (chỉ cho bạn xem)" className={glass.mtSm}>
              <TextInput
                value={editing.note}
                onChange={(e) => setEditing({ ...editing, note: e.target.value })}
                placeholder="ví dụ: kênh hoặc bộ phim bạn tham khảo"
              />
            </FormField>
            <FormField label="Mô tả" className={glass.mtSm}>
              <TextArea value={editing.description} onChange={(e) => setEditing({ ...editing, description: e.target.value })} rows={2} />
            </FormField>
            {RULES.map((r) => (
              <FormField key={r.key} label={`${r.label} · ${r.hint}`} className={glass.mtSm}>
                <TextArea
                  data-testid={`language-edit-${r.key}`}
                  value={editing.rules[r.key]}
                  onChange={(e) => setRule(r.key, e.target.value)}
                  rows={8}
                />
              </FormField>
            ))}
            <p className={glass.cardHint}>Lưu tạo một phiên bản mới; video đã ghim ngôn ngữ này giữ phiên bản cũ.</p>
          </>
        )}
      </Dialog>

      <ConfirmModal
        isOpen={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        onConfirm={() => {
          if (selected) {
            void run(async () => {
              await deleteVisualLanguage(selected.id);
              setSelectedId(null);
            }, "Đã xoá.");
          }
        }}
        title={selected ? `Xoá ${selected.name}?` : "Xoá ngôn ngữ?"}
        message="Xoá ngôn ngữ và mọi phiên bản của nó. Không khôi phục lại được. Chỉ xoá được khi không có video nào đang ghim."
        confirmLabel="Xoá ngôn ngữ"
        isDangerous
      />
    </div>
  );
}
