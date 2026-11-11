import { useEffect, useRef, useState } from "react";
import { Button, FormField, TextArea, TextInput } from "../ui";
import { getPromptTemplate } from "../../api/client";
import { useCopy } from "../../hooks/useCopy";
import glass from "../../styles/glass.module.css";
import { fillYoutubeUrl, isWebUrl } from "./youtubePrompt";

const PROMPT_ERROR = "Không lấy được prompt từ thư viện prompt";

/**
 * "Link YouTube" field with a button that copies the prompt asking an outside
 * AI to describe the video's style. The prompt is the active one of the
 * `reference_style_youtube` role in the prompt library (editable in Cài đặt ›
 * Prompt AI), fetched on click with the link filled in. The button stays locked
 * until the link is a valid http or https address. If the prompt cannot be
 * fetched, an error shows and nothing is copied. If the browser refuses the
 * clipboard, the prompt is shown selected so it can be copied by hand.
 */
export function YoutubePromptCopy() {
  const [url, setUrl] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [manualText, setManualText] = useState<string | null>(null);
  const { state, copy } = useCopy();
  const manualRef = useRef<HTMLTextAreaElement>(null);
  const valid = isWebUrl(url);

  useEffect(() => {
    if (manualText !== null) manualRef.current?.select();
  }, [manualText]);

  const onCopy = async () => {
    setBusy(true);
    setError(null);
    setManualText(null);
    let prompt: string;
    try {
      prompt = fillYoutubeUrl(
        (await getPromptTemplate("reference_style_youtube")).template_text,
        url.trim(),
      );
    } catch {
      setError(PROMPT_ERROR);
      setBusy(false);
      return;
    }
    if (!(await copy(prompt))) setManualText(prompt);
    setBusy(false);
  };

  return (
    <div data-testid="youtube-prompt">
      <FormField label="Link YouTube">
        <TextInput
          value={url}
          onChange={(e) => {
            setUrl(e.target.value);
            setError(null);
            setManualText(null);
          }}
          placeholder="https://www.youtube.com/watch?v=…"
          data-testid="youtube-url-input"
        />
      </FormField>
      <div className={glass.mtXs}>
        <Button
          variant="ghost"
          onClick={onCopy}
          disabled={!valid || busy}
          data-testid="youtube-copy"
        >
          {state === "copied" ? "Đã chép" : busy ? "Đang lấy prompt..." : "Copy prompt YouTube"}
        </Button>
        {!valid && (
          <span className={glass.cardHint} data-testid="youtube-url-hint">
            {" "}
            {url.trim() === ""
              ? "Nhập link YouTube để copy prompt."
              : "Link cần bắt đầu bằng http hoặc https."}
          </span>
        )}
      </div>
      {error && (
        <p className={glass.cardHint} role="alert" data-testid="youtube-error">
          {error}
        </p>
      )}
      {manualText !== null && (
        <FormField label="Trình duyệt không cho chép tự động. Prompt đã được chọn sẵn, bấm Ctrl+C (hoặc Cmd+C)">
          <TextArea
            ref={manualRef}
            readOnly
            value={manualText}
            rows={6}
            data-testid="youtube-manual"
          />
        </FormField>
      )}
    </div>
  );
}
