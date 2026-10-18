import { useEffect, useState } from "react";
import { FACT_TYPES, type FactInput, type FactType, type ResearchQuestion } from "../../api/client";
import { Button, Dialog, FormField, Select, TextArea, TextInput } from "../ui";
import { FACT_TYPE_NAMES } from "./factTypes";
import styles from "./research.module.css";

interface FactAddModalProps {
  open: boolean;
  onClose: () => void;
  questions: ResearchQuestion[];
  /** Adds the fact; a thrown error stays in the dialog. */
  onAdd: (input: FactInput) => Promise<void>;
}

/**
 * "Thêm dữ kiện tay": a fact the Creator writes, with the question it answers
 * and, if they have one, its source. A link with a quote is checked like a
 * pasted fact (docs/ux-ui-design-rules.md, rule 4).
 */
export function FactAddModal({ open, onClose, questions, onAdd }: FactAddModalProps) {
  const [claim, setClaim] = useState("");
  const [question, setQuestion] = useState("");
  const [type, setType] = useState<FactType>("fact");
  const [sourceTitle, setSourceTitle] = useState("");
  const [url, setUrl] = useState("");
  const [quote, setQuote] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setClaim("");
    setQuestion(questions[0]?.id ?? "");
    setType("fact");
    setSourceTitle("");
    setUrl("");
    setQuote("");
    setError(null);
    // Read when the dialog opens; the questions behind it may change while it is open.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function add() {
    setSaving(true);
    setError(null);
    try {
      await onAdd({ question, claimVi: claim, type, sourceTitle, url, quote });
      onClose();
    } catch (err) {
      setError(err instanceof Error && err.message ? err.message : "Không thêm được dữ kiện. Thử lại.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Thêm dữ kiện tay"
      hasInput
      size="lg"
      testId="fact-add"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid="fact-add-cancel">
            Huỷ
          </Button>
          <Button variant="primary" onClick={() => void add()} disabled={saving || claim.trim() === "" || question === ""} data-testid="fact-add-save">
            {saving ? "Đang thêm…" : "Thêm và kiểm link"}
          </Button>
        </>
      }
    >
      <FormField label="Dữ kiện">
        <TextArea value={claim} onChange={(e) => setClaim(e.target.value)} rows={2} placeholder="Một câu, một ý" data-testid="fact-add-claim" />
      </FormField>
      <div className={styles.grid3}>
        <FormField label="Câu hỏi">
          <Select value={question} onChange={(e) => setQuestion(e.target.value)} data-testid="fact-add-question">
            {questions.map((q) => (
              <option key={q.id} value={q.id}>
                {q.id}
              </option>
            ))}
          </Select>
        </FormField>
        <FormField label="Loại">
          <Select value={type} onChange={(e) => setType(e.target.value as FactType)} data-testid="fact-add-type">
            {FACT_TYPES.map((t) => (
              <option key={t} value={t}>
                {FACT_TYPE_NAMES[t]}
              </option>
            ))}
          </Select>
        </FormField>
        <FormField label="Tên nguồn">
          <TextInput
            value={sourceTitle}
            onChange={(e) => setSourceTitle(e.target.value)}
            placeholder="ví dụ: Encyclopaedia Britannica"
            data-testid="fact-add-source"
          />
        </FormField>
      </div>
      <div className={styles.grid2}>
        <FormField label="Link">
          <TextInput value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://" data-testid="fact-add-url" />
        </FormField>
        <FormField label="Đoạn trích nguyên văn">
          <TextArea className={styles.mono} value={quote} onChange={(e) => setQuote(e.target.value)} rows={3} data-testid="fact-add-quote" />
        </FormField>
      </div>
      <p className={styles.hint}>Có link và đoạn trích thì hệ thống tải trang và kiểm đoạn trích như dữ kiện dán vào.</p>
      {error && (
        <p className={styles.error} role="alert" data-testid="fact-add-error">
          {error}
        </p>
      )}
    </Dialog>
  );
}
