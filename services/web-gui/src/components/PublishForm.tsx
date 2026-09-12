import { useState, type FormEvent } from "react";
import { suggestPublishMetadata, ApiError } from "../api/client";
import type { PublishMetadata } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./PublishForm.module.css";

interface PublishFormProps {
  projectId: string;
  onSubmit: (metadata: PublishMetadata) => void;
  isSubmitting: boolean;
}

const TITLE_MAX_LENGTH = 100;

const VISIBILITY_OPTIONS: { value: PublishMetadata["visibility"]; label: string; icon: JSX.Element }[] = [
  {
    value: "private",
    label: "Riêng tư",
    icon: (
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <rect x="5" y="11" width="14" height="9" rx="2" />
        <path d="M8 11V7a4 4 0 018 0v4" />
      </svg>
    ),
  },
  {
    value: "unlisted",
    label: "Không công khai",
    icon: (
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M10 13a5 5 0 007 0l3-3a5 5 0 00-7-7l-1 1" />
        <path d="M14 11a5 5 0 00-7 0l-3 3a5 5 0 007 7l1-1" />
      </svg>
    ),
  },
  {
    value: "public",
    label: "Công khai",
    icon: (
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="12" r="9" />
        <path d="M3 12h18M12 3a14 14 0 010 18M12 3a14 14 0 000 18" />
      </svg>
    ),
  },
];

export function PublishForm({ projectId, onSubmit, isSubmitting }: PublishFormProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState("");
  const [visibility, setVisibility] = useState<PublishMetadata["visibility"]>("private");
  const [publishAt, setPublishAt] = useState("");
  const [isSuggesting, setIsSuggesting] = useState(false);
  const [suggestError, setSuggestError] = useState<string | null>(null);

  // Clear suggest error when user manually edits any field
  const clearSuggestErrorOnEdit = () => {
    if (suggestError) setSuggestError(null);
  };

  async function handleSuggestAI() {
    setSuggestError(null); // Clear previous error
    setIsSuggesting(true);
    try {
      const suggestion = await suggestPublishMetadata(projectId);
      // Normalised rather than trusted: this is a model-generated payload
      // crossing a service boundary, and a missing tags array used to throw
      // "Cannot read properties of null" instead of showing an error.
      setTitle((suggestion.title ?? "").slice(0, TITLE_MAX_LENGTH));
      setDescription(suggestion.description ?? "");
      setTags(Array.isArray(suggestion.tags) ? suggestion.tags.join(", ") : "");
    } catch (err) {
      setSuggestError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setIsSuggesting(false);
    }
  }

  const isTitleValid = title.trim().length > 0 && title.length <= TITLE_MAX_LENGTH;
  const tagList = tags
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
  const isPublishAtValid = !publishAt || new Date(publishAt).getTime() > Date.now();

  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    // A form submits on Enter too, where the button's disabled state is no guard.
    if (isSubmitting) return;
    if (!isTitleValid) return;
    if (visibility === "private" && publishAt && !isPublishAtValid) return;
    onSubmit({
      youtube_title: title,
      description: description || undefined,
      tags: tagList.length > 0 ? tagList : undefined,
      visibility,
      publish_at:
        visibility === "private" && publishAt ? new Date(publishAt).toISOString() : undefined,
    });
  }

  return (
    <form onSubmit={handleSubmit} className={`${glass.card} ${styles.form}`}>
      <div className={styles.ctaRow}>
        <button
          type="button"
          data-testid="publish-form-suggest-ai-button"
          className={styles.suggestBtn}
          disabled={isSuggesting}
          onClick={handleSuggestAI}
        >
          {isSuggesting ? "Đang tạo gợi ý..." : "✨ Gợi ý AI (tiêu đề, mô tả, tags)"}
        </button>
      </div>
      {suggestError && (
        <p role="alert" className={glass.helperText}>
          {suggestError}
        </p>
      )}

      <div className={styles.field}>
        <div className={styles.labelRow}>
          <label className={styles.label} htmlFor="publish-title">
            Tiêu đề
          </label>
          <span className={styles.charCount}>
            {title.length}/{TITLE_MAX_LENGTH}
          </span>
        </div>
        <input
          id="publish-title"
          type="text"
          data-testid="publish-form-title-input"
          className={glass.textInput}
          value={title}
          maxLength={TITLE_MAX_LENGTH}
          onChange={(event) => {
            setTitle(event.target.value);
            clearSuggestErrorOnEdit();
          }}
        />
      </div>

      <div className={styles.field}>
        <label className={styles.label} htmlFor="publish-description">
          Mô tả
        </label>
        <textarea
          id="publish-description"
          data-testid="publish-form-description-textarea"
          className={`${glass.textArea} ${styles.field}`}
          style={{ minHeight: 72, font: "inherit" }}
          value={description}
          onChange={(event) => {
            setDescription(event.target.value);
            clearSuggestErrorOnEdit();
          }}
        />
      </div>

      <div className={styles.field}>
        <label className={styles.label} htmlFor="publish-tags">
          Tags (phân cách bởi dấu phẩy)
        </label>
        <input
          id="publish-tags"
          type="text"
          data-testid="publish-form-tags-input"
          className={glass.textInput}
          value={tags}
          onChange={(event) => {
            setTags(event.target.value);
            clearSuggestErrorOnEdit();
          }}
        />
        {tagList.length > 0 && (
          <div className={styles.tagRow}>
            {tagList.map((tag) => (
              <span key={tag} className={styles.tagChip}>
                {tag}
              </span>
            ))}
          </div>
        )}
      </div>

      <div className={styles.field}>
        <div className={styles.label}>Chế độ hiển thị</div>
        <div className={styles.visibilitySwitch} data-testid="publish-form-visibility-select">
          {VISIBILITY_OPTIONS.map((option) => (
            <button
              key={option.value}
              type="button"
              className={`${styles.visOpt} ${visibility === option.value ? styles.active : ""}`}
              aria-pressed={visibility === option.value}
              onClick={() => setVisibility(option.value)}
            >
              {option.icon}
              {option.label}
            </button>
          ))}
        </div>
      </div>

      {visibility === "private" && (
        <div className={styles.field}>
          <label className={styles.label} htmlFor="publish-at">
            Tự động công khai lúc (tùy chọn)
          </label>
          <input
            id="publish-at"
            type="datetime-local"
            data-testid="publish-form-publish-at-input"
            className={glass.textInput}
            value={publishAt}
            onChange={(event) => setPublishAt(event.target.value)}
          />
          {publishAt && !isPublishAtValid && (
            <p role="alert" className={glass.helperText}>
              Thời gian phải ở tương lai
            </p>
          )}
          {publishAt && isPublishAtValid && (
            <p className={glass.helperText}>
              Video sẽ ở chế độ riêng tư, sau đó YouTube tự chuyển sang công khai đúng giờ đã chọn.
            </p>
          )}
        </div>
      )}

      <div className={styles.ctaRow}>
        <button
          type="submit"
          data-testid="publish-form-submit-button"
          className={glass.btnPrimary}
          disabled={!isTitleValid || isSubmitting || (visibility === "private" && !!publishAt && !isPublishAtValid)}
        >
          {isSubmitting ? "Đang gửi yêu cầu..." : "Đăng lên YouTube"}
        </button>
      </div>
    </form>
  );
}
