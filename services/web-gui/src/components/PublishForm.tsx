import { useState, type FormEvent } from "react";
import type { PublishMetadata } from "../types";

interface PublishFormProps {
  onSubmit: (metadata: PublishMetadata) => void;
  isSubmitting: boolean;
}

const TITLE_MAX_LENGTH = 100;

export function PublishForm({ onSubmit, isSubmitting }: PublishFormProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState("");
  const [visibility, setVisibility] = useState<PublishMetadata["visibility"]>("private");

  const isTitleValid = title.trim().length > 0 && title.length <= TITLE_MAX_LENGTH;

  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (!isTitleValid) return;
    onSubmit({
      youtube_title: title,
      description: description || undefined,
      tags: tags ? tags.split(",").map((tag) => tag.trim()) : undefined,
      visibility,
    });
  }

  return (
    <form onSubmit={handleSubmit}>
      <label>
        Tiêu đề
        <input
          type="text"
          data-testid="publish-form-title-input"
          value={title}
          maxLength={TITLE_MAX_LENGTH}
          onChange={(event) => setTitle(event.target.value)}
        />
        <span>
          {title.length}/{TITLE_MAX_LENGTH}
        </span>
      </label>
      <label>
        Mô tả
        <textarea
          data-testid="publish-form-description-textarea"
          value={description}
          onChange={(event) => setDescription(event.target.value)}
        />
      </label>
      <label>
        Tags (phân cách bởi dấu phẩy)
        <input
          type="text"
          data-testid="publish-form-tags-input"
          value={tags}
          onChange={(event) => setTags(event.target.value)}
        />
      </label>
      <label>
        Chế độ hiển thị
        <select
          data-testid="publish-form-visibility-select"
          value={visibility}
          onChange={(event) => setVisibility(event.target.value as PublishMetadata["visibility"])}
        >
          <option value="private">Riêng tư</option>
          <option value="unlisted">Không công khai</option>
          <option value="public">Công khai</option>
        </select>
      </label>
      <button
        type="submit"
        data-testid="publish-form-submit-button"
        disabled={!isTitleValid || isSubmitting}
      >
        Đăng lên YouTube
      </button>
    </form>
  );
}
