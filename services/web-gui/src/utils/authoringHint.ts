/** What the bottom bar of an authoring step (3, 4, 6) says before "Tiếp tục". */
export interface AuthoringHintInput {
  /** AI mode is on and the server has a provider. */
  aiMode: boolean;
  /** The step's result box is still empty. */
  isEmpty: boolean;
  /** The step's result, lower case: "dàn ý", "storyboard", "code Manim". */
  what: string;
  /** Said once the result box has content. */
  ready: string;
}

/**
 * The hint for the bottom bar of an authoring step, in the words of the mode
 * the Creator is in: run the AI or write it yourself (AI mode), or paste what
 * an outside AI wrote (manual mode).
 */
export function authoringHint({ aiMode, isEmpty, what, ready }: AuthoringHintInput): string {
  if (!isEmpty) return ready;
  return aiMode
    ? `Bấm Chạy bằng AI, hoặc tự viết ${what} vào ô bên dưới.`
    : `Dán ${what} từ AI để tiếp tục.`;
}
