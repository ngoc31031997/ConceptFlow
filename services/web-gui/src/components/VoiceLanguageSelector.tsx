interface VoiceLanguageSelectorProps {
  value: "vi" | "en";
  onChange: (lang: "vi" | "en") => void;
}

export function VoiceLanguageSelector({ value, onChange }: VoiceLanguageSelectorProps) {
  return (
    <select
      data-testid="new-project-voice-language-select"
      value={value}
      onChange={(event) => onChange(event.target.value as "vi" | "en")}
    >
      <option value="vi">Tiếng Việt</option>
      <option value="en">English</option>
    </select>
  );
}
