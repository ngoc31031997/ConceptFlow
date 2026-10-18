import { useEffect, useRef, useState } from "react";
import type { FactTextField, ResearchFact } from "../../api/client";
import { reachesLaterSteps, useUnitImpact } from "../../hooks/useUnitImpact";
import { Button, Dialog, FormField, TextArea, TextInput } from "../ui";
import { ImpactList } from "../units/ImpactList";
import styles from "./research.module.css";

interface FactEditModalProps {
  projectId: string;
  /** The fact being edited, or null while the dialog is closed. */
  fact: ResearchFact | null;
  onClose: () => void;
  /** Saves the fields whose text changed; a thrown error stays in the dialog. */
  onSave: (id: string, fields: Partial<Record<FactTextField, string>>) => Promise<void>;
}

/**
 * "Sửa" for one fact: its wording, link and quote as a form, then what saving
 * sends back in later steps (docs/ux-ui-design-rules.md, rules 4 and 5). Saving
 * a new link or quote checks the page again. The save button is orange when
 * later work uses the fact.
 */
export function FactEditModal({ projectId, fact, onClose, onSave }: FactEditModalProps) {
  const open = fact !== null;
  const impact = useUnitImpact(projectId, { kind: "fact", key: fact?.id ?? "" }, open);
  const [claim, setClaim] = useState("");
  const [url, setUrl] = useState("");
  const [quote, setQuote] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // The text to start from is read when the dialog opens, never again while it
  // is open: a poll redrawing the page must not overwrite what is typed.
  const latest = useRef(fact);
  useEffect(() => {
    latest.current = fact;
  });
  useEffect(() => {
    if (!open || !latest.current) return;
    setClaim(latest.current.claimVi);
    setUrl(latest.current.url);
    setQuote(latest.current.quote);
    setError(null);
  }, [open, fact?.id]);

  async function save() {
    const current = latest.current;
    if (!current) return;
    const changed: Partial<Record<FactTextField, string>> = {};
    if (claim.trim() !== current.claimVi) changed.claim_vi = claim;
    if (url.trim() !== current.url) changed.url = url;
    if (quote.trim() !== current.quote) changed.quote = quote;
    if (Object.keys(changed).length === 0) {
      onClose();
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onSave(current.id, changed);
      onClose();
    } catch (err) {
      setError(err instanceof Error && err.message ? err.message : "Không lưu được bản sửa. Thử lại.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={fact ? `Sửa dữ kiện ${fact.id}` : ""}
      hasInput
      size="lg"
      testId="fact-edit"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid="fact-edit-cancel">
            Huỷ
          </Button>
          <Button
            variant={reachesLaterSteps(impact) ? "impact" : "primary"}
            onClick={() => void save()}
            disabled={saving || impact.items === null || claim.trim() === ""}
            data-testid="fact-edit-save"
          >
            {saving ? "Đang lưu…" : "Lưu và kiểm lại link"}
          </Button>
        </>
      }
    >
      <FormField label="Dữ kiện">
        <TextArea value={claim} onChange={(e) => setClaim(e.target.value)} rows={2} data-testid="fact-edit-claim" />
      </FormField>
      <div className={styles.grid2}>
        <FormField label="Link nguồn">
          <TextInput value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://" data-testid="fact-edit-url" />
        </FormField>
        <FormField label="Đoạn trích nguyên văn">
          <TextArea
            className={styles.mono}
            value={quote}
            onChange={(e) => setQuote(e.target.value)}
            rows={3}
            data-testid="fact-edit-quote"
          />
        </FormField>
      </div>
      <ImpactList head="Lưu bản sửa thì phần phụ thuộc sẽ làm lại:" impact={impact} testId="fact-edit-impact" />
      {error && (
        <p className={styles.error} role="alert" data-testid="fact-edit-error">
          {error}
        </p>
      )}
    </Dialog>
  );
}
