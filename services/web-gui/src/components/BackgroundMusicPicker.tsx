import { useState } from "react";

interface BackgroundMusicPickerProps {
  value: string | null;
  onChange: (path: string | null) => void;
}

export function BackgroundMusicPicker({ value, onChange }: BackgroundMusicPickerProps) {
  const [enabled, setEnabled] = useState(value !== null);

  function handleToggle(checked: boolean) {
    setEnabled(checked);
    if (!checked) onChange(null);
  }

  return (
    <div>
      <label>
        <input
          type="checkbox"
          checked={enabled}
          onChange={(event) => handleToggle(event.target.checked)}
        />
        Thêm nhạc nền
      </label>
      {enabled && (
        <input
          type="text"
          data-testid="new-project-music-input"
          value={value ?? ""}
          onChange={(event) => onChange(event.target.value)}
          placeholder="Đường dẫn file nhạc nền"
        />
      )}
    </div>
  );
}
