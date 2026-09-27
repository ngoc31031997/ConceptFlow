import { useEffect, useState } from "react";
import { getIllustrationStyle, type CodeDiagnostic } from "../api/client";
import { describeWarning, parseRuleNames, type StyleWarning } from "../utils/styleRules";

let cached: Promise<Record<string, string>> | null = null;

/**
 * CR-045 — tên luật style theo mã (S3 → "Bo tròn"), tải một lần cho cả trang.
 * Lỗi mạng: trả về rỗng (cảnh báo vẫn hiện, chỉ thiếu tên luật) và lần sau thử lại.
 */
export function useStyleRuleNames(): Record<string, string> {
  const [names, setNames] = useState<Record<string, string>>({});
  useEffect(() => {
    if (!cached) {
      cached = getIllustrationStyle()
        .then((s) => parseRuleNames(s.rules ?? ""))
        .catch(() => {
          cached = null;
          return {};
        });
    }
    let cancelled = false;
    void cached.then((n) => {
      if (!cancelled) setNames(n);
    });
    return () => {
      cancelled = true;
    };
  }, []);
  return names;
}

/** Chỉ cho test: quên bản đã tải. */
export function resetStyleRuleNamesCache(): void {
  cached = null;
}

/** Cảnh báo đã kèm tên luật (tải một lần từ file luật của server). */
export function useStyleWarnings(warnings: CodeDiagnostic[]): StyleWarning[] {
  const names = useStyleRuleNames();
  return warnings.map((w) => describeWarning(w, names));
}
