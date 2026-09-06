// Mirrors NARRATION_RE (script-processing) and AUTO_WAIT_RE (rendering) exactly,
// so mismatches are caught client-side before a render is even started.
const NARRATION_RE = /^\s*#\s*NARRATION:\s*"(.*)"\s*$/;
const AUTO_WAIT_RE = /self\.wait\(\s*AUTO\s*\)/g;
const SCENE_CLASS_RE = /^class\s+(\w+)\s*\([^)]*Scene[^)]*\)\s*:/;

export interface ScriptValidation {
  narrationCount: number;
  autoWaitCount: number;
  hasSceneClass: boolean;
  isValid: boolean;
  message: string | null;
}

export function validateScript(script: string): ScriptValidation {
  const lines = script.split("\n");
  const narrationCount = lines.filter((line) => NARRATION_RE.test(line) && line.match(NARRATION_RE)?.[1]?.trim()).length;
  const autoWaitCount = (script.match(AUTO_WAIT_RE) ?? []).length;
  const hasSceneClass = lines.some((line) => SCENE_CLASS_RE.test(line.trim()));

  if (script.trim().length === 0) {
    return { narrationCount, autoWaitCount, hasSceneClass, isValid: true, message: null };
  }

  if (!hasSceneClass) {
    return {
      narrationCount,
      autoWaitCount,
      hasSceneClass,
      isValid: false,
      message: 'Chưa tìm thấy class Scene (cần dạng class TenScene(Scene):).',
    };
  }

  if (narrationCount === 0) {
    return {
      narrationCount,
      autoWaitCount,
      hasSceneClass,
      isValid: false,
      message: 'Chưa có marker # NARRATION: "..." nào trong script.',
    };
  }

  if (narrationCount !== autoWaitCount) {
    return {
      narrationCount,
      autoWaitCount,
      hasSceneClass,
      isValid: false,
      message: `Lệch số lượng: ${narrationCount} marker # NARRATION nhưng ${autoWaitCount} lệnh self.wait(AUTO). Mỗi NARRATION cần đúng một self.wait(AUTO) ngay sau nó — render sẽ thất bại nếu không sửa.`,
    };
  }

  return { narrationCount, autoWaitCount, hasSceneClass, isValid: true, message: null };
}
